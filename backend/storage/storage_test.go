package storage

import (
	"os"
	"testing"
	"time"
	"todo-backend/models"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})

	store, err := New()
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestInboxLifecycle(t *testing.T) {
	store := newTestStorage(t)

	first, err := store.AddInboxItem("first inbox item")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.AddInboxItem("second inbox item")
	if err != nil {
		t.Fatal(err)
	}

	next, ok := store.GetNextInboxItem()
	if !ok {
		t.Fatal("expected next inbox item")
	}
	if next.ID != first.ID {
		t.Fatalf("expected first item id %d, got %d", first.ID, next.ID)
	}

	deleted, ok, err := store.DeleteInboxItem(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || deleted.ID != first.ID {
		t.Fatalf("expected deleted item id %d, got %#v", first.ID, deleted)
	}

	next, ok = store.GetNextInboxItem()
	if !ok {
		t.Fatal("expected second inbox item")
	}
	if next.ID != second.ID {
		t.Fatalf("expected second item id %d, got %d", second.ID, next.ID)
	}

	count, err := store.ClearInbox()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected to clear 1 item, got %d", count)
	}
}

func TestDelayedTasksAreActivatedOnRead(t *testing.T) {
	store := newTestStorage(t)

	past := time.Now().Add(-time.Hour)
	task, err := store.AddProblem("delayed task", "", "", &past)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusActive {
		t.Fatalf("past delayed task should be active immediately, got %q", task.Status)
	}

	future := time.Now().Add(time.Hour)
	task, err = store.AddProblem("future task", "", "", &future)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusDelayed {
		t.Fatalf("future task should be delayed, got %q", task.Status)
	}

	activeTasks, err := store.GetProblems(StatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if len(activeTasks) != 1 {
		t.Fatalf("expected 1 active task, got %d", len(activeTasks))
	}
}

func TestDelayInboxItemMovesItToTaskPool(t *testing.T) {
	store := newTestStorage(t)

	item, err := store.AddInboxItem("call the client")
	if err != nil {
		t.Fatal(err)
	}

	delayUntil := time.Now().Add(time.Hour)
	task, ok, err := store.DelayInboxItem(item.ID, delayUntil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected inbox item to be delayed")
	}
	if task.Status != StatusDelayed {
		t.Fatalf("expected delayed status, got %q", task.Status)
	}
	if len(store.GetInboxItems()) != 0 {
		t.Fatal("expected inbox to be empty after delaying item")
	}
}

func TestProjectsLifecycle(t *testing.T) {
	store := newTestStorage(t)

	project, err := store.AddProject("Demo", "Prepare demo", "inbox source")
	if err != nil {
		t.Fatal(err)
	}

	projects := store.GetProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].ID != project.ID || projects[0].Name != "Demo" {
		t.Fatalf("unexpected project: %#v", projects[0])
	}
}

func TestProjectTasksAndDeletionLifecycle(t *testing.T) {
	store := newTestStorage(t)

	project, err := store.AddProject("Demo", "Prepare demo", "")
	if err != nil {
		t.Fatal(err)
	}

	_, task, ok, err := store.AddTaskToProject(project.ID, TaskForTest("project task"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected task to be added to project")
	}

	projects := store.GetProjects()
	if len(projects[0].Tasks) != 1 {
		t.Fatalf("expected 1 project task, got %d", len(projects[0].Tasks))
	}

	removed, ok, err := store.RemoveTaskFromProject(project.ID, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || removed.ID != task.ID {
		t.Fatalf("expected removed task id %d, got %#v", task.ID, removed)
	}

	deletedProject, ok, err := store.DeleteProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || deletedProject.ID != project.ID {
		t.Fatalf("expected deleted project id %d, got %#v", project.ID, deletedProject)
	}
}

func TestProjectTaskCanBeMarkedDone(t *testing.T) {
	store := newTestStorage(t)

	project, err := store.AddProject("Demo", "Prepare demo", "")
	if err != nil {
		t.Fatal(err)
	}

	_, task, ok, err := store.AddTaskToProject(project.ID, TaskForTest("project task"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected task to be added to project")
	}

	_, doneTask, ok, err := store.MarkProjectTaskDone(project.ID, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || doneTask.Status != StatusDone || !doneTask.Done {
		t.Fatalf("expected done project task, got %#v", doneTask)
	}

	projects := store.GetProjects()
	if projects[0].Tasks[0].Status != StatusDone || !projects[0].Tasks[0].Done {
		t.Fatalf("expected done task to be persisted, got %#v", projects[0].Tasks[0])
	}
}

func TestProblemsCanBeDeleted(t *testing.T) {
	store := newTestStorage(t)

	task, err := store.AddProblem("", "standalone problem", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if task.Description != "standalone problem" || task.Text != "standalone problem" {
		t.Fatalf("unexpected problem: %#v", task)
	}

	deleted, ok, err := store.DeleteProblem(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || deleted.ID != task.ID {
		t.Fatalf("expected deleted problem id %d, got %#v", task.ID, deleted)
	}
}

func TestCurrentWaveLifecycle(t *testing.T) {
	store := newTestStorage(t)

	task := TaskForTest("wave task")
	task.ID = 42
	task.ProblemID = 42
	savedTask, err := store.AddToCurrentWave(task)
	if err != nil {
		t.Fatal(err)
	}
	if savedTask.ID != 42 || savedTask.ProblemID != 42 {
		t.Fatalf("expected source problem id to be preserved, got %#v", savedTask)
	}

	tasks := store.GetCurrentWave()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 current wave task, got %d", len(tasks))
	}

	doneTask, ok, err := store.MarkCurrentWaveTaskDone(task.ProblemID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !doneTask.Done || doneTask.Status != StatusDone {
		t.Fatalf("expected done task, got %#v", doneTask)
	}
	if len(store.GetCurrentWave()) != 0 {
		t.Fatal("expected done task to be removed from current wave")
	}
}

func TestModeLifecycle(t *testing.T) {
	store := newTestStorage(t)

	mode, err := store.SetMode("projects")
	if err != nil {
		t.Fatal(err)
	}
	if mode != "projects" || store.GetMode() != "projects" {
		t.Fatalf("expected projects mode, got %q", store.GetMode())
	}
}

func TaskForTest(description string) models.Task {
	return models.Task{
		Text:        description,
		Description: description,
		Status:      StatusActive,
	}
}
