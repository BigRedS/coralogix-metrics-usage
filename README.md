# coralogix-metrics-usage

**coralogix-metrics-usage** is a CLI tool that reads Coralogix **dashboards**, **alert definitions (v3)**, and **SLOs**, pulls live **Prometheus-style series** from `https://api.<region-host>/metrics`, and correlates **PromQL** references with **billing-oriented CX usage** from the **Metrics Usage** API.

---

## API key permissions

Use one **personal** or **team** API key for `--key`. All HTTP traffic sends:

```http
Authorization: Bearer <your-api-key>
```

Billing additionally opens **gRPC** to `api.<your-region-host>:443` with the same `Bearer` token.

### What each part needs

| Feature | APIs touched | Typical preset / permission |
|--------|----------------|-----------------------------|
| **Dashboard catalog & definitions** | `GET …/mgmt/openapi/5/dashboards/dashboards/v1/catalog/list`, `GET …/dashboards/dashboards/v1/{id}` | API preset **`Dashboards`** — includes **`TEAM-DASHBOARDS:READ`** (and related dashboard permissions per [permissions list](https://coralogix.com/docs/user-guides/aaa/access-control/permissions/permissions-list/)) |
| **Alert definitions** | `GET …/mgmt/openapi/5/alerts/alerts/v3` (paginated) | API preset **`Alerts`** — covers alert-definition reads such as **`ALERTS:READCONFIG`**, **`METRICS.ALERTS:READCONFIG`**, **`LOGS.ALERTS:READCONFIG`**, **`SPANS.ALERTS:READCONFIG`** (see [Alert definitions API](https://docs.coralogix.com/api-reference/v5/alert-definitions-service/overview)) |
| **SLOs** | `GET …/mgmt/openapi/5/slo/slos/v1` | API preset **`SLO`** — includes **`SLO:READCONFIG`** and **`SLO-MGMT.ALERTS:READCONFIG`** (SLO management / SLO-based alerts) |
| **Live metric catalog** | `GET https://api.<host>/metrics/api/v1/label/__name__/values`, `GET …/api/v1/series` | **`DataQuerying`** (or **`Query Metrics`** / `metrics.data-api#high:ReadData`) — see [Metrics API](https://coralogix.com/docs/user-guides/data-query/metrics-api/) |
| **`unit_usage` / billing metrics** (unless `--skip-billing`) | gRPC `com.coralogix.metrics.metric_usages.UsageService.GetVariationUsagesByMetric` | **`DataAnalytics`** includes **`METRICS.DATA-ANALYTICS#HIGH:READ`** — see [Metric usage API](https://coralogix.com/docs/developer-portal/apis/data-management/metrics-usage-api/) |

Coralogix preset identifiers can change; start from **[API keys](https://coralogix.com/docs/user-guides/account-management/api-keys/api-keys/)** and merge presets until every stage succeeds.

Minimal merge usually resembles:

- **DataQuerying** — Prometheus `/metrics` queries (`METRICS.DATA-API#HIGH:READDATA`, etc.)  
- **DataAnalytics** — CX **`unit_usage`** / Metrics Usage (`METRICS.DATA-ANALYTICS#HIGH:READ`, etc.)  
- **Dashboards** — dashboard catalog + definitions (`TEAM-DASHBOARDS:READ`, …)  
- **Alerts** — alert definitions v3 listing (`ALERTS:READCONFIG`, `METRICS.ALERTS:READCONFIG`, …)  
- **SLO** — SLO list (`SLO:READCONFIG`, `SLO-MGMT.ALERTS:READCONFIG`, …)

**Read-only caveat:** The **`Dashboards`**, **`Alerts`**, and **`SLO`** API presets also bundle **update/manage** permissions (see the same column in the [permissions list](https://coralogix.com/docs/user-guides/aaa/access-control/permissions/permissions-list/)). For a **least-privilege read-only** key, create the key with **Advanced** permissions and grant only the `READ` / `READCONFIG` entries above (plus **DataQuerying** / **DataAnalytics** as needed), instead of attaching the full presets. Example permission strings to combine (drop any your definitions do not need): **`TEAM-DASHBOARDS:READ`**; **`ALERTS:READCONFIG`**, **`METRICS.ALERTS:READCONFIG`**, **`LOGS.ALERTS:READCONFIG`**, **`SPANS.ALERTS:READCONFIG`**; **`SLO:READCONFIG`**, **`SLO-MGMT.ALERTS:READCONFIG`**; **`METRICS.DATA-API#HIGH:READDATA`**; **`METRICS.DATA-ANALYTICS#HIGH:READ`** (billing).

---

## Creating a dedicated API key (curl example)

Creating keys uses management OpenAPI:

```http
POST https://api.<region-host>/mgmt/openapi/5/aaa/api-keys/v3
```

See **[Create API Key](https://docs.coralogix.com/api-reference/v5/api-keys-service/create-api-key)**.

Replace placeholders (`EXISTING_ADMIN_KEY`, `eu2`, your owner block). Preset names must match what your tenant exposes (`presets` is an array of preset identifiers). Besides **DataQuerying** and **DataAnalytics**, this tool needs read access to **dashboards**, **alert definitions**, and **SLOs** — in practice the matching API presets are **`Dashboards`**, **`Alerts`**, and **`SLO`**:

```bash
API_HOST="${CX_REGION_HOST:-api.eu2.coralogix.com}"

curl -sS -X POST "https://${API_HOST}/mgmt/openapi/5/aaa/api-keys/v3" \
  -H "Authorization: Bearer ${EXISTING_ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "coralogix-metrics-usage-scanner",
    "keyPermissions": {
      "presets": [
        "DataQuerying",
        "DataAnalytics",
        "Dashboards",
        "Alerts",
        "SLO"
      ]
    },
    "owner": { "userId": "YOUR_CORALOGIX_USER_ID" }
  }'
```

Adjust `owner` if your account uses team or organisation ownership ([OpenAPI `Owner` schema](https://docs.coralogix.com/api-reference/v5/api-keys-service/create-api-key)).

If you must avoid the extra write capabilities bundled into **`Dashboards`** / **`Alerts`** / **`SLO`**, omit those presets and use `"permissions": ["TEAM-DASHBOARDS:READ", "SLO:READCONFIG", …]` with only the granular **read** strings from the [permissions list](https://coralogix.com/docs/user-guides/aaa/access-control/permissions/permissions-list/) (filter by API preset column).

The response includes the **secret key value once** (`value`); store it safely.

**Requirements:** the caller must already use an API key allowed to **create** keys (API Keys admin / equivalent).

---

## UI alternative

If you prefer not to script creation: **Settings → Users and Teams → API Keys → Create**, and attach **DataQuerying**, **DataAnalytics**, **Dashboards**, **Alerts**, and **SLO** (or the granular read-only permissions from the note above), then pass that key to `--key`.

---

## Build & run

```bash
go build -o bin/coralogix-metrics-usage ./cmd/coralogix-metrics-usage/
./bin/coralogix-metrics-usage --region eu2 --key "$CX_API_KEY" --output-dir ./out
```

See `--help` for flags (`--usage-lookback-days`, `--usage-billing-calendar-months`, `--skip-billing`, `--skip-dashboards`, `--skip-alerts`, `--skip-slo`, etc.).

### Browser UI (lab)

Self-contained server under **`webui/`**: choose region, paste API key, run scan, download outputs (same files as `--output-dir`). The UI validates **personal** keys (`cxup_` prefix and fixed length) and shows a masked preview **before** submit; the CLI accepts any Bearer token format. **Localhost / trusted use only** — see **`webui/README.md`**.

```bash
go run ./webui -listen localhost:8765
```

Coralogix **platform metrics** whose `__name__` starts with **`cx_`** are omitted from the catalog, billing enrichment, and outputs (they are not customer telemetry). The dropped count is in **`meta.coralogix_internal_metric_names_skipped`**.

If your API key lacks **Dashboards**, **Alerts**, or **SLO** access, pass **`--skip-dashboards`**, **`--skip-alerts`**, and/or **`--skip-slo`** so the scan skips those HTTP calls. Correlation (used vs unused, OTEL drops/strips) then considers only PromQL from the sources that ran; skipped modes append **`warnings`** in `metric_usage_summary.json`. Skipping **all three** makes every catalog series appear unused.

When a Coralogix **HTTP** request fails (transport error, non-2xx, unreadable body, or unexpected Prometheus JSON while HTTP was 2xx), the error text includes a **replication block**: full URL (with query string), request headers with **`Authorization: Bearer <redacted>`** (and **`Cookie`** redacted if present), response status/headers/body (body capped at 64KiB for non-2xx), plus a multi-line **`curl`** example where **`Authorization`** is the **last** `-H` line so you can paste into a terminal and replace the token on the final line.

---

## Output files

| File | Description |
|------|-------------|
| `metric_usage_summary.json` | Full report: correlation results, warnings, `meta` (including `usage_lookback_days`, `series_with_billing_data`, `unused_series_with_billing`, `coralogix_internal_metric_names_skipped`). Unused entries here use optional nested `"billing": { "unit_usage": … }` when matched. |
| `metric_usage_unused_series.json` | Unused series only (alphabetically by full selector string). |
| **`metric_usage_unused_by_cost.json`** | Same unused series, **sorted by cost**, with **flat** billing fields on every row (easier for `jq` / tooling). |
| **`metric_usage_unused_by_cost.csv`** | Same data as the JSON cost file, as a spreadsheet-friendly CSV. |
| **`metric_usage_unused_by_metric.json`** | **Rollup**: one row per unused **`__name__`**, sorted by **`unit_usage_sum`** — sums billing fields over unused series that have CX data (see below). |
| **`metric_usage_unused_by_metric.csv`** | Same metric rollup as spreadsheet-friendly CSV. |
| **`metric_usage_otel_processors.yaml`** | Fragment for **otelcol-contrib**: drops metrics that are unused end-to-end, and strips label keys that appear only on unused series for partially-used metrics (see below). |

### CX billing window (**`unit_usage`** sample span)

Billing is skipped when **`--skip-billing`** is set, or when both **`--usage-lookback-days`** is **`0`** and **`--usage-billing-calendar-months`** is **`0`**.

- **`--usage-lookback-days`** (default **`7`**) — rolling window of inclusive **UTC calendar days** ending at **today’s UTC date** (midnight-aligned). Example: on **2026-05-15**, **`7`** means **2026-05-09** through **2026-05-15** inclusive.
- **`--usage-billing-calendar-months N`** (**`N > 0`**) — last **`N`** **complete UTC calendar months**, excluding the current partial month: from the **first day** of month **`current − N`** through the **last day** of the **previous** month. If both rolling days and **`N`** are non-zero, **calendar months win** for the API request (CLI still accepts both flags).

The exact dates sent to Metrics Usage are recorded in **`metric_usage_summary.json`** → **`meta.usage_billing_utc_start_date`**, **`meta.usage_billing_utc_end_date`**, **`meta.usage_billing_calendar_months`**, and **`meta.usage_lookback_days`** (inclusive day count for whichever window ran).

### Why **`unit_usage`** often looks tiny

CX **`unit_usage`** is whatever unit Coralogix exposes in Metrics Usage (not necessarily dollars). Values can be **fractional** by design. This tool also **attributes** one CX variation row to multiple Prometheus catalog series when labels only **subset**-match: usage is **split evenly** (**`billing_split_n`**), so each series sees **`1/N`** of that row — fine-grained and legitimately small.

For prioritization, prefer **`metric_usage_unused_by_metric.*`**, which adds **`unit_usage_sum`** per metric so you see aggregate attributed usage instead of thousands of thin per-series rows.

**Rollup semantics:** **`unit_usage_sum`** / **`bytes_volume_sum`** / **`sample_count_sum`** are sums of **per-series attributed** values (only rows with **`billing_present`**). They approximate total unused footprint for that metric name but are **not** a substitute for CX invoices. **`cardinality_sum`** sums CX-reported per-series cardinality figures (a heuristic, not “distinct labels across series”). **`unused_series_with_billing_count`** vs **`unused_series_count`** shows how many unused series lacked a billing match.

### OpenTelemetry Collector fragment (`metric_usage_otel_processors.yaml`)

Requires **otelcol-contrib** with the **filter** and **transform** processors. Merge the generated `processors:` block into your Collector config, then add both processor IDs to your metrics pipeline (recommended order: drop unused metrics first, then strip labels):

```yaml
service:
  pipelines:
    metrics:
      receivers: [prometheus, ...]
      processors:
        - memory_limiter
        - metrics_usage_drop_unused_metrics
        - metrics_usage_strip_unused_only_labels
        - batch
      exporters: [...]
```

Semantics:

- **`metrics_usage_drop_unused_metrics`** — `filter` with strict metric-name excludes: every series for that `__name__` was **unused** (no **scanned** dashboard, alert, or SLO matched any series for that metric in the window). If you used `--skip-dashboards`, `--skip-alerts`, or `--skip-slo`, “used” excludes references from skipped sources.
- **`metrics_usage_strip_unused_only_labels`** — `transform` OTTL on **datapoint** context: for metrics that still have **some** used series, emit `delete_key(attributes, "<key>") where metric.name == "<metric>"` for each label key that appears on at least one **unused** catalog series but **never** on any **used** catalog series for that metric. Prometheus scrape labels are assumed to live on datapoint attributes (typical **prometheusreceiver** layout).

If nothing qualifies, the file contains `processors: {}`. Review before production: stripping dimensions changes metric identity; the catalog may be incomplete if metrics hit `--series-limit-per-metric`.

### `metric_usage_unused_by_cost.json` — fields

Each element is one unused time series:

- **`metric_name`** — value of the `__name__` label.
- **`series`** — full canonical selector string (metric + sorted labels), same key used elsewhere in the report.
- **`labels`** — label map as JSON object.
- **`billing_present`** — `true` if CX Metrics Usage returned a matching row for this series over the configured billing window (**`--usage-lookback-days`** or **`--usage-billing-calendar-months`**).
- **`unit_usage`**, **`bytes_volume`**, **`sample_count`**, **`cardinality`**, **`days_in_range`** — from Coralogix when `billing_present` is true; otherwise numeric zeros.
- **`billing_split_n`** — if greater than `1`, one billing **variation** row applied to several catalog series, so usage was **divided evenly** across them (approximate per-series attribution).

### Example: top 20 unused by CX units (`jq`)

```bash
jq 'map(select(.billing_present)) | sort_by(-.unit_usage) | .[:20]' out/metric_usage_unused_by_cost.json
```

### Example: CSV in a terminal

```bash
column -t -s, < out/metric_usage_unused_by_cost.csv | less -S
```

Or open `metric_usage_unused_by_cost.csv` in Excel / Google Sheets. Columns match the JSON row shape (`labels_json` is one escaped JSON column).

### If `billing_present` is always false

- Confirm billing ran: `metric_usage_summary.json` → `meta.series_with_billing_data` should be > 0 when `--skip-billing` was not used and at least one of **`--usage-lookback-days`** or **`--usage-billing-calendar-months`** is positive.
- Ensure the API key includes **DataAnalytics** / metrics analytics read (see permissions table above).
- Billing dates are **UTC calendar days**; brand-new metrics may show zeros until usage appears.

