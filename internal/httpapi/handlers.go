package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gRPCvsREST/internal/todo"
)

type createTodoRequest struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListTodos(w, r)
	case http.MethodPost:
		h.handleCreateTodo(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleTodoByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idPart := strings.TrimPrefix(r.URL.Path, "/todos/")
	if idPart == "" || strings.Contains(idPart, "/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.svc.Get(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	var req createTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	item, err := h.svc.Create(req.Title, req.Done)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) handleListTodos(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryInt(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	offset, err := parseQueryInt(r, "offset")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}
	payloadKB, err := parseQueryInt(r, "payload_kb")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload_kb")
		return
	}
	if payloadKB < 0 {
		writeError(w, http.StatusBadRequest, "invalid payload_kb")
		return
	}

	items, err := h.svc.List(limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if payloadKB > 0 {
		payload := strings.Repeat("a", payloadKB*1024)
		for i := range items {
			items[i].Payload = payload
		}
	}

	writeJSON(w, http.StatusOK, items)
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, todo.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if errors.Is(err, todo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseQueryInt(r *http.Request, key string) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return value, nil
}
