# gpx-stats

A single, self-contained Go binary that computes statistics about a GPX file —
from the command line or through a small embedded web UI. **No database, no
network at runtime, nothing stored.**

## Features

- **CLI**: pass a GPX path, get raw statistics as text (or `--json`), plus
  optional ASCII charts in the terminal.
- **Web UI**: `serve` starts a local page where you upload a GPX file and see the
  same statistics rendered with inline SVG charts. The file is processed in
  memory and is never saved.
- **MVP statistics**: total distance (km), ascending elevation (m), total /
  moving / pause time, number of pauses, average and moving speed (km/h), and
  per-kilometer splits (time + speed).
- **Safe by construction**: GPX is parsed with the standard library
  `encoding/xml`, which ignores DTDs and external entities, so XXE and
  entity-expansion ("billion laughs") attacks are neutralised. Input is bounded
  by a maximum size and track-point count.

## Build

```sh
go build -o gpx-stats ./cmd/gpx-stats
# or: task build
```

The binary embeds all web assets (Bulma CSS, HTML templates); copy it anywhere
and run it offline.

## CLI usage

```sh
gpx-stats path/to/activity.gpx            # human-readable statistics
gpx-stats --json path/to/activity.gpx     # machine-readable JSON
gpx-stats --charts path/to/activity.gpx   # add ASCII elevation & speed charts

# Tune pause detection (defaults: 1 km/h sustained for 10s)
gpx-stats --pause-speed 1.5 --pause-duration 15s path/to/activity.gpx
```

Errors are written to stderr with a non-zero exit code (1 runtime, 2 usage).

## Web UI usage

```sh
gpx-stats serve --addr :8080
# open http://localhost:8080
```

Upload a GPX file to see the statistics and charts. Optional form fields override
the pause thresholds. Nothing is persisted.

## Configuration

| Flag | Default | Applies to | Meaning |
|------|---------|-----------|---------|
| `--json` | off | stats | JSON output |
| `--charts` | off | stats | ASCII charts |
| `--pause-speed` | 1.0 | both | stationary speed threshold (km/h) |
| `--pause-duration` | 10s | both | minimum pause duration |
| `--max-points` | 500000 | both | maximum track points accepted |
| `--addr` | `:8080` | serve | listen address |
| `--max-upload` | 26214400 (25 MB) | serve | maximum upload size (bytes) |

## Development

```sh
go test -race ./...   # unit + handler tests with the race detector (or: task test)
task lint             # gofmt + go vet + golangci-lint
```

## Project layout

- `cmd/gpx-stats` — thin entrypoint (CLI + `serve` dispatch)
- `internal/gpx` — hardened GPX parser (stdlib `encoding/xml`)
- `internal/stats` — pure statistics engine (distance, elevation, pauses, splits)
- `internal/config` — typed configuration
- `internal/cli` — text/JSON output and ASCII charts (`asciigraph`)
- `internal/web` — HTTP server, embedded templates/assets, SVG charts (`go-analyze/charts`)
- `testdata/` — sample and adversarial GPX fixtures

The domain packages (`gpx`, `stats`, `config`) never import the transport
packages (`cli`, `web`), so both interfaces compute identical results.
