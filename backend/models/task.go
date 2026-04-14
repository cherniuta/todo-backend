package models

import "time"

type Task struct {
    ID        int       `json:"id"`
    Text      string    `json:"text"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"createdAt"`
}

type CreateTaskRequest struct {
    Text   string `json:"text"`
    Status string `json:"status"`
}

type TaskListResponse struct {
    TaskObjects []Task `json:"taskObjects"`
}