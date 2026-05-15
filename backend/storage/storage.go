package storage

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
	"todo-backend/models"
)

const (
	dataDir      = "data"
	inboxFile    = "data/inbox.json"
	tasksFile    = "data/tasks.json"
	projectsFile = "data/projects.json"
	waveFile     = "data/current_wave.json"
	modeFile     = "data/mode.json"

	StatusInbox   = "inbox"
	StatusActive  = "active"
	StatusDelayed = "delayed"
	StatusDone    = "done"
)

type Storage struct {
	mu            sync.RWMutex
	inbox         []models.Task
	tasks         []models.Task
	projects      []models.Project
	currentWave   []models.Task
	mode          string
	nextInboxID   int
	nextTaskID    int
	nextProjectID int
	nextWaveID    int
}

func New() (*Storage, error) {
	s := &Storage{
		inbox:         make([]models.Task, 0),
		tasks:         make([]models.Task, 0),
		projects:      make([]models.Project, 0),
		currentWave:   make([]models.Task, 0),
		mode:          "inbox",
		nextInboxID:   1,
		nextTaskID:    1,
		nextProjectID: 1,
		nextWaveID:    1,
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	inboxExists, err := loadJSON(inboxFile, &s.inbox)
	if err != nil {
		return nil, err
	}

	tasksExists, err := loadJSON(tasksFile, &s.tasks)
	if err != nil {
		return nil, err
	}

	if _, err := loadJSON(projectsFile, &s.projects); err != nil {
		return nil, err
	}

	if _, err := loadJSON(waveFile, &s.currentWave); err != nil {
		return nil, err
	}

	var mode models.ModeResponse
	if modeExists, err := loadJSON(modeFile, &mode); err != nil {
		return nil, err
	} else if modeExists && mode.Mode != "" {
		s.mode = mode.Mode
	}

	if !inboxExists && tasksExists && looksLikeLegacyInbox(s.tasks) {
		s.inbox = normalizeInbox(s.tasks)
		s.tasks = make([]models.Task, 0)
		if err := s.saveInboxFile(); err != nil {
			return nil, err
		}
		if err := s.saveTasksFile(); err != nil {
			return nil, err
		}
	}

	s.refreshNextIDs()

	return s, nil
}

func loadJSON(path string, target any) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(data) == 0 {
		return true, nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return true, err
	}
	return true, nil
}

func looksLikeLegacyInbox(tasks []models.Task) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, task := range tasks {
		if task.Status != "" && task.Status != StatusInbox {
			return false
		}
	}
	return true
}

func normalizeInbox(tasks []models.Task) []models.Task {
	result := make([]models.Task, len(tasks))
	for i, task := range tasks {
		if task.Status == "" {
			task.Status = StatusInbox
		}
		result[i] = task
	}
	return result
}

func (s *Storage) refreshNextIDs() {
	for _, item := range s.inbox {
		if item.ID >= s.nextInboxID {
			s.nextInboxID = item.ID + 1
		}
	}
	for _, task := range s.tasks {
		if task.ID >= s.nextTaskID {
			s.nextTaskID = task.ID + 1
		}
	}
	for _, project := range s.projects {
		if project.Tasks == nil {
			project.Tasks = make([]models.Task, 0)
		}
		if project.ID >= s.nextProjectID {
			s.nextProjectID = project.ID + 1
		}
		for _, task := range project.Tasks {
			if task.ID >= s.nextTaskID {
				s.nextTaskID = task.ID + 1
			}
		}
	}
	for _, task := range s.currentWave {
		if task.ID >= s.nextWaveID {
			s.nextWaveID = task.ID + 1
		}
	}
}

func (s *Storage) saveInboxFile() error {
	return writeJSON(inboxFile, s.inbox)
}

func (s *Storage) saveTasksFile() error {
	return writeJSON(tasksFile, s.tasks)
}

func (s *Storage) saveProjectsFile() error {
	return writeJSON(projectsFile, s.projects)
}

func (s *Storage) saveWaveFile() error {
	return writeJSON(waveFile, s.currentWave)
}

func (s *Storage) saveModeFile() error {
	return writeJSON(modeFile, models.ModeResponse{Mode: s.mode})
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Storage) AddInboxItem(text string) (models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	item := models.Task{
		ID:        s.nextInboxID,
		Text:      text,
		Status:    StatusInbox,
		CreatedAt: now,
	}
	s.nextInboxID++
	s.inbox = append(s.inbox, item)

	if err := s.saveInboxFile(); err != nil {
		return models.Task{}, err
	}

	return item, nil
}

func (s *Storage) GetInboxItems() []models.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Task, len(s.inbox))
	copy(result, s.inbox)
	return result
}

func (s *Storage) GetNextInboxItem() (models.Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.inbox) == 0 {
		return models.Task{}, false
	}

	return s.inbox[0], true
}

func (s *Storage) DeleteInboxItem(id int) (models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.inbox {
		if item.ID == id {
			s.inbox = append(s.inbox[:i], s.inbox[i+1:]...)
			if err := s.saveInboxFile(); err != nil {
				return models.Task{}, false, err
			}
			return item, true, nil
		}
	}

	return models.Task{}, false, nil
}

func (s *Storage) ClearInbox() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := len(s.inbox)
	s.inbox = make([]models.Task, 0)
	if err := s.saveInboxFile(); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Storage) AddProblem(text, description, sourceText string, delayUntil *time.Time) (models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	text = firstNonEmpty(text, description)
	description = firstNonEmpty(description, text)

	now := time.Now()
	status := StatusActive
	if delayUntil != nil && delayUntil.After(now) {
		status = StatusDelayed
	}

	task := models.Task{
		ID:          s.nextTaskID,
		Text:        text,
		Description: description,
		SourceText:  sourceText,
		Status:      status,
		DelayUntil:  delayUntil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.nextTaskID++
	s.tasks = append(s.tasks, task)

	if err := s.saveTasksFile(); err != nil {
		return models.Task{}, err
	}

	return task, nil
}

func (s *Storage) DeleteProblem(id int) (models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.tasks {
		if task.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			if err := s.saveTasksFile(); err != nil {
				return models.Task{}, false, err
			}
			return task, true, nil
		}
	}

	return models.Task{}, false, nil
}

func (s *Storage) DelayInboxItem(id int, delayUntil time.Time) (models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.inbox {
		if item.ID == id {
			s.inbox = append(s.inbox[:i], s.inbox[i+1:]...)

			now := time.Now()
			task := models.Task{
				ID:          s.nextTaskID,
				Text:        item.Text,
				Description: item.Text,
				SourceText:  item.Text,
				Status:      StatusDelayed,
				DelayUntil:  &delayUntil,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if !delayUntil.After(now) {
				task.Status = StatusActive
			}

			s.nextTaskID++
			s.tasks = append(s.tasks, task)

			if err := s.saveInboxFile(); err != nil {
				return models.Task{}, false, err
			}
			if err := s.saveTasksFile(); err != nil {
				return models.Task{}, false, err
			}

			return task, true, nil
		}
	}

	return models.Task{}, false, nil
}

func (s *Storage) GetProblems(status string) ([]models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if changed := s.activateDelayedTasksLocked(time.Now()); changed {
		if err := s.saveTasksFile(); err != nil {
			return nil, err
		}
	}

	if status == "" || status == "all" {
		result := make([]models.Task, len(s.tasks))
		copy(result, s.tasks)
		return result, nil
	}

	filtered := make([]models.Task, 0)
	for _, task := range s.tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}

	return filtered, nil
}

func (s *Storage) activateDelayedTasksLocked(now time.Time) bool {
	changed := false
	for i := range s.tasks {
		task := &s.tasks[i]
		if task.Status == StatusDelayed && task.DelayUntil != nil && !task.DelayUntil.After(now) {
			task.Status = StatusActive
			task.UpdatedAt = now
			changed = true
		}
	}
	return changed
}

func (s *Storage) AddProject(name, description, sourceText string) (models.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project := models.Project{
		ID:          s.nextProjectID,
		Name:        name,
		Description: description,
		SourceText:  sourceText,
		Tasks:       make([]models.Task, 0),
		CreatedAt:   time.Now(),
	}
	s.nextProjectID++
	s.projects = append(s.projects, project)

	if err := s.saveProjectsFile(); err != nil {
		return models.Project{}, err
	}

	return project, nil
}

func (s *Storage) GetProjects() []models.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Project, len(s.projects))
	copy(result, s.projects)
	for i := range result {
		if result[i].Tasks == nil {
			result[i].Tasks = make([]models.Task, 0)
		}
	}
	return result
}

func (s *Storage) UpdateProject(id int, updated models.Project) (models.Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, project := range s.projects {
		if project.ID == id {
			updated.ID = id
			if updated.Name == "" {
				updated.Name = project.Name
			}
			if updated.Tasks == nil {
				updated.Tasks = make([]models.Task, 0)
			}
			updated.CreatedAt = project.CreatedAt
			updated.UpdatedAt = time.Now()

			for taskIndex := range updated.Tasks {
				if updated.Tasks[taskIndex].ID == 0 {
					updated.Tasks[taskIndex].ID = s.nextTaskID
					s.nextTaskID++
				}
				if updated.Tasks[taskIndex].CreatedAt.IsZero() {
					updated.Tasks[taskIndex].CreatedAt = time.Now()
				}
				if updated.Tasks[taskIndex].Description != "" && updated.Tasks[taskIndex].Text == "" {
					updated.Tasks[taskIndex].Text = updated.Tasks[taskIndex].Description
				}
				if updated.Tasks[taskIndex].Status == "" {
					updated.Tasks[taskIndex].Status = StatusActive
				}
			}

			s.projects[i] = updated
			if err := s.saveProjectsFile(); err != nil {
				return models.Project{}, false, err
			}
			return updated, true, nil
		}
	}

	return models.Project{}, false, nil
}

func (s *Storage) DeleteProject(id int) (models.Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, project := range s.projects {
		if project.ID == id {
			s.projects = append(s.projects[:i], s.projects[i+1:]...)
			if err := s.saveProjectsFile(); err != nil {
				return models.Project{}, false, err
			}
			return project, true, nil
		}
	}

	return models.Project{}, false, nil
}

func (s *Storage) AddTaskToProject(projectID int, task models.Task) (models.Project, models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.projects {
		if s.projects[i].ID == projectID {
			now := time.Now()
			if task.ID == 0 {
				task.ID = s.nextTaskID
				s.nextTaskID++
			}
			if task.Description == "" {
				task.Description = task.Text
			}
			if task.Text == "" {
				task.Text = task.Description
			}
			if task.Status == "" {
				task.Status = StatusActive
			}
			if task.CreatedAt.IsZero() {
				task.CreatedAt = now
			}
			task.UpdatedAt = now

			s.projects[i].Tasks = append(s.projects[i].Tasks, task)
			s.projects[i].UpdatedAt = now
			if err := s.saveProjectsFile(); err != nil {
				return models.Project{}, models.Task{}, false, err
			}
			return s.projects[i], task, true, nil
		}
	}

	return models.Project{}, models.Task{}, false, nil
}

func (s *Storage) RemoveTaskFromProject(projectID, taskID int) (models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for projectIndex := range s.projects {
		if s.projects[projectIndex].ID == projectID {
			for taskIndex, task := range s.projects[projectIndex].Tasks {
				if task.ID == taskID {
					s.projects[projectIndex].Tasks = append(s.projects[projectIndex].Tasks[:taskIndex], s.projects[projectIndex].Tasks[taskIndex+1:]...)
					s.projects[projectIndex].UpdatedAt = time.Now()
					if err := s.saveProjectsFile(); err != nil {
						return models.Task{}, false, err
					}
					return task, true, nil
				}
			}
			return models.Task{}, false, nil
		}
	}

	return models.Task{}, false, nil
}

func (s *Storage) MarkProjectTaskDone(projectID, taskID int) (models.Project, models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for projectIndex := range s.projects {
		if s.projects[projectIndex].ID == projectID {
			for taskIndex := range s.projects[projectIndex].Tasks {
				if s.projects[projectIndex].Tasks[taskIndex].ID == taskID {
					s.projects[projectIndex].Tasks[taskIndex].Status = StatusDone
					s.projects[projectIndex].Tasks[taskIndex].Done = true
					s.projects[projectIndex].Tasks[taskIndex].UpdatedAt = now
					s.projects[projectIndex].UpdatedAt = now
					if err := s.saveProjectsFile(); err != nil {
						return models.Project{}, models.Task{}, false, err
					}
					return s.projects[projectIndex], s.projects[projectIndex].Tasks[taskIndex], true, nil
				}
			}
			return models.Project{}, models.Task{}, false, nil
		}
	}

	return models.Project{}, models.Task{}, false, nil
}

func (s *Storage) AddToCurrentWave(task models.Task) (models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if task.ID == 0 {
		task.ID = s.nextWaveID
		s.nextWaveID++
	} else if task.ID >= s.nextWaveID {
		s.nextWaveID = task.ID + 1
	}
	if task.ProblemID == 0 {
		task.ProblemID = task.ID
	}
	if task.Description == "" {
		task.Description = task.Text
	}
	if task.Text == "" {
		task.Text = task.Description
	}
	if task.ProblemDescription == "" {
		task.ProblemDescription = task.Description
	}
	if task.Status == "" {
		task.Status = StatusActive
	}
	if task.SelectedAt == nil {
		task.SelectedAt = &now
	}
	task.CreatedAt = now
	task.UpdatedAt = now

	s.currentWave = append(s.currentWave, task)
	if err := s.saveWaveFile(); err != nil {
		return models.Task{}, err
	}

	return task, nil
}

func (s *Storage) GetCurrentWave() []models.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Task, len(s.currentWave))
	copy(result, s.currentWave)
	return result
}

func (s *Storage) DeleteCurrentWaveTask(id int) (models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.currentWave {
		if task.ID == id || task.ProblemID == id {
			s.currentWave = append(s.currentWave[:i], s.currentWave[i+1:]...)
			if err := s.saveWaveFile(); err != nil {
				return models.Task{}, false, err
			}
			return task, true, nil
		}
	}

	return models.Task{}, false, nil
}

func (s *Storage) MarkCurrentWaveTaskDone(id int) (models.Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.currentWave {
		if task.ID == id || task.ProblemID == id {
			task.Done = true
			task.Status = StatusDone
			task.UpdatedAt = time.Now()
			s.currentWave = append(s.currentWave[:i], s.currentWave[i+1:]...)
			if err := s.saveWaveFile(); err != nil {
				return models.Task{}, false, err
			}
			return task, true, nil
		}
	}

	return models.Task{}, false, nil
}

func (s *Storage) GetMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.mode == "" {
		return "inbox"
	}
	return s.mode
}

func (s *Storage) SetMode(mode string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode = firstNonEmpty(mode, "inbox")
	s.mode = mode
	if err := s.saveModeFile(); err != nil {
		return "", err
	}

	return s.mode, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
