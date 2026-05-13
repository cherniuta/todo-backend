package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"todo-backend/storage"
)

func newTestStore(t *testing.T) *storage.Storage {
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

	store, err := storage.New()
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestProjectHandlerAcceptsFrontendNestedPayload(t *testing.T) {
	store := newTestStore(t)
	handler := NewProjectHandler(store)

	body := []byte(`{
		"text": {
			"name": "Demo",
			"description": "Prepare next demo",
			"sourceText": "Inbox source"
		},
		"status": "inbox"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SourceText  string `json:"sourceText"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Name != "Demo" || response.Description != "Prepare next demo" || response.SourceText != "Inbox source" {
		t.Fatalf("unexpected project response: %#v", response)
	}
}

func TestProblemHandlerAcceptsFrontendNestedPayload(t *testing.T) {
	store := newTestStore(t)
	handler := NewProblemHandler(store)

	body := []byte(`{
		"text": {
			"text": "Write tests",
			"description": "Cover backend demo flow",
			"sourceText": "Inbox source",
			"status": "inbox"
		},
		"status": "inbox"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/problems", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response struct {
		Text        string `json:"text"`
		Description string `json:"description"`
		SourceText  string `json:"sourceText"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Text != "Write tests" || response.Description != "Cover backend demo flow" || response.SourceText != "Inbox source" {
		t.Fatalf("unexpected task response: %#v", response)
	}
	if response.Status != storage.StatusActive {
		t.Fatalf("expected active status, got %q", response.Status)
	}
}

func TestDeletingAlreadyMovedInboxItemIsSuccessful(t *testing.T) {
	store := newTestStore(t)
	handler := NewTaskHandler(store)

	item, err := store.AddInboxItem("later")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"delayUntil":"2099-01-01T10:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/1/delay", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected delay status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/1", nil)
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected repeated delete status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Deleted {
		t.Fatalf("expected deleted=false for already moved inbox item %d", item.ID)
	}
}

func TestProblemsResponseMatchesFrontendContract(t *testing.T) {
	store := newTestStore(t)
	handler := NewProblemHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/problems", bytes.NewReader([]byte(`{"description":"Sort inbox result"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/problems", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Problems []struct {
			Description string `json:"description"`
		} `json:"problems"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Problems) != 1 || response.Problems[0].Description != "Sort inbox result" {
		t.Fatalf("unexpected problems response: %#v", response)
	}
}

func TestProjectUpdateAndRemoveTaskRoutes(t *testing.T) {
	store := newTestStore(t)
	handler := NewProjectHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte(`{"name":"Demo","description":"Before"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected project create status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/projects/1", bytes.NewReader([]byte(`{
		"name":"Demo updated",
		"description":"After",
		"tasks":[{"id":123,"description":"Project task","createdAt":"2026-05-13T10:00:00Z"}]
	}`)))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project update status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/projects/1/tasks/123", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project task delete status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestCurrentWaveAddAndDoneRoutes(t *testing.T) {
	store := newTestStore(t)
	handler := NewCurrentWaveHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/current-wave", bytes.NewReader([]byte(`{
		"description":"Do selected task",
		"source":"project",
		"projectId":1,
		"projectName":"Demo"
	}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected wave create status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/current-wave", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected wave get status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var listResponse struct {
		Tasks []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			ProjectName string `json:"projectName"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResponse); err != nil {
		t.Fatal(err)
	}
	if len(listResponse.Tasks) != 1 || listResponse.Tasks[0].ProjectName != "Demo" {
		t.Fatalf("unexpected current wave response: %#v", listResponse)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/current-wave/1", bytes.NewReader([]byte(`{"done":true}`)))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected wave done status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	if len(store.GetCurrentWave()) != 0 {
		t.Fatal("expected done task to be removed from current wave")
	}
}

func TestModeAcceptsFrontendStringPayload(t *testing.T) {
	store := newTestStore(t)
	handler := NewModeHandler(store)

	req := httptest.NewRequest(http.MethodPut, "/api/mode", bytes.NewReader([]byte(`"projects"`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected mode update status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/mode", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected mode get status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Mode != "projects" {
		t.Fatalf("expected projects mode, got %q", response.Mode)
	}
}
