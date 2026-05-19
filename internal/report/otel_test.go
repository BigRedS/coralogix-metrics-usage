package report

import (
	"strings"
	"testing"
)

func TestOTELUnusedProcessorsYAML(t *testing.T) {
	used := []UsedSeries{
		{
			Series: `m_partial{a="1",b="2"}`,
			Labels: map[string]string{"__name__": "m_partial", "a": "1", "b": "2"},
		},
		{
			Series: `m_both{a="1"}`,
			Labels: map[string]string{"__name__": "m_both", "a": "1"},
		},
	}
	unused := []UnusedSeries{
		{
			Series: `m_partial{a="1",b="2",c="3"}`,
			Labels: map[string]string{"__name__": "m_partial", "a": "1", "b": "2", "c": "3"},
		},
		{
			Series: `m_unused_only{x="9"}`,
			Labels: map[string]string{"__name__": "m_unused_only", "x": "9"},
		},
		{
			Series: `m_both{a="2"}`,
			Labels: map[string]string{"__name__": "m_both", "a": "2"},
		},
	}

	y := string(OTELUnusedProcessorsYAML(used, unused))

	if !strings.Contains(y, "metrics_usage_drop_unused_metrics") {
		t.Fatal("expected drop processor")
	}
	if !strings.Contains(y, `"m_unused_only"`) {
		t.Fatal("expected wholly-unused metric in drop list")
	}
	if strings.Contains(y, "          - "+yamlQuotedScalar("m_partial")) {
		t.Fatal("m_partial is partially used and must not appear in filter metric_names")
	}
	if !strings.Contains(y, `metric.name == "m_partial"`) || !strings.Contains(y, `delete_key(attributes, "c")`) {
		t.Fatal("expected strip of unused-only label c on m_partial")
	}
	if strings.Contains(y, `delete_key(attributes, "a")`) || strings.Contains(y, `delete_key(attributes, "b")`) {
		t.Fatal("did not expect stripping labels shared with used series")
	}
	if strings.Contains(y, `delete_key(attributes, "__name__")`) {
		t.Fatal("did not expect stripping __name__")
	}
}

func TestOTELUnusedProcessorsYAML_emptyStripFullDrop(t *testing.T) {
	used := []UsedSeries{
		{Labels: map[string]string{"__name__": "keep", "x": "1"}},
	}
	unused := []UnusedSeries{
		{Labels: map[string]string{"__name__": "drop_me", "y": "2"}},
	}
	y := OTELUnusedProcessorsYAML(used, unused)
	if !strings.Contains(string(y), `"drop_me"`) {
		t.Fatal("expected drop_me in filter list")
	}
	if strings.Contains(string(y), `metric.name == "keep"`) {
		t.Fatal("no strip rules expected for keep (unused-only keys absent)")
	}
}

func TestOTELUnusedProcessorsYAML_emptyCatalog(t *testing.T) {
	y := string(OTELUnusedProcessorsYAML(nil, nil))
	if !strings.Contains(y, "processors: {}\n") {
		t.Fatalf("expected empty processors map: %q", y)
	}
}

func TestOTELUnusedProcessorsYAML_escapeMetricName(t *testing.T) {
	used := []UsedSeries{}
	unused := []UnusedSeries{
		{Labels: map[string]string{"__name__": `weird"name`, "k": "v"}},
	}
	y := string(OTELUnusedProcessorsYAML(used, unused))
	wantYAML := `          - "` + `weird\"name` + `"`
	if !strings.Contains(y, wantYAML) {
		t.Fatalf("expected YAML-escaped metric name %q in:\n%s", wantYAML, y)
	}
}
