package handlers

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "todo-backend/models"
    "todo-backend/storage"
)

type TaskHandler struct {
    store *storage.Storage
}

func NewTaskHandler(store *storage.Storage) *TaskHandler {
    return &TaskHandler{store: store}
}

func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    w.Header().Set("Content-Type", "application/json")

    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }

    switch r.Method {
    case http.MethodPost:
        h.addTask(w, r)
    case http.MethodGet:
        h.getAllTasks(w, r)
    default:
        http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
    }
}

func (h *TaskHandler) addTask(w http.ResponseWriter, r *http.Request) {
    var req models.CreateTaskRequest

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error": "Неверный формат JSON"}`, http.StatusBadRequest)
        return
    }

    if req.Text == "" {
        http.Error(w, `{"error": "Текст задачи обязателен"}`, http.StatusBadRequest)
        return
    }

    if req.Status == "" {
        req.Status = "inbox"
    }

    task := models.Task{
        Text:      req.Text,
        Status:    req.Status,
        CreatedAt: time.Now(),
    }

    savedTask, err := h.store.AddTask(task)
    if err != nil {
        log.Printf("Ошибка сохранения: %v", err)
        http.Error(w, `{"error": "Ошибка сервера"}`, http.StatusInternalServerError)
        return
    }

    log.Printf("Добавлена задача: \"%s\"", savedTask.Text)

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(savedTask)
}

func (h *TaskHandler) getAllTasks(w http.ResponseWriter, r *http.Request) {
    status := r.URL.Query().Get("status")

    tasks := h.store.GetTasks(status)

    response := models.TaskListResponse{
        TaskObjects: tasks,
    }

    log.Printf("📋 Отдано задач: %d (фильтр: %s)", len(tasks), statusLog(status))

    json.NewEncoder(w).Encode(response)
}

func statusLog(status string) string {
    if status == "" {
        return "все"
    }
    return fmt.Sprintf("\"%s\"", status)
}