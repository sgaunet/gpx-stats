# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

(The same statistics also ship as a CLI; this record covers the embedded web UI started by
`gpx-stats serve`. The CLI's own presentation rules live in `specs/001-gpx-stats/contracts/cli.md`.)

## Users

Mountain hikers and trail users looking back at their own outing. They come home from a day in the
mountains, export the GPX from their watch or app (Komoot, Garmin, etc.), and open the local page to
see the route on a map, how much they climbed and descended, how long it took, and what the day
cost in effort kilometers. Elevation and effort matter as much as distance; speed matters less than
it does to runners or cyclists.

## Product Purpose

A single, self-contained Go binary that computes statistics for one GPX file, from the command
line or through a small embedded web page. You upload or drop a file and get distance, ascent and
descent, total, moving and pause time, average speeds, effort kilometers, per-kilometer splits,
elevation and speed charts, and the route drawn on a map. Success means a hiker reads their day
at a glance, and every number can be trusted and traced to the file.

## Positioning

- **Nothing is stored.** The file is processed in memory and never written to disk. There is no
  database, no account, no cookie and no client-side persistence.
- **No server-side network.** The server makes zero outbound calls. Map tiles are the only network
  traffic, fetched by the browser directly from the tile provider, which can see the area being
  viewed.
- **Every recorded point is drawn.** The route is never simplified, decimated or smoothed.
  Interrupted recordings (multiple segments) are never joined across ground that was not
  travelled.
- **Effort kilometers under both common conventions** (climb only; climb + descent), each with its
  legend stated verbatim. Neither is presented as the correct one.
- **CLI and web parity.** Both surfaces read the same computed figures, with the same canonical
  labels.

## Operating Context

- Run locally (`gpx-stats serve`, default `:8080`), or from the Docker image. Single user, stateless.
- Input: one GPX 1.1 file per request, up to the configured upload cap (25 MB default) and a
  bounded track-point count.
- Typical files: day hikes and multi-hour mountain routes with elevation and timestamps. Some files
  lack elevation or timestamps, or contain several segments or tracks.
- Three per-upload analysis settings: pause speed threshold (km/h), minimum pause duration (Go
  duration, e.g. `2m0s`) and elevation noise threshold (m).

## Capabilities and Constraints

- Server-rendered HTML (`html/template`). Every statistic, chart and split renders without
  JavaScript. JavaScript only drives the interactive map (Leaflet 1.9.4, vendored) and drag-and-drop
  onto it.
- All assets are embedded in the binary and served by it. No CDN, no hosted fonts, no third-party
  scripts.
- Dropping a file must go through the same form submission as the upload control.
- Canonical labels and legend strings are a contract (`specs/003-km-effort/contracts/ui-labels.md`),
  asserted literally by tests and shared with the CLI and JSON.
- Unavailable figures read `n/a`, never a fabricated `0.00`.
- The page heading names the page ("Activity statistics"), never the file. File-supplied identity
  (name, type, creator, times) is the only user-controlled text and is always escaped.
- Activity timestamps are shown as RFC3339 on the web (user decision, 2026-11-16). `<metadata><time>`
  is the file's write time, never presented as the activity date.
- Charts are server-rendered SVG (`go-analyze/charts`). The kilometer splits table and the charts
  carry no effort column (FR-016).
- Base layers: OpenStreetMap, OpenTopoMap, CyclOSM, Humanitarian, each with its attribution kept
  legible.

## Brand Commitments

None beyond the name `gpx-stats` and the privacy statements above, which must stay literally true.

## Evidence on Hand

- `data/Balaïtous.gpx`: a real 18.85 km Pyrenean hike (1678 m ascent, about 12h53 elapsed), used as
  representative content.
- `testdata/*.gpx`: edge cases (no elevation, no time, single point, two segments, two tracks,
  dateline, malicious input).
- No users, testimonials, metrics or logos exist. Do not invent any.

## Product Principles

1. The numbers are the product: every figure is legible, traceable to the file, and never invented.
2. Privacy claims are exact. Say what leaves the machine and what does not.
3. Hiker-first hierarchy: climb, descent and effort sit beside distance, not beneath it.
4. Works without JavaScript and offline from the binary. Enhancement never replaces the baseline.
5. Fidelity over prettiness: the route and profiles show what was recorded.

## Accessibility & Inclusion

No formal standard has been set. Working baseline: WCAG 2.2 AA contrast, keyboard-operable controls,
visible focus, start and end markers that do not rely on colour alone, and full statistics without
JavaScript.
