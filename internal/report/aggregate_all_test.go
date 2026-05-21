package report

import (
	"math"
	"testing"
)

func TestAggregateAllByMetric(t *testing.T) {
	used := []UsedSeries{
		{Labels: map[string]string{"__name__": "m_a"}, Billing: &SeriesBilling{UnitUsage: 1.0}},
		{Labels: map[string]string{"__name__": "m_b"}},
	}
	unused := []UnusedSeries{
		{Labels: map[string]string{"__name__": "m_a"}, Billing: &SeriesBilling{UnitUsage: 0.5}},
		{Labels: map[string]string{"__name__": "m_a"}},
		{Labels: map[string]string{"__name__": "m_c"}, Billing: &SeriesBilling{UnitUsage: 2.5}},
	}

	got := AggregateAllByMetric(used, unused)
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3", len(got))
	}

	// Sorted by unit_usage_sum desc: m_c (2.5), m_a (1.5), m_b (0).
	if got[0].MetricName != "m_c" || math.Abs(got[0].UnitUsageSum-2.5) > 1e-9 || got[0].SeriesCount != 1 {
		t.Fatalf("row 0: %+v", got[0])
	}
	if got[1].MetricName != "m_a" || math.Abs(got[1].UnitUsageSum-1.5) > 1e-9 || got[1].SeriesCount != 3 {
		t.Fatalf("row 1: %+v", got[1])
	}
	if got[2].MetricName != "m_b" || got[2].UnitUsageSum != 0 || got[2].SeriesCount != 1 {
		t.Fatalf("row 2: %+v", got[2])
	}
}

func TestAggregateAllByMetric_MissingName(t *testing.T) {
	got := AggregateAllByMetric(nil, []UnusedSeries{{Labels: map[string]string{}}})
	if len(got) != 1 || got[0].MetricName != "(unknown)" || got[0].SeriesCount != 1 {
		t.Fatalf("unexpected: %+v", got)
	}
}
