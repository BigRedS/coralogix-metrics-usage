package metricusage_test

import (
	"testing"

	"github.com/BigRedS/coralogix-unused-metrics-finder/internal/metricusage"
)

func TestSeriesKeyFromVariation(t *testing.T) {
	key, err := metricusage.SeriesKeyFromVariation("http_requests_total", []string{"job=api", "env=prod"})
	if err != nil {
		t.Fatal(err)
	}
	if key == "" {
		t.Fatal("empty key")
	}
}

func TestLabelsFromVariationInvalid(t *testing.T) {
	_, err := metricusage.LabelsFromVariation("m", []string{"not-a-pair"})
	if err == nil {
		t.Fatal("expected error")
	}
}
