package models

import "time"

type Task struct {
	ID                 int        `json:"id"`
	Text               string     `json:"text,omitempty"`
	Description        string     `json:"description,omitempty"`
	ProblemID          int        `json:"problemId,omitempty"`
	ProblemDescription string     `json:"problemDescription,omitempty"`
	SourceText         string     `json:"sourceText,omitempty"`
	Source             string     `json:"source,omitempty"`
	ProjectID          int        `json:"projectId,omitempty"`
	ProjectName        string     `json:"projectName,omitempty"`
	Status             string     `json:"status"`
	DelayUntil         *time.Time `json:"delayUntil,omitempty"`
	SelectedAt         *time.Time `json:"selectedAt,omitempty"`
	Done               bool       `json:"done,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt,omitempty"`
}

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SourceText  string    `json:"sourceText,omitempty"`
	Tasks       []Task    `json:"tasks"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

type TaskListResponse struct {
	TaskObjects []Task `json:"taskObjects"`
}

type ProblemListResponse struct {
	Problems    []Task `json:"problems"`
	Tasks       []Task `json:"tasks"`
	TaskObjects []Task `json:"taskObjects"`
}

type ProjectListResponse struct {
	Projects []Project `json:"projects"`
}

type CurrentWaveResponse struct {
	Tasks []Task `json:"tasks"`
	Wave  []Task `json:"wave"`
}

type ModeResponse struct {
	Mode string `json:"mode"`
}
