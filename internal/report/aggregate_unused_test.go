package report

import (
	"math"
	"testing"
)

func TestAggregateUnusedByMetric(t *testing.T) {
	rows := []UnusedByCostRow{
		{MetricName: "m_a", BillingPresent: true, UnitUsage: 0.5, BytesVolume: 10, SampleCount: 100, Cardinality: 2, DaysInRange: 7},
		{MetricName: "m_a", BillingPresent: true, UnitUsage: 0.25, BytesVolume: 5, SampleCount: 50, Cardinality: 1, DaysInRange: 5},
		{MetricName: "m_a", BillingPresent: false},
		{MetricName: "m_b", BillingPresent: false},
	}
	got := AggregateUnusedByMetric(rows)
	if len(got) != 2 {
		t.Fatalf("got %d rows", len(got))
	}
	if got[0].MetricName != "m_a" || math.Abs(got[0].UnitUsageSum-0.75) > 1e-9 {
		t.Fatalf("first row: %+v", got[0])
	}
	if got[0].UnusedSeriesCount != 3 || got[0].UnusedSeriesWithBillingCount != 2 {
		t.Fatalf("counts: %+v", got[0])
	}
	if got[0].BytesVolumeSum != 15 || got[0].SampleCountSum != 150 || got[0].CardinalitySum != 3 || got[0].DaysInRangeMax != 7 {
		t.Fatalf("sums: %+v", got[0])
	}
	if got[1].MetricName != "m_b" || got[1].UnitUsageSum != 0 || got[1].UnusedSeriesCount != 1 {
		t.Fatalf("second row: %+v", got[1])
	}
}
