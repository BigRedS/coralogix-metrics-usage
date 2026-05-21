package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
)

// MetricRollupRow rolls up every catalog series for a single __name__,
// regardless of whether it was classified as used or unused.
type MetricRollupRow struct {
	MetricName   string  `json:"metric_name"`
	SeriesCount  int     `json:"series_count"`
	UnitUsageSum float64 `json:"unit_usage_sum"`
}

// AggregateAllByMetric groups every catalog series by __name__ and sums billing
// unit_usage across both used and unused series.
func AggregateAllByMetric(used []UsedSeries, unused []UnusedSeries) []MetricRollupRow {
	type bucket struct {
		n  int
		uu float64
	}
	m := make(map[string]*bucket)
	add := func(labels map[string]string, billing *SeriesBilling) {
		name := labels["__name__"]
		if name == "" {
			name = "(unknown)"
		}
		b := m[name]
		if b == nil {
			b = &bucket{}
			m[name] = b
		}
		b.n++
		if billing != nil {
			b.uu += billing.UnitUsage
		}
	}
	for _, s := range used {
		add(s.Labels, s.Billing)
	}
	for _, s := range unused {
		add(s.Labels, s.Billing)
	}
	out := make([]MetricRollupRow, 0, len(m))
	for name, b := range m {
		out = append(out, MetricRollupRow{
			MetricName:   name,
			SeriesCount:  b.n,
			UnitUsageSum: b.uu,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UnitUsageSum != out[j].UnitUsageSum {
			return out[i].UnitUsageSum > out[j].UnitUsageSum
		}
		return out[i].MetricName < out[j].MetricName
	})
	return out
}

// WriteAllByMetricCSV writes one row per metric name covering both used and unused series.
func WriteAllByMetricCSV(path string, rows []MetricRollupRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := []string{
		"unit_usage_sum",
		"series_count",
		"metric_name",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		rec := []string{
			strconv.FormatFloat(row.UnitUsageSum, 'g', -1, 64),
			strconv.Itoa(row.SeriesCount),
			row.MetricName,
		}
		if err := w.Write(rec); err != nil {
			return fmt.Errorf("write csv row for %s: %w", row.MetricName, err)
		}
	}
	w.Flush()
	return w.Error()
}
