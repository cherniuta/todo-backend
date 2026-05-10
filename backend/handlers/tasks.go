package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"todo-backend/models"
	"todo-backend/storage"
)

type TaskHandler struct {
	store *storage.Storage
}

type ProblemHandler struct {
	store *storage.Storage
}

type ProjectHandler struct {
	store *storage.Storage
}

type normalizedPayload struct {
	Text        string
	Name        string
	Description string
	SourceText  string
	Status      string
	DelayUntil  *time.Time
}

func NewTaskHandler(store *storage.Storage) *TaskHandler {
	return &TaskHandler{store: store}
}

func NewProblemHandler(store *storage.Storage) *ProblemHandler {
	return &ProblemHandler{store: store}
}

func NewProjectHandler(store *storage.Storage) *ProjectHandler {
	return &ProjectHandler{store: store}
}

func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/tasks")
	if strings.HasPrefix(r.URL.Path, "/api/inbox") {
		path = strings.TrimPrefix(r.URL.Path, "/api/inbox")
	}

	switch {
	case path == "" || path == "/":
		h.handleCollection(w, r)
	case path == "/next":
		h.getNextInboxItem(w, r)
	default:
		h.handleItem(w, r, path)
	}
}

func (h *TaskHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.addInboxItem(w, r)
	case http.MethodGet:
		h.getInboxItems(w, r)
	case http.MethodDelete:
		h.clearInbox(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
	}
}

func (h *TaskHandler) addInboxItem(w http.ResponseWriter, r *http.Request) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "task text is required")
		return
	}

	task, err := h.store.AddInboxItem(text)
	if err != nil {
		log.Printf("failed to save inbox item: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	log.Printf("added inbox item: %q", task.Text)
	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) getInboxItems(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != storage.StatusInbox {
		tasks, err := h.store.GetProblems(status)
		if err != nil {
			log.Printf("failed to load tasks: %v", err)
			writeError(w, http.StatusInternalServerError, "server error")
			return
		}
		writeJSON(w, http.StatusOK, models.ProblemListResponse{
			Tasks:       tasks,
			TaskObjects: tasks,
		})
		return
	}

	tasks := h.store.GetInboxItems()
	log.Printf("returned inbox items: %d", len(tasks))
	writeJSON(w, http.StatusOK, models.TaskListResponse{TaskObjects: tasks})
}

func (h *TaskHandler) getNextInboxItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		return
	}

	task, ok := h.store.GetNextInboxItem()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"task": nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task": task,
	})
}

func (h *TaskHandler) handleItem(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
			return
		}
		h.deleteInboxItem(w, id)
		return
	}

	if len(parts) == 2 && parts[1] == "delay" {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
			return
		}
		h.delayInboxItem(w, r, id)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (h *TaskHandler) deleteInboxItem(w http.ResponseWriter, id int) {
	task, ok, err := h.store.DeleteInboxItem(id)
	if err != nil {
		log.Printf("failed to delete inbox item: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
		"task":    task,
	})
}

func (h *TaskHandler) clearInbox(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != storage.StatusInbox {
		writeError(w, http.StatusBadRequest, "only inbox can be cleared through this endpoint")
		return
	}

	count, err := h.store.ClearInbox()
	if err != nil {
		log.Printf("failed to clear inbox: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": count,
	})
}

func (h *TaskHandler) delayInboxItem(w http.ResponseWriter, r *http.Request, id int) {
	var req struct {
		DelayUntil string `json:"delayUntil"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	delayUntil, err := parseTime(req.DelayUntil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "delayUntil must be a valid date")
		return
	}

	task, ok, err := h.store.DelayInboxItem(id, delayUntil)
	if err != nil {
		log.Printf("failed to delay inbox item: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "task was not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *ProblemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.addProblem(w, r)
	case http.MethodGet:
		h.getProblems(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
	}
}

func (h *ProblemHandler) addProblem(w http.ResponseWriter, r *http.Request) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "task text is required")
		return
	}

	task, err := h.store.AddProblem(text, strings.TrimSpace(req.Description), strings.TrimSpace(req.SourceText), req.DelayUntil)
	if err != nil {
		log.Printf("failed to save task: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *ProblemHandler) getProblems(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	tasks, err := h.store.GetProblems(status)
	if err != nil {
		log.Printf("failed to load tasks: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, models.ProblemListResponse{
		Tasks:       tasks,
		TaskObjects: tasks,
	})
}

func (h *ProjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.addProject(w, r)
	case http.MethodGet:
		h.getProjects(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
	}
}

func (h *ProjectHandler) addProject(w http.ResponseWriter, r *http.Request) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.Text)
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "project name is required")
		return
	}

	project, err := h.store.AddProject(name, strings.TrimSpace(req.Description), strings.TrimSpace(req.SourceText))
	if err != nil {
		log.Printf("failed to save project: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (h *ProjectHandler) getProjects(w http.ResponseWriter, r *http.Request) {
	projects := h.store.GetProjects()
	writeJSON(w, http.StatusOK, models.ProjectListResponse{Projects: projects})
}

func decodePayload(r *http.Request) (normalizedPayload, error) {
	var raw struct {
		Text        json.RawMessage `json:"text"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		SourceText  string          `json:"sourceText"`
		Status      string          `json:"status"`
		DelayUntil  string          `json:"delayUntil"`
	}

	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return normalizedPayload{}, err
	}

	result := normalizedPayload{
		Name:        raw.Name,
		Description: raw.Description,
		SourceText:  raw.SourceText,
		Status:      raw.Status,
	}

	if raw.DelayUntil != "" {
		delayUntil, err := parseTime(raw.DelayUntil)
		if err != nil {
			return normalizedPayload{}, err
		}
		result.DelayUntil = &delayUntil
	}

	if len(raw.Text) == 0 || string(raw.Text) == "null" {
		return result, nil
	}

	var text string
	if err := json.Unmarshal(raw.Text, &text); err == nil {
		result.Text = text
		return result, nil
	}

	var nested struct {
		Text        string `json:"text"`
		Name        string `json:"name"`
		Description string `json:"description"`
		SourceText  string `json:"sourceText"`
		Status      string `json:"status"`
		DelayUntil  string `json:"delayUntil"`
	}
	if err := json.Unmarshal(raw.Text, &nested); err != nil {
		return normalizedPayload{}, fmt.Errorf("unsupported text payload: %w", err)
	}

	result.Text = nested.Text
	if nested.Name != "" {
		result.Name = nested.Name
	}
	if nested.Description != "" {
		result.Description = nested.Description
	}
	if nested.SourceText != "" {
		result.SourceText = nested.SourceText
	}
	if nested.Status != "" {
		result.Status = nested.Status
	}
	if nested.DelayUntil != "" {
		delayUntil, err := parseTime(nested.DelayUntil)
		if err != nil {
			return normalizedPayload{}, err
		}
		result.DelayUntil = &delayUntil
	}

	return result, nil
}

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}

	return time.Parse("2006-01-02T15:04", value)
}

func prepareResponse(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
