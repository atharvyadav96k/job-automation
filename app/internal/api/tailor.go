package api

import (
	"net/http"
	"strconv"

	"job-automation/app/internal/tailor"
)

type TailorHandler struct {
	pipeline *tailor.Pipeline
}

func NewTailorHandler(pipeline *tailor.Pipeline) *TailorHandler {
	return &TailorHandler{pipeline: pipeline}
}

func (h *TailorHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/jobs/{id}/tailor", h.tailor)
}

// tailor is triggered on demand rather than automatically for every
// discovered job — tailoring costs an LLM call, discovery can turn up
// dozens of loosely-relevant jobs per run, and the review gate (Phase 3)
// means a human picks which jobs are worth tailoring anyway.
func (h *TailorHandler) tailor(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	result, err := h.pipeline.Run(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
