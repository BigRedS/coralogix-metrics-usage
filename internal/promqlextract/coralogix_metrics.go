package promqlextract

import "strings"

// IsCoralogixInternalMetricName reports whether name looks like a Coralogix platform metric:
// the Prometheus "__name__" starts with "cx_". Those series are emitted by Coralogix itself,
// not by customer workloads; treating them as tenant metrics floods unused-series outputs.
func IsCoralogixInternalMetricName(name string) bool {
	return strings.HasPrefix(name, "cx_")
}
