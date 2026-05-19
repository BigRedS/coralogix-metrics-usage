package promqlextract

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// VectorSelector is a deduplicated PromQL vector selector reference.
type VectorSelector struct {
	Canonical string
	Selector  *parser.VectorSelector
}

// ExtractFromPromQL parses a query and returns vector selectors (including inside matrix ranges).
func ExtractFromPromQL(text string) []VectorSelector {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	expr, err := parser.ParseExpr(text)
	if err != nil {
		return nil
	}
	seen := map[string]*parser.VectorSelector{}
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		switch n := node.(type) {
		case *parser.VectorSelector:
			canon := n.String()
			if _, ok := seen[canon]; !ok {
				seen[canon] = n
			}
		case *parser.MatrixSelector:
			vs, ok := n.VectorSelector.(*parser.VectorSelector)
			if !ok {
				return nil
			}
			canon := vs.String()
			if _, ok := seen[canon]; !ok {
				seen[canon] = vs
			}
		}
		return nil
	})
	out := make([]VectorSelector, 0, len(seen))
	for canon, vs := range seen {
		out = append(out, VectorSelector{Canonical: canon, Selector: vs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

// ExtractFromJSON walks every string in a JSON value and parses PromQL from likely candidates.
func ExtractFromJSON(raw json.RawMessage) []VectorSelector {
	seen := map[string]*parser.VectorSelector{}
	extractFromValue(raw, seen)
	out := make([]VectorSelector, 0, len(seen))
	for canon, vs := range seen {
		out = append(out, VectorSelector{Canonical: canon, Selector: vs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

func extractFromValue(raw json.RawMessage, seen map[string]*parser.VectorSelector) {
	if len(raw) == 0 {
		return
	}
	switch raw[0] {
	case '"':
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return
		}
		if len(s) < 3 {
			return
		}
		if !strings.ContainsAny(s, "{}()[]") {
			return
		}
		for _, vs := range ExtractFromPromQL(s) {
			seen[vs.Canonical] = vs.Selector
		}
	case '{':
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return
		}
		for _, v := range obj {
			extractFromValue(v, seen)
		}
	case '[':
		var arr []json.RawMessage
		if json.Unmarshal(raw, &arr) != nil {
			return
		}
		for _, v := range arr {
			extractFromValue(v, seen)
		}
	}
}

// MetricName returns the metric name from a vector selector.
func MetricName(vs *parser.VectorSelector) string {
	if vs.Name != "" {
		return vs.Name
	}
	for _, m := range vs.LabelMatchers {
		if m.Name == labels.MetricName && m.Type == labels.MatchEqual {
			return m.Value
		}
	}
	return ""
}

// MatchesSeries reports whether concrete label set satisfies the selector (Prometheus matcher semantics).
func MatchesSeries(vs *parser.VectorSelector, lbls map[string]string) bool {
	mname := MetricName(vs)
	if mname == "" {
		return false
	}
	if lbls["__name__"] != mname {
		return false
	}
	for _, m := range vs.LabelMatchers {
		val, ok := lbls[m.Name]
		if !ok {
			val = ""
		}
		if !m.Matches(val) {
			return false
		}
	}
	return true
}

// CanonicalSeries builds a stable selector string from a label map.
func CanonicalSeries(lbls map[string]string) string {
	pairs := make([]labels.Label, 0, len(lbls))
	for n, v := range lbls {
		pairs = append(pairs, labels.Label{Name: n, Value: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Name == labels.MetricName {
			return true
		}
		if pairs[j].Name == labels.MetricName {
			return false
		}
		return pairs[i].Name < pairs[j].Name
	})
	b := strings.Builder{}
	b.WriteByte('{')
	for i, l := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(l.Name)
		b.WriteString(`="`)
		b.WriteString(escapeLabelValue(l.Value))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

func escapeLabelValue(v string) string {
	return strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`)
}
