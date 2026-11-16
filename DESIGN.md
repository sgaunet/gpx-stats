---
name: gpx-stats
description: The map sheet of your day. GPX statistics set as a topographic sheet's marginalia beside the live map.
colors:
  paper: "#fafaf7"
  paper-2: "#f1efe7"
  ink: "#1c1c1c"
  ink-2: "#57524b"
  rule: "#dad5ca"
  neatline: "#1c1c1c"
  contour: "#a3683c"
  contour-ink: "#8a5530"
  water: "#2f77ae"
  water-ink: "#245f8c"
  forest: "#cfe2b0"
  route: "#c8102e"
  start: "#257942"
  danger: "#a5221b"
  danger-field: "#fbeae7"
  paper-night: "#16181a"
  paper-2-night: "#1e2124"
  ink-night: "#eceae4"
  ink-2-night: "#ada79c"
  rule-night: "#383b3f"
  neatline-night: "#ada79c"
  contour-night: "#d49a68"
  contour-ink-night: "#e2b285"
  water-night: "#7db5e0"
  water-ink-night: "#a6cdec"
  forest-night: "#2f3c28"
  route-night: "#ff5c6c"
  start-night: "#4cc07a"
  danger-night: "#ff9a90"
  danger-field-night: "#3a1f1d"
  map-halo: "#ffffff"
typography:
  headline:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "1.75rem"
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "0.04em"
  figure:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "1.75rem"
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: "-0.01em"
    fontFeature: "\"tnum\" 1, \"lnum\" 1"
  figure-minor:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "1.375rem"
    fontWeight: 600
    lineHeight: 1.1
    fontFeature: "\"tnum\" 1, \"lnum\" 1"
  title:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 700
    letterSpacing: "0.1em"
  body:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 400
    lineHeight: 1.45
  body-sm:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.3
  label:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 700
    letterSpacing: "0.12em"
  note:
    fontFamily: "\"Source Sans 3\", system-ui, -apple-system, \"Segoe UI\", sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.4
  code:
    fontFamily: "ui-monospace, \"SF Mono\", Menlo, Consolas, monospace"
    fontSize: "0.9em"
    fontWeight: 400
rounded:
  none: "0"
spacing:
  s-1: "4px"
  s-2: "8px"
  s-3: "12px"
  s-4: "16px"
  s-5: "24px"
  s-6: "32px"
  s-7: "48px"
  pad-x: "28px"
  pad-x-narrow: "20px"
  legend-w: "clamp(340px, 28vw, 420px)"
components:
  button-primary:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
    rounded: "{rounded.none}"
    padding: "8px 24px"
    height: "44px"
  button-primary-hover:
    backgroundColor: "{colors.water-ink}"
    textColor: "{colors.paper}"
  button-quiet:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "6px 12px"
    height: "36px"
  button-quiet-hover:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
  input:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "8px 12px"
    height: "42px"
  file-drop-field:
    backgroundColor: "{colors.paper-2}"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.none}"
    padding: "24px 16px"
  tag:
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "1px 7px"
  notice:
    backgroundColor: "{colors.paper-2}"
    rounded: "{rounded.none}"
    padding: "12px 16px"
  notice-error:
    backgroundColor: "{colors.danger-field}"
    textColor: "{colors.ink}"
  legend-entry:
    textColor: "{colors.ink-2}"
    padding: "12px 0"
  map-control:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
  map-control-hover:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
---

# Design System: gpx-stats

## Overview

**Creative North Star: "The Topographic Sheet"**

Every page of the web UI is a 1:25 000 map sheet in the swisstopo / IGN tradition: a legend column of marginalia on the left, a large map field on the right, and further plates continuing the same two-column grid below. The statistics are not cards floating over a map; they are the sheet's own legend. Distance is drawn as a checkered scale bar, ascent and descent carry spot-height triangles, and the elevation profile runs along the sheet's bottom margin under a double neatline.

The page is a working document, not a poster. Density is high but ordered: one hairline between legend entries, tabular lining figures right-aligned against their labels, tracked capitals that letter each group the way a map names itself, italic for notes and the file's own identity. Colour is printed ink with fixed meanings, never decoration. Depth is absent: the sheet is flat paper, framed by rules.

The world has a night sheet (`prefers-color-scheme: dark`) that redefines tokens only, lifting the same inks for a dark ground. Map tiles are not re-tinted; the night sheet themes the chrome, the legend and the charts.

**Key Characteristics:**
- Two-column sheet: legend column (`legend-w`) beside the map field, repeated by every plate below.
- Double-rule neatlines (3px double) frame the legend column, the profile strip and each plate.
- Inks with fixed meanings: black lettering, contour brown, water blue, forest green, route red.
- One self-hosted humanist sans (Source Sans 3) in roman, italic and tracked caps; tabular figures.
- Square, flat, ink-bordered controls; zero radius, zero shadow in the chrome.
- Everything embedded in the binary: fonts, icons, Leaflet; no CDN, no hosted font.

## Colors

A map's printed inks on map paper: near-neutral warm grounds, one black, and four hues whose meaning never moves.

### Primary
- **Lettering Black** (`ink`): all text, figures, square control borders, the scale bar's filled segments and the primary button's ground.
- **Neatline** (`neatline`): the double rules that frame legend, profile and plates, and the splits table's header rule. Equal to ink by day, lifted to warm grey at night.

### Secondary
- **Contour Brown** (`contour`): elevation and nothing else of substance. The elevation profiles (line and a 24% tint fill, 30% at night), the spot-height triangles, the brand mark (a peak hillshaded from the north-west: the lit face is contour mixed 60% into paper and the shaded face full contour, swapped at night so the lit face stays the lighter tone), the dashed file-drop border and the settings disclosure chevron. `contour-ink` is its text-weight shade.
- **Water Blue** (`water`, `water-ink`): interaction. Links (`water-ink`), the focus ring, the caret, the base-layer radio's accent, the landing map's dashed drop frame, and the primary button's hover ground.

### Tertiary
- **Route Red** (`route`, drawn on the map as the fixed `map-route`): the route polyline and the end marker. `map-route` is the same red held fixed in both schemes: the tiles do not change with the scheme, so neither do the inks drawn on them. Only the route is red.
- **Start Green** (`start`, drawn on the map as the fixed `map-start`): the start marker only.
- **Map Halo** (`map-halo`, white, fixed in both schemes): the ring and letter of the start and end markers, which sit on light tiles in either scheme.
- **Forest Tint** (`forest`): the text-selection highlight, like a woodland tint on a sheet.

### Neutral
- **Map Paper** (`paper`): page ground and the face of every control.
- **Paper Shade** (`paper-2`): the map field while tiles load, the blank map face (no coordinates, the error page), notices, the file-drop field, splits row hover.
- **Grey Lettering** (`ink-2`): labels, units, notes, captions, chart axes and text, and the speed-per-kilometer bars.
- **Hairline** (`rule`): the 1px rule between legend entries, settings borders, table rows, chart grid lines.
- **Danger** (`danger`, `danger-field`): invalid input border and the error notice's border and ground.

### Named Rules
**The Fixed Inks Rule.** Each hue has one meaning. Brown is elevation, blue is interaction, red is the route, green is the start. A new chart or figure picks ink-2 before it borrows a meaning: that is why speed bars are ink-2, not brown and not blue.

**The Water Is Interaction Rule.** Blue never colours data. It marks what can be pressed, followed, focused or chosen. (Text selection is the one "selection" that is not blue: it is the forest tint.)

**The Tokens-Only Night Rule.** The night sheet redefines tokens on `:root` and nothing else. Any colour a component hard-codes will not follow it.

## Typography

**Display Font:** none; there is no display face.
**Body Font:** Source Sans 3 variable (weights 200–900, roman and italic, Latin subset, OFL, self-hosted woff2) with `system-ui, -apple-system, "Segoe UI", sans-serif`.
**Label/Mono Font:** the same family in tracked capitals; `ui-monospace` only for inline `code` in help text.

**Character:** One humanist sans in the map-lettering tradition, doing every job through case, weight, italic and figure style rather than a second family.

### Hierarchy
- **Headline** (700, 1.75rem / 28px, 1.1, 0.04em, uppercase, balanced): the page's h1, which names the page ("Activity statistics", "GPX Statistics", "Could not process the file"), never the file.
- **Figure** (600, 1.75rem / 28px, 1.1, -0.01em, tabular lining): primary legend values; units ride along in 15px/400 ink-2. Drops to 22px under 480px.
- **Figure Minor** (600, 1.375rem / 22px): the Time & speed figures.
- **Title** (700, 0.9375rem / 15px, 0.1em, uppercase): plate headings ("Kilometer splits").
- **Body** (400, 1.0625rem / 17px, 1.45): running text; ledes capped at 40ch, privacy note at 60ch.
- **Body Small** (400, 0.9375rem / 15px, 1.3): legend labels, rates, tables, settings, controls' secondary text.
- **Label** (700, 0.8125rem / 13px, 0.12em, uppercase, ink-2): legend group headings and chart captions; table headers at 0.08em; the activity-type tag at 600/0.08em.
- **Note** (400 italic, 0.8125rem / 13px, 1.4, ink-2): legends of the effort conventions, segment notes, settings summary. The identity line is italic at 15px.
- **Code** (400, system monospace at 0.9em of its context): literal values the visitor types, such as the Go duration examples (`10s`, `1m30s`) in the form help. Never a costume for "technical" text.

### Named Rules
**The Lettering Rule.** Hierarchy comes from case, weight and italic inside one family. Spaced capitals name things (the sheet, a group, a plate, a chart); italic annotates (notes, units, identity); roman states figures. No second typeface, and never a system display face.

**The Tabular Rule.** Every figure, scale tick, rate and split uses tabular lining numerals so columns of numbers align. Unavailable figures read `n/a`, never `0.00`.

## Layout

A two-column sheet grid: `legend-w` (clamp 340–420px) beside `minmax(0, 1fr)`. The legend stacks its blocks at 24px gaps with 28px side padding; entries sit 12px top and bottom between 1px hairlines. Short figures pair two per row (label above value); long ones run label left, figure right on a baseline grid. Below the first sheet, each plate (Elevation vs. time, Speed per kilometer, Kilometer splits) repeats the grid: a heading-and-note head in the legend column, the body in the field.

At 1100px and wider, the field (map above, profile strip at its foot) is sticky at one full viewport while the legend scrolls past, so route and climb are always in view together. Below 1100px everything collapses to one column: legend first, closed by a double rule, then the map at `min(62vh, 560px)` (56vh with a 300px floor under 480px), then the profile and plates; side padding drops to 20px. Charts ship a wide (960×260) and a compact (420×260) render; a container query at 600px picks one.

Spacing is a fixed 4/8/12/16/24/32/48 scale (`s-1`…`s-7`). The type scale is equally tight: five steps from 13 to 28px. This is a working page, not a poster.

## Elevation & Depth

None. The sheet is flat paper: depth is replaced by rules (1px hairlines inside a column, 3px double neatlines between regions) and by the paper shade for inert grounds. Leaflet's own control shadows are explicitly removed.

### Shadow Vocabulary
- **Marker halo** (`box-shadow: 0 1px 3px rgba(0, 0, 0, 0.45)` with a 2px white ring): only on the start and end markers, which sit on arbitrary tile imagery and must stay legible there. It is a map-symbol halo, not a chrome elevation.

### Named Rules
**The Printed Sheet Rule.** Nothing in the chrome casts a shadow or floats. If a region needs separating, rule it.

## Shapes

Square everywhere in the chrome: buttons, inputs, the file selector, tags, notices, map controls and the drop hint all carry zero radius and a 1px ink (or ink-2) border. Circles appear only as map point symbols (the 26px start and end markers). The recurring cartographic forms are the double neatline, the checkered scale bar (8px tall, alternating ink and paper segments in a 1px ink frame, closed by a 5px tick at the total), the 13px spot-height triangles (up for ascent, down for descent) in the label gutter, and dashed borders for drop targets (1.5px contour on the file field, 3px water on the map).

## Components

### Buttons
Ink-solid and square; they read like a stamp on the sheet.
- **Shape:** square (0), 1px ink border.
- **Primary:** ink ground, paper text, 650 weight, 44px minimum height, 8px × 24px padding.
- **Hover / Focus:** ground shifts to `water-ink` over 160ms ease-out; 1px press on `:active`; focus is the global 2px water outline at 2px offset. Disabled during submit at 0.7 opacity with a progress cursor.
- **Quiet:** transparent with an ink border, 15px text, 36px minimum height; hover inverts to ink ground and paper text. Used for "Analyze another file".

### Chips
- **Tag:** the activity type in the identity line: 1px ink-2 border, 13px 600 tracked capitals, square. It shows a value from the file, not a heading.

### Cards / Containers
There are no cards. Containers are regions of the sheet: the legend column, the map field, the profile strip and plates, separated by neatlines.
- **Notice:** 1px rule border on paper-shade, 12px × 16px padding, 15px text; the error variant takes the danger border and danger-field ground.

### Inputs / Fields
- **Style:** 1px ink-2 border, square, paper ground, 42px minimum height, tabular numerals. Hover darkens the border to ink.
- **Focus:** border turns water and the 2px water outline sits flush (offset 0).
- **Error:** `:user-invalid` turns the border danger.
- **File drop field:** the landing page's first drop target: a large dashed contour field on paper-shade; its selector button is a square ink-bordered paper button that inverts on hover.
- **Analysis settings:** a `details` disclosure between hairlines, a contour chevron that rotates 90° over 160ms, the current values summarised in italic beneath the title.

### Navigation
The legend's top row: the brand (hillshaded peak mark at 26px with the "gpx-stats" wordmark, 650 weight, 15px) linking home at left, and a quiet button at right on the results page. It wraps under 480px.

### Legend Entry (signature)
A `dl` row between hairlines: label in 15px ink-2, value as a right-aligned tabular figure with its unit in ink-2. Distance carries the scale bar beneath; ascent and descent carry spot heights in the label gutter. The effort block groups by convention: the total in ink 600 label weight, its two rates as a small label/value list, then the convention's legend once in italic note style.

### Map Field
Leaflet controls redrawn in the sheet's vocabulary: square, flat, 1px ink border, paper face, hairline separators, inverting to ink on hover and focus; the layers toggle is an inline SVG token (`--icon-layers`), never Leaflet's PNG. Attribution stays at a legible 13px (the note size). On the landing page, a dragged file draws a 3px dashed water frame inset 8px, with a square ink-bordered hint at top centre (hidden on touch-only devices).

### Charts
Server-rendered SVG themed through `--chart-text`, `--chart-axis`, `--chart-grid`, `--chart-series`, `--chart-area` and `--chart-font`, so they follow the night sheet. Captions are tracked capitals with italic units. Elevation series are contour; speed overrides the series to ink-2.

## Do's and Don'ts

### Do:
- **Do** put every new statistic in the legend column as a hairline-ruled entry with a tabular figure, its unit in ink-2, and `n/a` when the data is absent.
- **Do** continue below the sheet with plates on the same `legend-w` / field grid, separated by the 3px double neatline.
- **Do** theme anything new through the `:root` tokens so the night sheet follows; add a night value for every new token.
- **Do** keep labels and legends verbatim from `specs/003-km-effort/contracts/ui-labels.md`, and keep the privacy note literally true ("not stored", "your browser").
- **Do** keep the h1 naming the page, with the file's identity in the italic line beneath it and activity timestamps in RFC3339.
- **Do** self-host every asset (fonts, icons, Leaflet) so it embeds in the Go binary.

### Don't:
- **Don't** use floating KPI cards, rounded containers or shadows in the chrome; the sheet is flat and square.
- **Don't** use water blue for data, or contour brown for anything that is not elevation (profiles, spot heights, the brand mark) or the file field's small disclosure and drop affordances; don't colour anything but the route red.
- **Don't** set a tracked-caps label above another heading as a kicker or eyebrow; spaced capitals are the heading itself.
- **Don't** add a second typeface, a hosted or CDN font, or an icon font.
- **Don't** hard-code a hex in a component; it will not follow the night sheet.
