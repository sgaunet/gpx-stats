#!/usr/bin/env bash
#
# Generate a synthetic GPX track for performance testing.
#
# The map draws every recorded point unreduced, so render cost scales with
# track size. These fixtures make that measurable and reproducible instead of
# depending on whatever large file happens to be lying around.
#
#   ./scripts/gen-gpx.sh 50000  > /tmp/50k.gpx    # the SC-002/SC-005 limit
#   ./scripts/gen-gpx.sh 500000 > /tmp/huge.gpx   # the parser's hard cap
#
# Output goes to stdout. These files are large and are not committed.

set -euo pipefail

points="${1:-50000}"

if ! [[ "${points}" =~ ^[0-9]+$ ]] || [[ "${points}" -lt 1 ]]; then
  echo "usage: $(basename "$0") <point-count>" >&2
  exit 2
fi

# Laps around a fixed 2.2 km loop. The radius must stay BOUNDED: an expanding
# spiral puts adjacent points ever further apart, producing an absurd total
# distance and a kilometre-splits table that dwarfs everything else on the page,
# which measures the wrong thing entirely.
#
# Geometry: radius 0.02 degrees (~2.2 km) with an angular step chosen so
# consecutive points are ~2 m apart, matching a 1 Hz recording at jogging pace.
# 50,000 points is then ~100 km, and 260,000 points ~520 km — long, but the
# kind of distance a multi-day activity plausibly reaches.
awk -v n="${points}" '
BEGIN {
  print "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
  print "<gpx version=\"1.1\" creator=\"gen-gpx.sh\" xmlns=\"http://www.topografix.com/GPX/1/1\">"
  print "  <trk><name>Synthetic " n " points</name><trkseg>"

  lat0 = 45.0; lon0 = 6.0
  epoch = 1686816000            # 2023-06-15T08:00:00Z

  radius = 0.02                              # ~2.2 km
  step   = 2.0 / (radius * 111320.0)         # ~2 m between points

  for (i = 0; i < n; i++) {
    angle = i * step
    lat = lat0 + radius * sin(angle)
    lon = lon0 + radius * cos(angle)
    ele = 100 + (i % 500) * 0.4

    t = epoch + i
    # gmtime is not portable in awk; format the timestamp arithmetically.
    days = int(t / 86400); rem = t % 86400
    hh = int(rem / 3600); mm = int((rem % 3600) / 60); ss = rem % 60

    printf "    <trkpt lat=\"%.6f\" lon=\"%.6f\"><ele>%.1f</ele><time>2023-06-%02dT%02d:%02d:%02dZ</time></trkpt>\n", \
      lat, lon, ele, 15 + int(days - 19523), hh, mm, ss
  }

  print "  </trkseg></trk>"
  print "</gpx>"
}'
