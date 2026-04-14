package storage

import (
    "encoding/json"
    "os"
    "sync"
    "todo-backend/models"
)

const dataFile = "data/tasks.json"

type Storage struct {
    mu    sync.RWMutex 
    tasks []models.Task
    nextID int
}

func New() (*Storage, error) {
    s := &Storage{
        tasks:  make([]models.Task, 0),
        nextID: 1,
    }

    if err := os.MkdirAll("data", 0755); err != nil {
        return nil, err
    }

    data, err := os.ReadFile(dataFile)
    if err == nil && len(data) > 0 {
        if err := json.Unmarshal(data, &s.tasks); err == nil {
            for _, t := range s.tasks {
                if t.ID >= s.nextID {
                    s.nextID = t.ID + 1
                }
            }
        }
    }

    return s, nil
}

func (s *Storage) saveToFile() error {
    data, err := json.MarshalIndent(s.tasks, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(dataFile, data, 0644)
}

func (s *Storage) AddTask(task models.Task) (models.Task, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    task.ID = s.nextID
    s.nextID++

    s.tasks = append(s.tasks, task)

    if err := s.saveToFile(); err != nil {
        return models.Task{}, err
    }

    return task, nil
}

func (s *Storage) GetTasks(status string) []models.Task {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if status == "" {
        result := make([]models.Task, len(s.tasks))
        copy(result, s.tasks)
        return result
    }

    var filtered []models.Task
    for _, t := range s.tasks {
        if t.Status == status {
            filtered = append(filtered, t)
        }
    }

    if filtered == nil {
        filtered = make([]models.Task, 0)
    }

    return filtered
}