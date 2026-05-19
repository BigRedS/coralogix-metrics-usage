package metricusage

import "testing"

func TestLabelsSubset(t *testing.T) {
	if !labelsSubset(map[string]string{"a": "1"}, map[string]string{"a": "1", "b": "2"}) {
		t.Fatal("expected subset")
	}
	if labelsSubset(map[string]string{"a": "2"}, map[string]string{"a": "1"}) {
		t.Fatal("expected not subset")
	}
}

func TestParseCanonicalSeries_bracedPrometheusSelector(t *testing.T) {
	key := `{__name__="http_requests_total",job="api",instance="1"}`
	lbls, err := parseCanonicalSeries(key)
	if err != nil {
		t.Fatal(err)
	}
	if lbls["__name__"] != "http_requests_total" || lbls["job"] != "api" || lbls["instance"] != "1" {
		t.Fatalf("labels: %#v", lbls)
	}
}
