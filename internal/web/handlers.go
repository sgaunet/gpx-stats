package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	s.renderIndex(w, http.StatusOK, "")
}

func (s *Server) renderIndex(w http.ResponseWriter, status int, errMsg string) {
	data := map[string]any{
		"MaxUploadMB":   s.cfg.MaxUploadBytes / (1 << 20),
		"PauseSpeed":    s.cfg.StationarySpeedKmh,
		"PauseDuration": s.cfg.MinPauseDuration.String(),
		"Error":         errMsg,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		s.log.Error("rendering index page", "err", err)
	}
}

func (s *Server) renderError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.tmpl.ExecuteTemplate(w, "error.html", map[string]any{"Message": msg}); err != nil {
		s.log.Error("rendering error page", "err", err)
	}
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	// Cap the request body. Keeping the multipart form entirely in memory (the
	// same cap as maxMemory) avoids spilling parts to temporary files on disk,
	// honoring the "nothing is stored" guarantee.
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes)
	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			s.log.Warn("upload rejected: too large", "limit_bytes", s.cfg.MaxUploadBytes)
			s.renderError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("The uploaded file is too large. The maximum size is %d MB.", s.cfg.MaxUploadBytes/(1<<20)))
			return
		}
		s.renderError(w, http.StatusBadRequest, "Could not read the upload. Please submit a GPX file using the form.")
		return
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		s.renderError(w, http.StatusBadRequest, "Please choose a GPX file to upload.")
		return
	}
	defer func() { _ = file.Close() }()

	cfg := s.cfg
	if v := r.FormValue("pause_speed"); v != "" {
		f, perr := strconv.ParseFloat(v, 64)
		if perr != nil {
			s.renderError(w, http.StatusBadRequest, "Pause speed must be a number in km/h.")
			return
		}
		cfg.StationarySpeedKmh = f
	}
	if v := r.FormValue("pause_duration"); v != "" {
		d, perr := time.ParseDuration(v)
		if perr != nil {
			s.renderError(w, http.StatusBadRequest, "Pause duration must be a duration such as 10s or 1m30s.")
			return
		}
		cfg.MinPauseDuration = d
	}
	if verr := cfg.Validate(); verr != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid settings: "+verr.Error())
		return
	}

	track, err := gpx.Parse(file, cfg.MaxUploadBytes, cfg.MaxTrackPoints)
	if err != nil {
		s.log.Warn("rejected gpx upload", "err", err)
		s.renderError(w, http.StatusBadRequest, "Could not parse the GPX file: "+err.Error())
		return
	}

	res := stats.Compute(track, cfg)
	view, err := s.buildView(track, res)
	if err != nil {
		s.log.Error("building results view", "err", err)
		s.renderError(w, http.StatusInternalServerError, "Failed to render the results.")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "results.html", view); err != nil {
		s.log.Error("rendering results page", "err", err)
	}
	s.log.Info("analyzed gpx", "points", res.PointCount, "distance_km", res.TotalDistanceKm)
}

type splitView struct {
	Index      int
	DistanceKm string
	Duration   string
	SpeedKmh   string
}

type resultView struct {
	TotalDistanceKm        string
	AscendingElevationM    string
	HasElevation           bool
	TotalTime              string
	MovingTime             string
	PauseTime              string
	PauseCount             int
	HasTimes               bool
	AvgSpeedKmh            string
	AvgMovingSpeedKmh      string
	PointCount             int
	Splits                 []splitView
	ElevationDistanceChart template.HTML
	ElevationTimeChart     template.HTML
	SpeedChart             template.HTML
}

func (s *Server) buildView(track gpx.Track, res stats.Result) (resultView, error) {
	v := resultView{
		TotalDistanceKm: fmt.Sprintf("%.2f", res.TotalDistanceKm),
		HasElevation:    res.HasElevation,
		HasTimes:        res.HasTimes,
		PointCount:      res.PointCount,
	}
	if res.HasElevation {
		v.AscendingElevationM = fmt.Sprintf("%.0f", res.AscendingElevationM)
	}
	if res.HasTimes {
		v.TotalTime = formatDuration(res.TotalTime)
		v.MovingTime = formatDuration(res.MovingTime)
		v.PauseTime = formatDuration(res.PauseTime)
		v.PauseCount = res.PauseCount
		v.AvgSpeedKmh = fmt.Sprintf("%.2f", res.AvgSpeedKmh)
		v.AvgMovingSpeedKmh = fmt.Sprintf("%.2f", res.AvgMovingSpeedKmh)
		for _, sp := range res.Splits {
			v.Splits = append(v.Splits, splitView{
				Index:      sp.Index,
				DistanceKm: fmt.Sprintf("%.2f", sp.DistanceKm),
				Duration:   formatDuration(sp.Duration),
				SpeedKmh:   fmt.Sprintf("%.2f", sp.SpeedKmh),
			})
		}
	}

	eleDist, err := elevationVsDistanceSVG(track)
	if err != nil {
		return resultView{}, err
	}
	eleTime, err := elevationVsTimeSVG(track)
	if err != nil {
		return resultView{}, err
	}
	spd, err := speedSVG(res)
	if err != nil {
		return resultView{}, err
	}
	// The SVG is generated by our charting library, not user input, so marking
	// it as trusted HTML for inlining is safe.
	if eleDist != nil {
		v.ElevationDistanceChart = template.HTML(svgFragment(eleDist)) //nolint:gosec // library-generated SVG
	}
	if eleTime != nil {
		v.ElevationTimeChart = template.HTML(svgFragment(eleTime)) //nolint:gosec // library-generated SVG
	}
	if spd != nil {
		v.SpeedChart = template.HTML(svgFragment(spd)) //nolint:gosec // library-generated SVG
	}
	return v, nil
}

// formatDuration renders a duration as a compact H/M/S string.
func formatDuration(d time.Duration) string {
	total := int(d.Round(time.Second) / time.Second)
	h := total / 3600
	m := (total % 3600) / 60
	sec := total % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, sec)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, sec)
	default:
		return fmt.Sprintf("%ds", sec)
	}
}
