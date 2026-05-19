package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
)

// UnusedByMetricRow rolls up unused catalog series to a single row per __name__.
// Use this when per-series unit_usage is too granular or noisy to interpret.
type UnusedByMetricRow struct {
	MetricName string `json:"metric_name"`

	UnusedSeriesCount            int `json:"unused_series_count"`
	UnusedSeriesWithBillingCount int `json:"unused_series_with_billing_count"`

	UnitUsageSum   float64 `json:"unit_usage_sum"`
	BytesVolumeSum uint64  `json:"bytes_volume_sum"`
	SampleCountSum uint64  `json:"sample_count_sum"`
	CardinalitySum uint64  `json:"cardinality_sum"`   // sum of CX-reported per-series cardinality (heuristic, not distinct-cardinality across series)
	DaysInRangeMax int     `json:"days_in_range_max"` // max among attributed rows (0 if none)
}

// AggregateUnusedByMetric groups flat unused-by-cost rows by metric name.
// Sums include only rows with billing_present (same zero semantics as series rows without CX match).
func AggregateUnusedByMetric(rows []UnusedByCostRow) []UnusedByMetricRow {
	type bucket struct {
		metric string
		n      int
		nb     int
		uu     float64
		bv     uint64
		sc     uint64
		card   uint64
		days   int
	}
	m := make(map[string]*bucket)
	for _, row := range rows {
		mn := row.MetricName
		if mn == "" {
			mn = "(unknown)"
		}
		b := m[mn]
		if b == nil {
			b = &bucket{metric: mn}
			m[mn] = b
		}
		b.n++
		if row.BillingPresent {
			b.nb++
			b.uu += row.UnitUsage
			b.bv += row.BytesVolume
			b.sc += row.SampleCount
			b.card += row.Cardinality
			if row.DaysInRange > b.days {
				b.days = row.DaysInRange
			}
		}
	}
	out := make([]UnusedByMetricRow, 0, len(m))
	for _, b := range m {
		out = append(out, UnusedByMetricRow{
			MetricName:                   b.metric,
			UnusedSeriesCount:            b.n,
			UnusedSeriesWithBillingCount: b.nb,
			UnitUsageSum:                 b.uu,
			BytesVolumeSum:               b.bv,
			SampleCountSum:               b.sc,
			CardinalitySum:               b.card,
			DaysInRangeMax:               b.days,
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

// WriteUnusedByMetricCSV writes metric rollup rows.
func WriteUnusedByMetricCSV(path string, rows []UnusedByMetricRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := []string{
		"unit_usage_sum",
		"unused_series_count",
		"unused_series_with_billing_count",
		"bytes_volume_sum",
		"sample_count_sum",
		"cardinality_sum",
		"days_in_range_max",
		"metric_name",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		rec := []string{
			strconv.FormatFloat(row.UnitUsageSum, 'g', -1, 64),
			strconv.Itoa(row.UnusedSeriesCount),
			strconv.Itoa(row.UnusedSeriesWithBillingCount),
			strconv.FormatUint(row.BytesVolumeSum, 10),
			strconv.FormatUint(row.SampleCountSum, 10),
			strconv.FormatUint(row.CardinalitySum, 10),
			strconv.Itoa(row.DaysInRangeMax),
			row.MetricName,
		}
		if err := w.Write(rec); err != nil {
			return fmt.Errorf("write csv row for %s: %w", row.MetricName, err)
		}
	}
	w.Flush()
	return w.Error()
}
