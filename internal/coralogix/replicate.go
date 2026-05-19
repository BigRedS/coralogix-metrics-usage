package coralogix

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// maxDiagnosticBody limits how much of an error response body we buffer for logs (successful reads still use io.ReadAll).
const maxDiagnosticBody = 65536

func redactedHeadersCopy(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	out := h.Clone()
	if out.Get("Authorization") != "" {
		out.Set("Authorization", "Bearer <redacted>")
	}
	if out.Get("Cookie") != "" {
		out.Set("Cookie", "<redacted>")
	}
	return out
}

func sortedHeaderLines(h http.Header) []string {
	if h == nil {
		return nil
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		for _, v := range h.Values(k) {
			lines = append(lines, k+": "+v)
		}
	}
	return lines
}

// curlHeaderLines returns request headers as curl -H lines with Authorization last (and Cookie
// just before it) so a pasted multi-line curl keeps the secret on the final editable line.
func curlHeaderLines(h http.Header) []string {
	rh := redactedHeadersCopy(h)
	if rh == nil {
		return nil
	}
	var keys []string
	for k := range rh {
		switch http.CanonicalHeaderKey(k) {
		case "Authorization", "Cookie":
			continue
		default:
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		for _, v := range rh.Values(k) {
			lines = append(lines, k+": "+v)
		}
	}
	for _, v := range rh.Values("Cookie") {
		lines = append(lines, "Cookie: "+v)
	}
	for _, v := range rh.Values("Authorization") {
		lines = append(lines, "Authorization: "+v)
	}
	return lines
}

// formatHTTPReplication prints enough detail to replay the request with curl (Authorization/Cookie redacted).
func formatHTTPReplication(req *http.Request, resp *http.Response, respBody []byte) string {
	var sb strings.Builder
	sb.WriteString("--- HTTP replication (secrets redacted; put your API key in Authorization) ---\n")
	sb.WriteString("request:\n")
	fmt.Fprintf(&sb, "  %s %s\n", req.Method, req.URL.String())
	for _, line := range sortedHeaderLines(redactedHeadersCopy(req.Header)) {
		fmt.Fprintf(&sb, "  %s\n", line)
	}
	sb.WriteString("  body: <empty>\n")

	sb.WriteString("\nresponse:\n")
	if resp == nil {
		sb.WriteString("  <none — error before HTTP response>\n")
	} else {
		fmt.Fprintf(&sb, "  %s\n", resp.Status)
		for _, line := range sortedHeaderLines(resp.Header) {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
		sb.WriteString("  body:\n")
		if len(respBody) == 0 {
			sb.WriteString("  <empty>\n")
		} else {
			sb.WriteString(string(respBody))
			sb.WriteByte('\n')
		}
	}

	sb.WriteString("\ncurl:\n")
	fmt.Fprintf(&sb, "curl -g -sS -X %s %q \\\n", req.Method, req.URL.String())
	hlines := curlHeaderLines(req.Header)
	for i, line := range hlines {
		sep := " \\\n"
		if i == len(hlines)-1 {
			sep = "\n"
		}
		fmt.Fprintf(&sb, "  -H %q%s", line, sep)
	}
	sb.WriteString("# Replace \"Bearer <redacted>\" with your real Bearer token.\n")
	return sb.String()
}

// readResponseBodyLimited reads at most limit+1 bytes to detect truncation.
func readResponseBodyLimited(resp *http.Response, limit int) ([]byte, bool, error) {
	b, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return nil, false, err
	}
	if len(b) > limit {
		return b[:limit], true, nil
	}
	return b, false, nil
}

// formatGETReplication builds the same replication block get() would use, without performing the request.
func (c *Client) formatGETReplication(rawURL string, query url.Values) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return formatHTTPReplication(req, nil, nil)
}
