package metricusage

import "testing"

func TestVariationKeyFromCatalogLabels_matchesVariationKey(t *testing.T) {
	// A catalog series whose label keys exactly match a variation's label_names set
	// must produce the same key; that's the equality EnrichCatalog relies on.
	catalog := map[string]string{
		"__name__": "http_requests_total",
		"job":      "api",
		"instance": "1",
	}
	variation := VariationKey("http_requests_total", []string{"__name__", "job", "instance"})
	got := variationKeyFromCatalogLabels(catalog)
	if got != variation {
		t.Fatalf("catalog key %q != variation key %q", got, variation)
	}
}

func TestVariationKeyFromCatalogLabels_differingValuesShareKey(t *testing.T) {
	// Values differ but label-name sets match → same variation key.
	a := variationKeyFromCatalogLabels(map[string]string{"__name__": "m", "job": "api", "env": "prod"})
	b := variationKeyFromCatalogLabels(map[string]string{"__name__": "m", "job": "batch", "env": "dev"})
	if a != b {
		t.Fatalf("expected same key, got %q and %q", a, b)
	}
}

func TestVariationKeyFromCatalogLabels_differingLabelSetsDiffer(t *testing.T) {
	a := variationKeyFromCatalogLabels(map[string]string{"__name__": "m", "job": "api"})
	b := variationKeyFromCatalogLabels(map[string]string{"__name__": "m", "job": "api", "env": "prod"})
	if a == b {
		t.Fatalf("expected different keys, both = %q", a)
	}
}
