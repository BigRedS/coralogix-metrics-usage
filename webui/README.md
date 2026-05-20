# coralogix-metrics-usage Web UI (hack / lab)

Small **localhost** server: pick region, paste API key, run a scan, download the generated files.

```bash
cd /path/to/coralogix-metrics-usage
go run ./webui -listen localhost:8765
```

Open http://localhost:8765 — fill in **environment** and **API key**, then **Run scan**.


## Docker-compose

If you've docker-compose installed, then

    docker-compose up

should bring this up, listening on http://localhost:8765


## Docker (no compose)

The same prebuilt image is on GHCR (multi-arch, `linux/amd64` + `linux/arm64`):

    docker run --rm \
        --read-only \
        --tmpfs /tmp:rw,size=512m,mode=1777 \
        --cap-drop=ALL \
        --security-opt=no-new-privileges \
        -p 127.0.0.1:8765:8765 \
        ghcr.io/bigreds/coralogix-metrics-usage-webui:latest

Then open http://localhost:8765. The `--read-only` + `--tmpfs /tmp` pair keeps the
container rootfs immutable; job output lives in the tmpfs and dies with the container.
Drop the `127.0.0.1:` prefix to bind on all interfaces (don't do that on an untrusted
network — your API key is sent to this process).
