package models

import "time"

type Task struct {
	ID          int        `json:"id"`
	Text        string     `json:"text"`
	Description string     `json:"description,omitempty"`
	SourceText  string     `json:"sourceText,omitempty"`
	Status      string     `json:"status"`
	DelayUntil  *time.Time `json:"delayUntil,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
}

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SourceText  string    `json:"sourceText,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type TaskListResponse struct {
	TaskObjects []Task `json:"taskObjects"`
}

type ProblemListResponse struct {
	Tasks       []Task `json:"tasks"`
	TaskObjects []Task `json:"taskObjects"`
}

type ProjectListResponse struct {
	Projects []Project `json:"projects"`
}
