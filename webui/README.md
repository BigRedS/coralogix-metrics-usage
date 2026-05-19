# coralogix-metrics-usage Web UI (hack / lab)

Small **localhost** server: pick region, paste API key, run a scan, download the generated files.

```bash
cd /path/to/coralogix-metrics-usage
go run ./webui -listen localhost:8765
```

Open http://localhost:8765 — fill in **environment** and **API key**, then **Run scan**. When the job finishes you get download links plus a **read-only textarea** with `metric_usage_otel_processors.yaml` for quick copy-paste (same bytes as the download).

The page checks **personal** Coralogix API keys client-side: must start with **`cxup_`** and match the expected length (**35** characters as of current UI copy format). A **visible check** line under the field shows `cxup_`, bullet-masked middle, and the **last four characters** so you can confirm which key you pasted (the password field itself stays fully masked).

**Security**

- Intended for **trusted networks only**. The browser sends your API key to this process; do not expose the port to the internet without TLS and authentication.
- Keys are kept **in memory** only for the duration of the job; output lives under the system temp dir until the job expires (~1 hour) or the server exits.

**Requirements**

- Same API key presets as the CLI (`DataQuerying`, `DataAnalytics`, `Dashboards`, `Alerts`, `SLO`, …).
