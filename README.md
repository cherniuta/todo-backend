# todo-backend

Бекенд для Effective Planning. Данные хранятся в JSON-файлах в `backend/data`, API работает на:

```text
http://localhost:3000/api
```

## Запуск

```powershell
cd M:\crsr\pr1\todo-backend\backend
go run .
```

Тесты:

```powershell
cd M:\crsr\pr1\todo-backend\backend
go test ./...
```

## Файлы данных

| Файл | Что хранит |
| --- | --- |
| `inbox.json` | сырые дела из inbox |
| `tasks.json` | отдельные задачи/problems |
| `projects.json` | проекты и задачи внутри проектов |
| `current_wave.json` | задачи, выбранные в текущую волну |
| `mode.json` | текущий режим интерфейса |

## Current Wave

Текущая волна принимает задачи двух типов.

Задача из проекта:

```json
{
  "projectId": 1,
  "projectName": "Проект",
  "problemId": 12,
  "problemDescription": "Сделать задачу проекта",
  "selectedAt": "2026-05-15T10:00:00Z"
}
```

Обычная задача без проекта:

```json
{
  "problemId": 34,
  "problemDescription": "Сделать отдельную задачу",
  "selectedAt": "2026-05-15T10:00:00Z"
}
```

### `POST /api/current-wave`

Добавляет задачу в `current_wave.json`. Поле `id` в записи текущей волны равно исходному `problemId`, поэтому дальше задачу можно удалять из волны по тому же id.

Ответ:

```json
{
  "id": 34,
  "problemId": 34,
  "problemDescription": "Сделать отдельную задачу",
  "description": "Сделать отдельную задачу",
  "status": "active",
  "selectedAt": "2026-05-15T10:00:00Z",
  "createdAt": "2026-05-15T10:00:01Z",
  "updatedAt": "2026-05-15T10:00:01Z"
}
```

### `GET /api/current-wave`

Возвращает массив задач, не объект:

```json
[
  {
    "id": 12,
    "problemId": 12,
    "projectId": 1,
    "projectName": "Проект",
    "problemDescription": "Сделать задачу проекта",
    "description": "Сделать задачу проекта",
    "status": "active",
    "selectedAt": "2026-05-15T10:00:00Z"
  },
  {
    "id": 34,
    "problemId": 34,
    "problemDescription": "Сделать отдельную задачу",
    "description": "Сделать отдельную задачу",
    "status": "active",
    "selectedAt": "2026-05-15T10:00:00Z"
  }
]
```

### `DELETE /api/current-wave/{problemId}`

Удаляет задачу из текущей волны по исходному `problemId`.

Ответ:

```json
{
  "deleted": true,
  "task": {
    "id": 34,
    "problemId": 34,
    "description": "Сделать отдельную задачу"
  }
}
```

### `PUT /api/current-wave/{problemId}`

Отмечает задачу текущей волны сделанной и удаляет ее из `current_wave.json`.

Тело запроса:

```json
{
  "done": true
}
```

Также поддерживается:

```json
{
  "status": "done"
}
```

## Projects

### `GET /api/projects`

Возвращает проекты:

```json
{
  "projects": [
    {
      "id": 1,
      "name": "Проект",
      "tasks": [
        {
          "id": 12,
          "description": "Сделать задачу проекта",
          "status": "active"
        }
      ]
    }
  ]
}
```

### `PUT /api/projects/{projectId}/tasks/{problemId}/done`

Отмечает задачу проекта выполненной в `projects.json`: у задачи ставятся `status: "done"` и `done: true`.

Ответ:

```json
{
  "done": true,
  "project": {
    "id": 1,
    "name": "Проект"
  },
  "task": {
    "id": 12,
    "description": "Сделать задачу проекта",
    "status": "done",
    "done": true
  }
}
```

### `DELETE /api/projects/{projectId}/tasks/{problemId}`

Удаляет задачу из проекта. Ручка оставлена для старого сценария, где выбранная задача сразу переносилась из проекта в текущую волну.

## Problems

### `GET /api/problems`

Возвращает отдельные задачи в объекте:

```json
{
  "problems": [
    {
      "id": 34,
      "description": "Сделать отдельную задачу",
      "status": "active"
    }
  ]
}
```

### `DELETE /api/problems/{problemId}`

Удаляет отдельную задачу из `tasks.json`. Это можно вызывать после добавления обычной задачи в текущую волну.

Ответ:

```json
{
  "deleted": true,
  "problem": {
    "id": 34,
    "description": "Сделать отдельную задачу"
  }
}
```

## Остальные ручки

Бекенд также поддерживает существующие ручки:

| Метод и путь | Назначение |
| --- | --- |
| `POST /api/tasks` | добавить дело в inbox |
| `GET /api/tasks?status=inbox` | получить inbox в поле `taskObjects` |
| `DELETE /api/tasks/{id}` | удалить дело из inbox |
| `POST /api/problems` | создать отдельную задачу |
| `POST /api/projects` | создать проект |
| `PUT /api/projects/{projectId}` | обновить проект целиком |
| `POST /api/projects/{projectId}/tasks` | добавить задачу в проект |
| `POST /api/delayed` | создать отложенную задачу |
| `GET /api/mode` | получить режим интерфейса |
| `PUT /api/mode` | обновить режим интерфейса |

Все API-ручки отвечают с CORS-заголовками и поддерживают `OPTIONS`.
