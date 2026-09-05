package api

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/tailor"
)

type SkillGapsHandler struct {
	pool *pgxpool.Pool
}

func NewSkillGapsHandler(pool *pgxpool.Pool) *SkillGapsHandler {
	return &SkillGapsHandler{pool: pool}
}

func (h *SkillGapsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/jobs/{id}/skill-gaps", h.list)
	mux.HandleFunc("POST /api/skill-gaps/{id}/review", h.review)
}

func (h *SkillGapsHandler) list(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	gaps, err := tailor.ListSkillGaps(r.Context(), h.pool, jobID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, gaps)
}

func (h *SkillGapsHandler) review(w http.ResponseWriter, r *http.Request) {
	gapID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid skill gap id", http.StatusBadRequest)
		return
	}
	if err := tailor.MarkGapReviewed(r.Context(), h.pool, gapID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
