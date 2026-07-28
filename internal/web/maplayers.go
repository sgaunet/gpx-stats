package web

// Base map layers offered to the browser.
//
// All four are free to use, need no API key and no registration, and serve over
// HTTPS. They are all in the OpenStreetMap ecosystem, which keeps the licensing
// story to one thing (ODbL data) rather than four. Esri World Imagery and the
// CARTO styles were deliberately left out: their terms require accepting a ToS
// or registering for a key, which is not something this project can do on
// behalf of everyone who runs the binary.
//
// Obligations that come with using them — these are requirements, not advice:
//
//   - Attribution must stay visible on the map. The OSM Foundation tile usage
//     policy requires it and forbids hiding it behind a toggle or off-screen.
//   - No bulk downloading, pre-seeding or offline caching of tiles. The policy
//     bans it outright and blocks offenders without notice.
//   - Tiles are fetched by the browser, never proxied or cached by this server.
//   - Do not set a restrictive Referrer-Policy on pages that show a map: OSM
//     requires web applications to send a valid Referer.
//
// OpenTopoMap in particular is a best-effort service with no uptime guarantee,
// which is why plain OpenStreetMap is the default and why the page must stay
// usable when tiles fail to load.

// baseLayer describes one selectable tile source.
type baseLayer struct {
	Key         string `json:"key"`         // stable identifier
	Name        string `json:"name"`        // label in the layer control
	URLTemplate string `json:"url"`         // Leaflet tile URL template, HTTPS only
	Attribution string `json:"attribution"` // required credit, rendered as HTML
	MaxZoom     int    `json:"maxZoom"`     // deepest zoom the provider supplies
	Default     bool   `json:"default"`     // exactly one layer sets this
}

// attrOSM is the OpenStreetMap credit. Its data underlies every layer offered
// here, so this fragment is required on all of them.
const attrOSM = `&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors`

// baseLayers is the immutable table of offered layers. It is a package-level
// constant table rather than configuration: there is no present need to make
// the provider list tunable, and adding a knob "just in case" is exactly the
// speculative complexity the project constitution rules out.
//
// Exactly one entry must have Default set — enforced by test.
var baseLayers = []baseLayer{
	{
		Key:         "osm",
		Name:        "OpenStreetMap",
		URLTemplate: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
		Attribution: attrOSM,
		MaxZoom:     19,
		Default:     true,
	},
	{
		Key:         "topo",
		Name:        "OpenTopoMap",
		URLTemplate: "https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png",
		Attribution: `Map data: ` + attrOSM +
			`, <a href="https://viewfinderpanoramas.org">SRTM</a> | ` +
			`Style: &copy; <a href="https://opentopomap.org">OpenTopoMap</a> ` +
			`(<a href="https://creativecommons.org/licenses/by-sa/3.0/">CC-BY-SA</a>)`,
		MaxZoom: 17,
	},
	{
		Key:         "cyclosm",
		Name:        "CyclOSM",
		URLTemplate: "https://{s}.tile-cyclosm.openstreetmap.fr/cyclosm/{z}/{x}/{y}.png",
		Attribution: attrOSM +
			` | Tiles: <a href="https://www.cyclosm.org/">CyclOSM</a>, ` +
			`hosted by <a href="https://openstreetmap.fr/">OpenStreetMap France</a>`,
		MaxZoom: 20,
	},
	{
		Key:         "humanitarian",
		Name:        "Humanitarian",
		URLTemplate: "https://{s}.tile.openstreetmap.fr/hot/{z}/{x}/{y}.png",
		Attribution: attrOSM +
			` | Tiles: <a href="https://www.hotosm.org/">HOT</a>, ` +
			`hosted by <a href="https://openstreetmap.fr/">OpenStreetMap France</a>`,
		MaxZoom: 20,
	},
}
