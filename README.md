# gpx-stats

A single, self-contained Go binary that computes statistics about a GPX file —
from the command line or through a small embedded web UI. **No database, no
server-side network calls, nothing stored.**

> The web UI now shows your track on a map. Map *backgrounds* are loaded by your
> browser directly from the tile provider, which can see the area you are
> viewing. Your GPX file and its statistics never leave the server. See
> [Privacy](#privacy).

## Features

- **CLI**: pass a GPX path, get raw statistics as text (or `--json`), plus
  optional ASCII charts in the terminal.
- **Web UI**: `serve` starts a local page where you upload a GPX file — or drop
  it straight onto the map — and see the same statistics rendered with inline
  SVG charts. The file is processed in memory and is never saved.
- **Interactive map**: the recorded track is drawn over a choice of four free
  base layers (OpenStreetMap, OpenTopoMap, CyclOSM, Humanitarian), framed on the
  activity with start/end markers, and expandable to full screen. Every recorded
  point is drawn — the line is never simplified.
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

Upload a GPX file — or drag one onto the map — to see the statistics, charts and
the route. Optional form fields override the pause thresholds, and a dropped file
uses whatever values the form currently shows. Nothing is persisted.

### The map

The track is drawn over one of four base layers, switchable from the control in
the top-right corner:

| Layer | Attribution |
|-------|-------------|
| OpenStreetMap *(default)* | © OpenStreetMap contributors |
| OpenTopoMap | © OpenStreetMap contributors, SRTM; style © OpenTopoMap (CC-BY-SA) |
| CyclOSM | © OpenStreetMap contributors; tiles by CyclOSM, hosted by OpenStreetMap France |
| Humanitarian | © OpenStreetMap contributors; tiles by HOT, hosted by OpenStreetMap France |

All four are free and need no API key. Their attribution is displayed on the map
as their licences require.

The map is built with [Leaflet](https://leafletjs.com/) 1.9.4 (BSD-2-Clause),
vendored into the repository and embedded in the binary — there is no CDN
dependency. See `internal/web/assets/static/vendor/leaflet/VERSION` for the
version, licence and checksums.

If the tile provider is unreachable, the route, the markers and every control
stay usable over a blank backdrop, and the statistics are unaffected. With
JavaScript disabled the map is replaced by a short note; everything else on the
page still renders.

## Privacy

- Your GPX file is processed **in memory** and never written to disk.
- The **server makes no outbound network connections at all**. It does not
  proxy, relay or cache map tiles.
- Map background tiles are requested by **your browser**, directly from the tile
  provider. The provider therefore sees the geographic area you are looking at
  (as it would for any map on the web). It never receives your GPX file, your
  coordinates or your statistics.
- Nothing is stored client-side either: the selected layer, zoom and position
  are forgotten when you leave the page.

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

In the web UI the two limits interact: a GPX file averages ~100 bytes per track
point, so `--max-upload` binds well before `--max-points`. At the defaults the
practical ceiling is around **260,000 points** (a ~26 MB file); larger uploads are
rejected with `413` before the point limit applies. Raise `--max-upload` too if
you need to analyse longer tracks in the browser. The CLI reads from disk and is
bounded only by `--max-points`.

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
