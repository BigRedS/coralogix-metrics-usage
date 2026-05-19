package promqlextract_test

import (
	"testing"

	"github.com/avi/coralogix-metrics-usage/internal/promqlextract"
)

func TestExtractFromPromQL(t *testing.T) {
	sel := promqlextract.ExtractFromPromQL(`sum(rate(http_requests_total{job="api"}[5m])) by (instance)`)
	if len(sel) != 1 {
		t.Fatalf("got %d selectors, want 1", len(sel))
	}
	if promqlextract.MetricName(sel[0].Selector) != "http_requests_total" {
		t.Fatalf("metric name = %q", promqlextract.MetricName(sel[0].Selector))
	}
}

func TestMatchesSeries(t *testing.T) {
	sel := promqlextract.ExtractFromPromQL(`http_requests_total{job="api",env=~"prod.*"}`)
	if len(sel) != 1 {
		t.Fatal("expected one selector")
	}
	vs := sel[0].Selector
	if !promqlextract.MatchesSeries(vs, map[string]string{
		"__name__": "http_requests_total",
		"job":      "api",
		"env":      "prod-eu",
	}) {
		t.Fatal("expected match")
	}
	if promqlextract.MatchesSeries(vs, map[string]string{
		"__name__": "http_requests_total",
		"job":      "other",
		"env":      "prod-eu",
	}) {
		t.Fatal("expected no match")
	}
}
