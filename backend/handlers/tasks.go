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

type DelayedHandler struct {
	store *storage.Storage
}

type CurrentWaveHandler struct {
	store *storage.Storage
}

type ModeHandler struct {
	store *storage.Storage
}

type normalizedPayload struct {
	Text               string
	Name               string
	Description        string
	ProblemID          int
	ProblemDescription string
	SourceText         string
	Source             string
	ProjectID          int
	ProjectName        string
	Status             string
	DelayUntil         *time.Time
	SelectedAt         *time.Time
	Done               bool
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

func NewDelayedHandler(store *storage.Storage) *DelayedHandler {
	return &DelayedHandler{store: store}
}

func NewCurrentWaveHandler(store *storage.Storage) *CurrentWaveHandler {
	return &CurrentWaveHandler{store: store}
}

func NewModeHandler(store *storage.Storage) *ModeHandler {
	return &ModeHandler{store: store}
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
			Problems:    tasks,
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

	path := strings.TrimPrefix(r.URL.Path, "/api/problems")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			h.addProblem(w, r)
		case http.MethodGet:
			h.getProblems(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		}
		return
	}

	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		return
	}

	id, err := idFromPath(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	h.deleteProblem(w, id)
}

func (h *ProblemHandler) addProblem(w http.ResponseWriter, r *http.Request) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	text := strings.TrimSpace(firstNonEmpty(req.Text, req.Description))
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
		Problems:    tasks,
		Tasks:       tasks,
		TaskObjects: tasks,
	})
}

func (h *ProblemHandler) deleteProblem(w http.ResponseWriter, id int) {
	task, ok, err := h.store.DeleteProblem(id)
	if err != nil {
		log.Printf("failed to delete problem: %v", err)
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
		"problem": task,
	})
}

func (h *ProjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/projects")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			h.addProject(w, r)
		case http.MethodGet:
			h.getProjects(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		}
		return
	}

	h.handleProjectItem(w, r, path)
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

func (h *ProjectHandler) handleProjectItem(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	projectID, err := strconv.Atoi(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPut:
			h.updateProject(w, r, projectID)
		case http.MethodDelete:
			h.deleteProject(w, projectID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		}
		return
	}

	if len(parts) == 4 && parts[1] == "tasks" && parts[3] == "done" {
		taskID, err := strconv.Atoi(parts[2])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
			return
		}
		h.markTaskInProjectDone(w, projectID, taskID)
		return
	}

	if len(parts) == 3 && parts[1] == "tasks" {
		taskID, err := strconv.Atoi(parts[2])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
			return
		}
		h.removeTaskFromProject(w, projectID, taskID)
		return
	}

	if len(parts) == 2 && parts[1] == "tasks" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
			return
		}
		h.addTaskToProject(w, r, projectID)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (h *ProjectHandler) updateProject(w http.ResponseWriter, r *http.Request, id int) {
	project, err := decodeProject(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	updated, ok, err := h.store.UpdateProject(id, project)
	if err != nil {
		log.Printf("failed to update project: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "project was not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *ProjectHandler) deleteProject(w http.ResponseWriter, id int) {
	project, ok, err := h.store.DeleteProject(id)
	if err != nil {
		log.Printf("failed to delete project: %v", err)
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
		"project": project,
	})
}

func (h *ProjectHandler) addTaskToProject(w http.ResponseWriter, r *http.Request, projectID int) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	description := strings.TrimSpace(firstNonEmpty(req.Description, req.Text))
	if description == "" {
		writeError(w, http.StatusBadRequest, "task description is required")
		return
	}

	project, task, ok, err := h.store.AddTaskToProject(projectID, models.Task{
		Text:        description,
		Description: description,
		Status:      storage.StatusActive,
	})
	if err != nil {
		log.Printf("failed to add project task: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "project was not found")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"project": project,
		"task":    task,
	})
}

func (h *ProjectHandler) removeTaskFromProject(w http.ResponseWriter, projectID, taskID int) {
	task, ok, err := h.store.RemoveTaskFromProject(projectID, taskID)
	if err != nil {
		log.Printf("failed to remove project task: %v", err)
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

func (h *ProjectHandler) markTaskInProjectDone(w http.ResponseWriter, projectID, taskID int) {
	project, task, ok, err := h.store.MarkProjectTaskDone(projectID, taskID)
	if err != nil {
		log.Printf("failed to mark project task as done: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"done": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"done":    true,
		"project": project,
		"task":    task,
	})
}

func (h *DelayedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		return
	}

	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.DelayUntil == nil {
		writeError(w, http.StatusBadRequest, "delayUntil must be a valid date")
		return
	}

	description := strings.TrimSpace(firstNonEmpty(req.Description, req.Text))
	if description == "" {
		writeError(w, http.StatusBadRequest, "task description is required")
		return
	}

	task, err := h.store.AddProblem(description, description, strings.TrimSpace(req.SourceText), req.DelayUntil)
	if err != nil {
		log.Printf("failed to save delayed task: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *CurrentWaveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/current-wave")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			h.addTask(w, r)
		case http.MethodGet:
			h.getTasks(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method is not supported")
		}
		return
	}

	id, err := idFromPath(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid current wave task id")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.deleteTask(w, id)
	case http.MethodPut, http.MethodPatch:
		h.updateTask(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
	}
}

func (h *CurrentWaveHandler) addTask(w http.ResponseWriter, r *http.Request) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	description := strings.TrimSpace(firstNonEmpty(req.ProblemDescription, req.Description, req.Text))
	if description == "" {
		writeError(w, http.StatusBadRequest, "task description is required")
		return
	}

	task, err := h.store.AddToCurrentWave(models.Task{
		ID:                 req.ProblemID,
		Text:               description,
		Description:        description,
		ProblemID:          req.ProblemID,
		ProblemDescription: description,
		Source:             strings.TrimSpace(req.Source),
		ProjectID:          req.ProjectID,
		ProjectName:        strings.TrimSpace(req.ProjectName),
		Status:             storage.StatusActive,
		SelectedAt:         req.SelectedAt,
	})
	if err != nil {
		log.Printf("failed to add current wave task: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *CurrentWaveHandler) getTasks(w http.ResponseWriter, r *http.Request) {
	tasks := h.store.GetCurrentWave()
	writeJSON(w, http.StatusOK, tasks)
}

func (h *CurrentWaveHandler) deleteTask(w http.ResponseWriter, id int) {
	task, ok, err := h.store.DeleteCurrentWaveTask(id)
	if err != nil {
		log.Printf("failed to delete current wave task: %v", err)
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

func (h *CurrentWaveHandler) updateTask(w http.ResponseWriter, r *http.Request, id int) {
	req, err := decodePayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Done || req.Status == storage.StatusDone {
		task, ok, err := h.store.MarkCurrentWaveTaskDone(id)
		if err != nil {
			log.Printf("failed to mark current wave task as done: %v", err)
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
			"done":    true,
			"task":    task,
		})
		return
	}

	writeError(w, http.StatusBadRequest, "only done=true update is supported")
}

func (h *ModeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepareResponse(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, models.ModeResponse{Mode: h.store.GetMode()})
	case http.MethodPut:
		h.setMode(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method is not supported")
	}
}

func (h *ModeHandler) setMode(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	mode, err := modeFromRaw(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mode")
		return
	}

	mode, err = h.store.SetMode(mode)
	if err != nil {
		log.Printf("failed to save mode: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, models.ModeResponse{Mode: mode})
}

func decodePayload(r *http.Request) (normalizedPayload, error) {
	var raw struct {
		Text               json.RawMessage `json:"text"`
		Name               string          `json:"name"`
		Description        string          `json:"description"`
		ProblemID          int             `json:"problemId"`
		ProblemDescription string          `json:"problemDescription"`
		SourceText         string          `json:"sourceText"`
		Source             string          `json:"source"`
		ProjectID          int             `json:"projectId"`
		ProjectName        string          `json:"projectName"`
		Status             string          `json:"status"`
		DelayUntil         string          `json:"delayUntil"`
		SelectedAt         string          `json:"selectedAt"`
		Done               bool            `json:"done"`
	}

	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return normalizedPayload{}, err
	}

	result := normalizedPayload{
		Name:               raw.Name,
		Description:        raw.Description,
		ProblemID:          raw.ProblemID,
		ProblemDescription: raw.ProblemDescription,
		SourceText:         raw.SourceText,
		Source:             raw.Source,
		ProjectID:          raw.ProjectID,
		ProjectName:        raw.ProjectName,
		Status:             raw.Status,
		Done:               raw.Done,
	}

	if raw.DelayUntil != "" {
		delayUntil, err := parseTime(raw.DelayUntil)
		if err != nil {
			return normalizedPayload{}, err
		}
		result.DelayUntil = &delayUntil
	}
	if raw.SelectedAt != "" {
		selectedAt, err := parseTime(raw.SelectedAt)
		if err != nil {
			return normalizedPayload{}, err
		}
		result.SelectedAt = &selectedAt
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
		Text               string `json:"text"`
		Name               string `json:"name"`
		Description        string `json:"description"`
		ProblemID          int    `json:"problemId"`
		ProblemDescription string `json:"problemDescription"`
		SourceText         string `json:"sourceText"`
		Source             string `json:"source"`
		ProjectID          int    `json:"projectId"`
		ProjectName        string `json:"projectName"`
		Status             string `json:"status"`
		DelayUntil         string `json:"delayUntil"`
		SelectedAt         string `json:"selectedAt"`
		Done               bool   `json:"done"`
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
	if nested.ProblemID != 0 {
		result.ProblemID = nested.ProblemID
	}
	if nested.ProblemDescription != "" {
		result.ProblemDescription = nested.ProblemDescription
	}
	if nested.SourceText != "" {
		result.SourceText = nested.SourceText
	}
	if nested.Source != "" {
		result.Source = nested.Source
	}
	if nested.ProjectID != 0 {
		result.ProjectID = nested.ProjectID
	}
	if nested.ProjectName != "" {
		result.ProjectName = nested.ProjectName
	}
	if nested.Status != "" {
		result.Status = nested.Status
	}
	if nested.Done {
		result.Done = true
	}
	if nested.DelayUntil != "" {
		delayUntil, err := parseTime(nested.DelayUntil)
		if err != nil {
			return normalizedPayload{}, err
		}
		result.DelayUntil = &delayUntil
	}
	if nested.SelectedAt != "" {
		selectedAt, err := parseTime(nested.SelectedAt)
		if err != nil {
			return normalizedPayload{}, err
		}
		result.SelectedAt = &selectedAt
	}

	return result, nil
}

func decodeProject(r *http.Request) (models.Project, error) {
	var project models.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		return models.Project{}, err
	}
	if project.Tasks == nil {
		project.Tasks = make([]models.Task, 0)
	}
	for i := range project.Tasks {
		if project.Tasks[i].Description == "" {
			project.Tasks[i].Description = project.Tasks[i].Text
		}
		if project.Tasks[i].Text == "" {
			project.Tasks[i].Text = project.Tasks[i].Description
		}
		if project.Tasks[i].Status == "" {
			project.Tasks[i].Status = storage.StatusActive
		}
	}
	return project, nil
}

func idFromPath(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("missing id")
	}
	return strconv.Atoi(parts[0])
}

func modeFromRaw(raw json.RawMessage) (string, error) {
	var mode string
	if err := json.Unmarshal(raw, &mode); err == nil {
		return strings.TrimSpace(mode), nil
	}

	var payload struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Mode), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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
