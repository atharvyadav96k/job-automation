package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"job-automation/app/internal/jobs"
)

type JobsHandler struct {
	store *jobs.Store
}

func NewJobsHandler(store *jobs.Store) *JobsHandler {
	return &JobsHandler{store: store}
}

func (h *JobsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/jobs", h.list)
	mux.HandleFunc("GET /api/jobs/{id}", h.get)
	mux.HandleFunc("POST /api/jobs/{id}/resume-versions/{versionId}/approve", h.approve)
	mux.HandleFunc("POST /api/jobs/{id}/reject", h.reject)
	mux.HandleFunc("POST /api/jobs/{id}/mark-applied", h.markApplied)
	mux.HandleFunc("POST /api/applications/{id}/reply", h.recordReply)
}

func (h *JobsHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *JobsHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	detail, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *JobsHandler) approve(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	versionID, err := strconv.ParseInt(r.PathValue("versionId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid version id", http.StatusBadRequest)
		return
	}
	if err := h.store.Approve(r.Context(), jobID, versionID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobsHandler) reject(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	if err := h.store.Reject(r.Context(), jobID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobsHandler) markApplied(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	app, err := h.store.MarkApplied(r.Context(), jobID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

type replyRequest struct {
	Channel string `json:"channel"`
	Outcome string `json:"outcome"`
}

// recordReply is the "I got a reply" button — the account owner reports
// where a reply came from, there's no inbox integration doing this
// automatically.
func (h *JobsHandler) recordReply(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid application id", http.StatusBadRequest)
		return
	}
	var req replyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.store.RecordReply(r.Context(), appID, req.Channel, req.Outcome); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
