package httpapi

import (
	"math"
	"net/http"
	"strconv"
)

// bandPreview answers what a proposed pair of bands would do to the thoughts
// this installation already holds.
//
// The two numbers are standard deviations above the mean of a space's own
// scores, so their effect depends on the corpus. Before this, the only way to
// find out was to save them — and saving them changes everyone's canvas at
// once, with the person who changed it finding out last.
//
// Read-only in every sense: nothing is written, and the settings the comparison
// is made against are the ones in force right now.
func (s *Server) bandPreview(w http.ResponseWriter, r *http.Request) {
	current := s.Store.IntelligenceSettings(r.Context())
	related, ok := bandParam(r.URL.Query().Get("related_band"), current.RelatedBand)
	if !ok {
		writeError(w, 400, "관련 생각 기준은 0에서 4 사이의 숫자여야 합니다.")
		return
	}
	cluster, ok := bandParam(r.URL.Query().Get("cluster_band"), current.ClusterBand)
	if !ok {
		writeError(w, 400, "묶음 기준은 0에서 4 사이의 숫자여야 합니다.")
		return
	}
	preview, err := s.Store.PreviewBands(r.Context(), related, cluster)
	if err != nil {
		writeError(w, 500, "지금 데이터로 미리 보지 못했습니다.")
		return
	}
	writeJSON(w, 200, preview)
}

// bandParam refuses what the settings themselves refuse, rather than quietly
// substituting the default: a preview of a value the server replaced would
// describe a setting nobody asked for.
func bandParam(raw string, fallback float64) (float64, bool) {
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || value < 0 || value > 4 {
		return 0, false
	}
	return value, true
}
