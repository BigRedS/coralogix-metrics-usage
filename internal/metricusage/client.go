// Package metricusage queries Coralogix Metrics Usage API (gRPC) for billing units per variation.
package metricusage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	metriccommon "github.com/avi/coralogix-metrics-usage/internal/gen/metriccommon"
	metricusages "github.com/avi/coralogix-metrics-usage/internal/gen/metricusages"
	"github.com/avi/coralogix-metrics-usage/internal/promqlextract"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// UnitsUsage is CX billing usage for a time-series (variation) over a date range.
type UnitsUsage struct {
	UnitUsage   float64 `json:"unit_usage"`
	BytesVolume uint64  `json:"bytes_volume"`
	Cardinality uint64  `json:"cardinality"`
	SampleCount uint64  `json:"sample_count"`
	DaysInRange int     `json:"days_in_range"`
}

// Client calls UsageService on api.<region>:443.
type Client struct {
	APIHost string
	svc     metricusages.UsageServiceClient
	conn    *grpc.ClientConn
}

// NewClient dials the regional Coralogix API host (e.g. api.eu2.coralogix.com).
func NewClient(apiHost, apiKey string) (*Client, error) {
	target := apiHost + ":443"
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		grpc.WithPerRPCCredentials(bearerAuth{token: apiKey}),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", target, err)
	}
	return &Client{
		APIHost: apiHost,
		svc:     metricusages.NewUsageServiceClient(conn),
		conn:    conn,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type bearerAuth struct {
	token string
}

func (b bearerAuth) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + b.token}, nil
}

func (b bearerAuth) RequireTransportSecurity() bool { return true }

// variationPageSize is the per-request cap on variations sent back by Coralogix
// (CommonRequestFields.length). Pagination is per-day: each VariationUsageDay carries
// its own OutputSetStats.matched_count, and start_offset/length slice each day independently.
// 1000 keeps responses well under the default 4MiB gRPC frame even with ~7 days × wide labels.
const variationPageSize uint32 = 1000

// FetchVariationUnits loads per-variation usage for one metric across inclusive UTC calendar dates.
// Pages through CommonRequestFields.start_offset / length until offset >= max(matched_count)
// across all days in the window.
func (c *Client) FetchVariationUnits(ctx context.Context, metricName string, startDay, endDay time.Time) (map[string]UnitsUsage, error) {
	startDay = startDay.UTC().Truncate(24 * time.Hour)
	endDay = endDay.UTC().Truncate(24 * time.Hour)
	if endDay.Before(startDay) {
		return nil, fmt.Errorf("usage end date before start date")
	}

	out := make(map[string]UnitsUsage)
	var offset uint32
	for {
		req := &metricusages.GetVariationUsagesByMetricRequest{
			Common: &metriccommon.CommonRequestFields{
				StartDate:   toProtoDate(startDay),
				EndDate:     toProtoDate(endDay),
				StartOffset: offset,
				Length:      variationPageSize,
			},
			MetricName: metricName,
		}

		resp, err := c.svc.GetVariationUsagesByMetric(ctx, req)
		if err != nil {
			return nil, err
		}

		var maxMatched uint32
		for _, day := range resp.GetDailyUsages() {
			for _, v := range day.GetVariationUsages() {
				key, err := SeriesKeyFromVariation(metricName, v.GetLabelNames())
				if err != nil {
					continue
				}
				agg := out[key]
				if u := v.GetUsage(); u != nil {
					agg.UnitUsage += float64(u.GetUnitUsage())
					agg.BytesVolume += u.GetBytesVolume()
					agg.Cardinality += u.GetCardinality()
				}
				agg.SampleCount += v.GetSampleCount()
				agg.DaysInRange++
				out[key] = agg
			}
			if mc := day.GetOutputSetStats().GetMatchedCount(); mc > maxMatched {
				maxMatched = mc
			}
		}

		offset += variationPageSize
		if offset >= maxMatched {
			return out, nil
		}
	}
}

// SeriesKeyFromVariation builds the canonical series key for a Coralogix variation.
func SeriesKeyFromVariation(metricName string, labelNames []string) (string, error) {
	labels, err := LabelsFromVariation(metricName, labelNames)
	if err != nil {
		return "", err
	}
	return promqlextract.CanonicalSeries(labels), nil
}

// LabelsFromVariation parses Coralogix variation label_names (e.g. "job=api") into a label map.
func LabelsFromVariation(metricName string, labelNames []string) (map[string]string, error) {
	labels := map[string]string{"__name__": metricName}
	for _, s := range labelNames {
		k, v, ok := splitLabelPair(s)
		if !ok {
			return nil, fmt.Errorf("invalid label pair %q", s)
		}
		labels[k] = v
	}
	return labels, nil
}

func splitLabelPair(s string) (key, value string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

func labelsSubset(subset, full map[string]string) bool {
	for k, v := range subset {
		if full[k] != v {
			return false
		}
	}
	return true
}

func splitUsage(u UnitsUsage, n int) UnitsUsage {
	if n <= 1 {
		return u
	}
	return UnitsUsage{
		UnitUsage:   u.UnitUsage / float64(n),
		BytesVolume: u.BytesVolume / uint64(n),
		Cardinality: u.Cardinality / uint64(n),
		SampleCount: u.SampleCount / uint64(n),
		DaysInRange: u.DaysInRange,
	}
}

// EnrichCatalog maps variation-level CX billing onto Prometheus catalog series keys.
// When Coralogix reports a partial label set for a variation, all catalog series whose labels
// are a superset share that row's usage (values are divided evenly).
func (c *Client) EnrichCatalog(
	ctx context.Context,
	catalogSeries map[string]map[string]string,
	metricNames []string,
	startDay, endDay time.Time,
	workers int,
	onProgress func(done, total int, metric string),
) (map[string]UnitsUsage, map[string]int, error) {
	if workers <= 0 {
		workers = 4
	}

	seriesKeysByMetric := make(map[string][]string)
	for sk, lbls := range catalogSeries {
		mn := lbls["__name__"]
		if mn == "" {
			continue
		}
		seriesKeysByMetric[mn] = append(seriesKeysByMetric[mn], sk)
	}

	result := make(map[string]UnitsUsage)
	splitCount := make(map[string]int)
	var mu sync.Mutex

	total := len(metricNames)
	var done atomic.Int32

	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, workers)

	for _, mname := range metricNames {
		mname := mname
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-sem }()

			skList := seriesKeysByMetric[mname]
			if len(skList) == 0 {
				if onProgress != nil {
					onProgress(int(done.Add(1)), total, mname)
				}
				return nil
			}

			byVar, err := c.FetchVariationUnits(gctx, mname, startDay, endDay)
			if err != nil {
				return fmt.Errorf("%q: %w", mname, err)
			}

			type varEntry struct {
				key    string
				labels map[string]string
				usage  UnitsUsage
			}
			entries := make([]varEntry, 0, len(byVar))
			for k, u := range byVar {
				lbls, err := parseCanonicalSeries(k)
				if err != nil {
					continue
				}
				entries = append(entries, varEntry{key: k, labels: lbls, usage: u})
			}

			localResult := make(map[string]UnitsUsage)
			localSplit := make(map[string]int)
			exact := make(map[string]bool)

			for _, sk := range skList {
				if u, ok := byVar[sk]; ok {
					localResult[sk] = u
					localSplit[sk] = 1
					exact[sk] = true
				}
			}

			subsetGroups := make(map[string][]string)
			for _, sk := range skList {
				if exact[sk] {
					continue
				}
				sl := catalogSeries[sk]
				var best *varEntry
				bestScore := -1
				var bestUsage float64
				for i := range entries {
					e := &entries[i]
					if !labelsSubset(e.labels, sl) {
						continue
					}
					score := len(e.labels)
					u := e.usage.UnitUsage
					if score > bestScore || (score == bestScore && u > bestUsage) {
						bestScore = score
						bestUsage = u
						best = e
					}
				}
				if best != nil {
					subsetGroups[best.key] = append(subsetGroups[best.key], sk)
				}
			}

			for vkey, group := range subsetGroups {
				u := byVar[vkey]
				n := len(group)
				if n == 0 {
					continue
				}
				part := splitUsage(u, n)
				for _, sk := range group {
					if exact[sk] {
						continue
					}
					localResult[sk] = part
					localSplit[sk] = n
				}
			}

			mu.Lock()
			for k, v := range localResult {
				result[k] = v
			}
			for k, v := range localSplit {
				splitCount[k] = v
			}
			mu.Unlock()

			if onProgress != nil {
				onProgress(int(done.Add(1)), total, mname)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return result, splitCount, nil
}

func parseCanonicalSeries(seriesKey string) (map[string]string, error) {
	if seriesKey == "" {
		return nil, fmt.Errorf("empty series key")
	}
	// Catalog keys are promqlextract.CanonicalSeries output: {__name__="m",label="v",...}
	// Parse them directly as PromQL vector selectors. Do not prefix with a fake metric name:
	// x{__name__="m",...} is invalid ("metric name must not be set twice") and breaks subset matching.
	sel := promqlextract.ExtractFromPromQL(seriesKey)
	if len(sel) == 0 {
		return nil, fmt.Errorf("cannot parse %q", seriesKey)
	}
	mname := promqlextract.MetricName(sel[0].Selector)
	if mname == "" {
		return nil, fmt.Errorf("no metric name in %q", seriesKey)
	}
	labels := map[string]string{"__name__": mname}
	for _, m := range sel[0].Selector.LabelMatchers {
		labels[m.Name] = m.Value
	}
	return labels, nil
}

func toProtoDate(t time.Time) *date.Date {
	t = t.UTC()
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}
