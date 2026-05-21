package metricusage_test

import (
	"testing"

	"github.com/BigRedS/coralogix-unused-metrics-finder/internal/metricusage"
)

func TestVariationKey_bareLabelNames(t *testing.T) {
	got := metricusage.VariationKey("http_requests_total", []string{"job", "env", "__name__"})
	want := "http_requests_total|__name__,env,job"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestVariationKey_keyEqualsValueIsAccepted(t *testing.T) {
	// CX docs show "method=GET" form; only the key contributes to variation identity.
	got := metricusage.VariationKey("m", []string{"method=GET", "status=200"})
	want := "m|__name__,method,status"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestVariationKey_orderIndependent(t *testing.T) {
	a := metricusage.VariationKey("m", []string{"a", "b", "c"})
	b := metricusage.VariationKey("m", []string{"c", "a", "b"})
	if a != b {
		t.Fatalf("keys differ by input order: %q vs %q", a, b)
	}
}
