package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"job-automation/app/internal/profile"
)

type ProfileHandler struct {
	store     *profile.Store
	resumeDir string
}

func NewProfileHandler(store *profile.Store, resumeDir string) *ProfileHandler {
	return &ProfileHandler{store: store, resumeDir: resumeDir}
}

func (h *ProfileHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/profile", h.getProfile)
	mux.HandleFunc("PUT /api/profile", h.putBasics)
	mux.HandleFunc("POST /api/profile/resume", h.uploadResume)

	mux.HandleFunc("POST /api/profile/{field}", h.addItem)
	mux.HandleFunc("PUT /api/profile/{field}/{id}", h.updateItem)
	mux.HandleFunc("DELETE /api/profile/{field}/{id}", h.deleteItem)
}

func (h *ProfileHandler) getProfile(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.Get(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProfileHandler) putBasics(w http.ResponseWriter, r *http.Request) {
	var b profile.Basics
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.store.UpdateBasics(r.Context(), b); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProfileHandler) addItem(w http.ResponseWriter, r *http.Request) {
	field := r.PathValue("field")
	if !profile.ValidField(field) {
		http.Error(w, "unknown field", http.StatusNotFound)
		return
	}
	var item map[string]any
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	saved, err := h.store.AddItem(r.Context(), field, item)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (h *ProfileHandler) updateItem(w http.ResponseWriter, r *http.Request) {
	field := r.PathValue("field")
	id := r.PathValue("id")
	if !profile.ValidField(field) {
		http.Error(w, "unknown field", http.StatusNotFound)
		return
	}
	var item map[string]any
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.store.UpdateItem(r.Context(), field, id, item); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProfileHandler) deleteItem(w http.ResponseWriter, r *http.Request) {
	field := r.PathValue("field")
	id := r.PathValue("id")
	if !profile.ValidField(field) {
		http.Error(w, "unknown field", http.StatusNotFound)
		return
	}
	if err := h.store.DeleteItem(r.Context(), field, id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const maxResumeUploadBytes = 10 << 20 // 10MB

func (h *ProfileHandler) uploadResume(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxResumeUploadBytes)
	if err := r.ParseMultipartForm(maxResumeUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	file, header, err := r.FormFile("resume")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	if !strings.EqualFold(filepath.Ext(header.Filename), ".docx") {
		http.Error(w, "only .docx files are accepted", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(h.resumeDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	destPath := filepath.Join(h.resumeDir, "base_resume.docx")
	dest, err := os.Create(destPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.store.SetResumePath(r.Context(), destPath); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": destPath})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf("%v", err)})
}
