package main

import (
	"log"
	"net/http"
	"todo-backend/handlers"
	"todo-backend/storage"
)

func main() {
	store, err := storage.New()
	if err != nil {
		log.Fatalf("storage initialization error: %v", err)
	}

	taskHandler := handlers.NewTaskHandler(store)
	problemHandler := handlers.NewProblemHandler(store)
	projectHandler := handlers.NewProjectHandler(store)

	http.Handle("/api/tasks", taskHandler)
	http.Handle("/api/tasks/", taskHandler)
	http.Handle("/api/inbox", taskHandler)
	http.Handle("/api/inbox/", taskHandler)
	http.Handle("/api/problems", problemHandler)
	http.Handle("/api/projects", projectHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "server is running"}`))
	})

	port := ":3000"
	log.Printf("server started: http://localhost%s", port)
	log.Printf("inbox API:      http://localhost%s/api/tasks", port)
	log.Printf("tasks API:      http://localhost%s/api/problems", port)
	log.Printf("projects API:   http://localhost%s/api/projects", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("server startup error: %v", err)
	}
}
