# gpx-stats demo.
# Regenerate with: task demo   (or: vhs doc/demo.tape, run from the repository root)
#
# The recording uses data/Balaïtous.gpx, a real hike in the Pyrenees, tracked here
# so the tape renders from a clean clone.

Output doc/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1000
Set Height 1170
Set Padding 20
Set Framerate 24
Set TypingSpeed 40ms
Set CursorBlink false

Hide
Type "export PATH=$PWD:$PATH" Enter
Type "export PS1='$ '" Enter
Type "clear" Enter
Show

Type@25ms "# Statistics for a GPX track: a hike up the Balaïtous, Pyrenees" Enter
Sleep 500ms
Type "gpx-stats data/Balaïtous.gpx" Enter
Sleep 5s

Hide
Type "clear" Enter
Show

Type@25ms "# ASCII elevation and speed charts, right in the terminal" Enter
Sleep 500ms
Type "gpx-stats --charts data/Balaïtous.gpx" Enter
Sleep 5s

Hide
Type "clear" Enter
Show

Type@25ms "# Machine-readable output for scripts" Enter
Sleep 500ms
Type "gpx-stats --json data/Balaïtous.gpx | head -40" Enter
Sleep 3s

Type@25ms "# Or serve the same statistics with an interactive map" Enter
Sleep 500ms
Type "gpx-stats serve --addr :8080" Enter
Sleep 3s
Ctrl+C
Sleep 1500ms
