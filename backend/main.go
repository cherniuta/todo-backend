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
        log.Fatalf("Ошибка инициализации хранилища: %v", err)
    }

    taskHandler := handlers.NewTaskHandler(store)

    http.Handle("/api/tasks", taskHandler)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message": "Сервер работает!"}`))
    })

    port := ":3000"
    log.Printf("Сервер запущен: http://localhost%s", port)
    log.Printf("API доступен:   http://localhost%s/api/tasks", port)

    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatalf("Ошибка запуска сервера: %v", err)
    }
}