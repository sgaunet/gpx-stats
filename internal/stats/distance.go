package stats

import "math"

// earthRadiusKm is the mean Earth radius used for great-circle distance.
const earthRadiusKm = 6371.0088

// haversineKm returns the 2D great-circle distance in kilometers between two
// coordinates given in decimal degrees. Elevation is intentionally excluded
// (distance is horizontal; elevation is reported separately as gain).
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	rlat1 := lat1 * math.Pi / 180
	rlat2 := lat2 * math.Pi / 180
	dlat := (lat2 - lat1) * math.Pi / 180
	dlon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(rlat1)*math.Cos(rlat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}
