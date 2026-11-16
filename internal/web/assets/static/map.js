/* gpx-stats map view.
 *
 * Deliberately thin glue. Everything worth testing — coordinate rounding,
 * antimeridian unwrapping, bounds, the layer table — is computed in Go and
 * arrives as JSON. This file initialises Leaflet, draws what it is given, and
 * wires four controls. It never computes geometry and never talks to the
 * server except by submitting the page's existing form.
 *
 * Privacy: map tiles are requested by the browser directly from the tile
 * provider. The GPX file and the computed statistics are never sent anywhere.
 */
(function () {
  'use strict';

  /** Reads and parses a JSON payload embedded in the page. */
  function readJSON(id) {
    var el = document.getElementById(id);
    if (!el) {
      return null;
    }
    try {
      return JSON.parse(el.textContent);
    } catch (e) {
      return null;
    }
  }

  // ---------------------------------------------------------------- layers

  function buildLayers(map, specs) {
    var layers = {};
    var initial = null;

    specs.forEach(function (spec) {
      var layer = L.tileLayer(spec.url, {
        attribution: spec.attribution,
        maxZoom: spec.maxZoom,
      });
      // A provider being down must never break the page: the route, the
      // markers and every control stay usable over an empty backdrop.
      layer.on('tileerror', function () {
        /* intentionally silent - see FR-019 */
      });
      layers[spec.name] = layer;
      if (spec.default || initial === null) {
        initial = layer;
      }
    });

    if (initial) {
      initial.addTo(map);
    }
    if (specs.length > 1) {
      L.control.layers(layers, null, { position: 'topright' }).addTo(map);
    }
  }

  // ----------------------------------------------------------------- route

  function endpointIcon(kind, label) {
    return L.divIcon({
      className: '',
      html:
        '<div class="gpx-endpoint gpx-endpoint-' +
        kind +
        '" title="' +
        label +
        '" aria-label="' +
        label +
        '">' +
        label.charAt(0) +
        '</div>',
      iconSize: [26, 26],
      iconAnchor: [13, 13],
    });
  }

  /**
   * Reads a colour token from the page's stylesheet, so the map's inks are
   * defined in one place with the rest of the palette. Falls back when the
   * token is missing (a stylesheet that failed to load must not cost the user
   * the route).
   */
  function cssToken(name, fallback) {
    var value = window.getComputedStyle(document.documentElement).getPropertyValue(name);
    return value && value.trim() ? value.trim() : fallback;
  }

  function drawRoute(map, cfg) {
    // One entry per recorded segment, each already a list of [lat, lon] pairs:
    // exactly the multi-polyline shape L.polyline accepts, so there is nothing
    // to convert here. Segments are drawn as separate lines on purpose — the
    // ground between them was never travelled.
    var segs = readJSON('gpx-route');
    if (!segs || !segs.length) {
      return;
    }

    // Counted rather than flattened: concatenating a 500k-point track just to
    // learn its length would allocate a second copy of the whole payload.
    var count = 0;
    for (var i = 0; i < segs.length; i++) {
      count += segs[i].length;
    }
    if (!count) {
      return;
    }
    var lastSeg = segs[segs.length - 1];
    var first = segs[0][0];
    var last = lastSeg[lastSeg.length - 1];

    if (count === 1) {
      // A lone point: a marker, no line. Drawing a zero-length polyline would
      // render nothing at all.
      L.marker(first, { icon: endpointIcon('start', 'Start') }).addTo(map);
      return;
    }

    L.polyline(segs, {
      color: cssToken('--map-route', '#c8102e'),
      weight: 3,
      opacity: 0.85,
      // Canvas rather than SVG: one huge path in the DOM is what actually
      // stalls the browser on a long track.
      renderer: L.canvas(),
      // Leaflet's default smoothFactor applies Douglas-Peucker simplification
      // at render time. The spec requires every recorded point to be drawn
      // (FR-001a), so it is disabled. This is a deliberate fidelity-over-
      // smoothness trade, not an oversight - do not "optimise" it back.
      smoothFactor: 0,
    }).addTo(map);

    // The activity's start and end, which are the first point of the first
    // segment and the last point of the last one.
    L.marker(first, { icon: endpointIcon('start', 'Start') }).addTo(map);
    L.marker(last, { icon: endpointIcon('end', 'End') }).addTo(map);
  }

  function frame(map, cfg) {
    // No animation: framing is a layout step, and it must finish (and fire
    // its move events) before refitOnResize's guard is lowered.
    if (cfg.useBounds && cfg.bounds) {
      map.fitBounds(
        [
          [cfg.bounds[0], cfg.bounds[1]],
          [cfg.bounds[2], cfg.bounds[3]],
        ],
        { padding: [20, 20], animate: false }
      );
      return;
    }
    // Single point, or an extent too small to frame: the server has already
    // decided this and supplied a centre and zoom.
    map.setView(cfg.center, cfg.zoom, { animate: false });
  }

  /**
   * Keeps the server's framing when the map's box changes size: a window
   * resize, a phone rotating, the layout switching between its desktop and
   * single-column forms, or entering full screen. Leaflet only lays tiles out
   * for the size it last measured, and the framing was computed for that size
   * too. Once the visitor has panned or zoomed, the view is theirs and only
   * the tile layout is refreshed.
   */
  function refitOnResize(map, cfg, container) {
    if (typeof window.ResizeObserver !== 'function') {
      return;
    }
    var touched = false;
    var framing = false;
    // Drag, wheel, buttons, keyboard and pinch all start a move or a zoom;
    // the refit's own move is fenced off by the framing flag.
    map.on('movestart zoomstart', function () {
      if (!framing) {
        touched = true;
      }
    });
    var observer = new window.ResizeObserver(function () {
      map.invalidateSize({ pan: false });
      if (touched) {
        return;
      }
      framing = true;
      frame(map, cfg);
      framing = false;
    });
    observer.observe(container);
  }

  // ------------------------------------------------------------ fullscreen

  function addFullscreenControl(map, container) {
    var Control = L.Control.extend({
      options: { position: 'topleft' },
      onAdd: function () {
        var wrap = L.DomUtil.create('div', 'leaflet-bar gpx-fullscreen-control');
        var link = L.DomUtil.create('a', '', wrap);
        link.href = '#';
        // A drawn icon, not a Unicode glyph: the glyph's rendering depends on
        // whichever font happens to carry it. Fixed markup, no user input.
        link.innerHTML =
          '<svg viewBox="0 0 20 20" width="18" height="18" aria-hidden="true" focusable="false">' +
          '<path d="M3 8V3h5M12 3h5v5M17 12v5h-5M8 17H3v-5" fill="none" stroke="currentColor" ' +
          'stroke-width="1.75" stroke-linecap="square"/></svg>';
        link.title = 'Toggle full screen';
        link.setAttribute('role', 'button');
        link.setAttribute('aria-label', 'Toggle full screen');

        L.DomEvent.on(link, 'click', function (e) {
          L.DomEvent.stop(e);
          toggle();
        });
        return wrap;
      },
    });
    map.addControl(new Control());

    function isPseudo() {
      return container.classList.contains('is-pseudo-fullscreen');
    }

    function enterPseudo() {
      container.classList.add('is-pseudo-fullscreen');
      map.invalidateSize();
    }

    function exitPseudo() {
      container.classList.remove('is-pseudo-fullscreen');
      map.invalidateSize();
    }

    function toggle() {
      if (document.fullscreenElement === container) {
        document.exitFullscreen();
        return;
      }
      if (isPseudo()) {
        exitPseudo();
        return;
      }
      if (!container.requestFullscreen) {
        // No Fullscreen API: enlarge within the page instead. The user sees a
        // working control, never an error (FR-010).
        enterPseudo();
        return;
      }
      var result = container.requestFullscreen();
      if (result && typeof result.catch === 'function') {
        // Permission policy can reject this; fall back rather than fail.
        result.catch(enterPseudo);
      }
    }

    // Fires for both the control and the browser's own Escape gesture, which
    // is why Escape works for free. Leaflet lays tiles out from the container
    // size, so it must be told the size changed - in BOTH directions, or tiles
    // render into the wrong area.
    document.addEventListener('fullscreenchange', function () {
      map.invalidateSize();
    });
  }

  // -------------------------------------------------------------- dropzone

  function addDropzone(map, container) {
    var form = document.querySelector('form[action="/analyze"]');
    var input = document.getElementById('file');
    var message = document.getElementById('gpx-drop-message');

    // Without these two the drop cannot be turned into an ordinary form
    // submission, so no drop affordance is advertised at all and the form
    // remains the working path (FR-016f).
    var supported =
      form &&
      input &&
      typeof window.DataTransfer === 'function' &&
      typeof form.requestSubmit === 'function';
    if (!supported) {
      return;
    }

    // Only now that a drop can really be turned into a form submission is the
    // affordance advertised. Without scripting the hint stays hidden, and the
    // form remains the one path the page offers.
    var hint = document.getElementById('gpx-drop-hint');
    if (hint) {
      hint.hidden = false;
    }

    function say(text, isError) {
      if (!message) {
        return;
      }
      message.textContent = text;
      message.classList.toggle('is-error', !!isError);
      message.hidden = !text;
    }

    // dragenter/dragleave fire for child elements too; counting keeps the
    // highlight stable while the pointer moves across the map.
    var depth = 0;

    function highlight(on) {
      container.classList.toggle('is-drop-target', on);
    }

    ['dragenter', 'dragover'].forEach(function (name) {
      container.addEventListener(name, function (e) {
        e.preventDefault();
        e.stopPropagation();
        if (name === 'dragenter') {
          depth++;
        }
        if (e.dataTransfer) {
          e.dataTransfer.dropEffect = 'copy';
        }
        highlight(true);
        say('Drop your GPX file to analyze it.', false);
      });
    });

    container.addEventListener('dragleave', function (e) {
      e.preventDefault();
      e.stopPropagation();
      depth = Math.max(0, depth - 1);
      if (depth === 0) {
        highlight(false);
        say('', false);
      }
    });

    container.addEventListener('drop', function (e) {
      e.preventDefault();
      e.stopPropagation();
      depth = 0;
      highlight(false);

      var dt = e.dataTransfer;
      var files = dt && dt.files ? dt.files : null;

      if (!files || files.length === 0) {
        say('That does not look like a file. Drop a GPX file, or use the form below.', true);
        return;
      }
      if (files.length > 1) {
        // Never silently pick one of them (FR-016d).
        say('Please drop a single GPX file at a time.', true);
        return;
      }
      if (isDirectory(dt)) {
        say('Folders cannot be analyzed. Drop a single GPX file.', true);
        return;
      }

      // Hand the file to the existing form and submit it normally. This is
      // what makes the drop path *be* the form path: same request, same
      // validation, same errors, same pause-detection options currently shown
      // in the form, and a real navigation with working back-button behaviour.
      var carrier = new DataTransfer();
      carrier.items.add(files[0]);
      input.files = carrier.files;

      say('Analyzing ' + files[0].name + '…', false);
      form.requestSubmit();
    });

    function isDirectory(dt) {
      if (!dt.items || !dt.items.length) {
        return false;
      }
      var item = dt.items[0];
      if (typeof item.webkitGetAsEntry !== 'function') {
        return false;
      }
      var entry = item.webkitGetAsEntry();
      return !!entry && entry.isDirectory;
    }
  }

  // ------------------------------------------------------------ busy state

  /**
   * Marks the upload form busy once it is submitted, by the button or by a
   * drop, so a large file does not look like a dead click. Purely visual: the
   * submission itself is the browser's ordinary form post.
   */
  function addBusyState() {
    var form = document.querySelector('form[action="/analyze"]');
    if (!form) {
      return;
    }
    form.addEventListener('submit', function () {
      var button = form.querySelector('button[type="submit"]');
      form.setAttribute('aria-busy', 'true');
      if (button) {
        button.disabled = true;
        button.textContent = 'Analyzing…';
      }
    });
    // Coming back with the browser's back button restores the page from the
    // back-forward cache, busy state included; undo it.
    window.addEventListener('pageshow', function (e) {
      if (!e.persisted) {
        return;
      }
      var button = form.querySelector('button[type="submit"]');
      form.removeAttribute('aria-busy');
      if (button) {
        button.disabled = false;
        button.textContent = button.getAttribute('data-label') || button.textContent;
      }
    });
  }

  // ------------------------------------------------------------ bootstrap

  function init() {
    addBusyState();
    var container = document.getElementById('gpx-map');
    if (!container || typeof L === 'undefined') {
      return;
    }
    var cfg = readJSON('gpx-map-config');
    if (!cfg || !cfg.layers || !cfg.layers.length) {
      // Nothing to initialise from. Leave the rest of the page untouched
      // rather than throwing.
      return;
    }

    var map = L.map(container, {
      // Leaflet's defaults already cover pointer drag, wheel zoom, touch pinch
      // and the on-map zoom buttons.
      zoomControl: true,
    });

    buildLayers(map, cfg.layers);
    frame(map, cfg);
    if (cfg.hasRoute) {
      drawRoute(map, cfg);
    }
    addFullscreenControl(map, container);
    refitOnResize(map, cfg, container);
    if (cfg.dropzone) {
      addDropzone(map, container);
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
