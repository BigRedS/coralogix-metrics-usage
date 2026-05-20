# coralogix-metrics-usage Web UI (hack / lab)

Small **localhost** server: pick region, paste API key, run a scan, download the generated files.

```bash
cd /path/to/coralogix-metrics-usage
go run ./webui -listen localhost:8765
```

Open http://localhost:8765 — fill in **environment** and **API key**, then **Run scan**.


## Docker-compose

If you've docker-compose installed, then

    docker-compose up --build

should bring this up, listening on http://localhost:8765
