package promqlextract

import "testing"

func TestIsCoralogixInternalMetricName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"cx_foo", true},
		{"cx_", true},
		{"customer_metric", false},
		{"cx", false},
		{"CX_upper", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCoralogixInternalMetricName(tt.name); got != tt.want {
				t.Fatalf("IsCoralogixInternalMetricName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
