# Сервис назначения ревьюеров для Pull Request’ов
Микросервис для автоматического назначения ревьюверов на Pull Request'ы внутри команд.

*при разработке я намеренно оставил возможность клиенту генерировать ID пользоваталей и pull request'ов, так как это соответствует предоставленной OpenAPI-спецификации, а также упрощает ручное тестирование и демонстрацию*

## Стек
- **Backend**: Go 1.25.0
- **Database**: PostgreSQL 16.10  
- **HTTP Framework**: Gin
- **Logging**: Zap
- **Configuration**: godotenv
- **Containerization**: Docker + Docker Compose

## Структура проекта
```text
internal/
├── config/     # Конфигурация
├── db/         # Подключение к БД  
├── handler/    # HTTP обработчики
│   └── dto/    # Структуры запросов и ответов
├── logger/     # Создание логера
├── models/     # Структуры данных
├── repository/ # Работа с БД
└── service/    # Бизнес-логика
```
## Быстрый старт

### Docker Compose:
```bash
docker-compose up --build
```
### Makefile:
```bash
make dev          # Запустить сервис
make logs         # Просмотр логов
make db           # Подключиться к БД
make down         # Остановить сервис
```
Сервис будет доступен на http://localhost:8080

## Структура базы данных
### Схема БД
```sql
teams (team_name PK)
  │
users (user_id PK, team_name FK, is_active)
  │
pull_requests (pull_request_id PK, author_id FK, status)
  │
pull_request_reviewers (pull_request_id FK, user_id FK) -- many-to-many
```
### Таблицы
- *teams* - команды разработки

- *users* - участники команд с флагом активности

- *pull_requests* - PR с статусами OPEN/MERGED

- *pull_request_reviewers* - связь PR и ревьюверов

## API Endpoints

| Method | Endpoint | Назначение |
|--------|----------|------------|
| **GET** | `/health` | Проверка работоспособности сервиса |
| **POST** | `/team/add` | Создание команды с участниками |
| **GET** | `/team/get` | Получение информации о команде |
| **POST** | `/users/setIsActive` | Изменение активности пользователя |
| **GET** | `/users/getReview` | Получение PR пользователя как ревьювера |
| **POST** | `/pullRequest/create` | Создание PR с автоназначением ревьюверов |
| **POST** | `/pullRequest/merge` | Мерж PR (идемпотентная операция) |
| **POST** | `/pullRequest/reassign` | Переназначение ревьювера в PR |
| **GET** | `/stats` | Возвращает статистику по сервису для мониторинга и анализа |

## Описание статистики 

Статистика включает в себя общее количество пользователей, активных пользователей, команд, pull request'ов, pull request'ов со статусом "OPEN", pull request'ов со статусом "MERGED" и топ-5 ревьюеров(позволяет выявить кто перегружен).

Получить статистику:
```bash
curl http://localhost:8080/stats
```

Ответ:
```json
{
    "total_users": 9,
    "active_users": 8,
    "total_teams": 3,
    "total_prs": 2,
    "open_prs": 2,
    "merged_prs": 0,
    "top_reviewers": [
        {
            "user_id": "z2",
            "username": "Bob",
            "review_count": 1
        },
        {
            "user_id": "u3",
            "username": "Charlie",
            "review_count": 1
        },
        {
            "user_id": "z3",
            "username": "Charlie",
            "review_count": 1
        },
        {
            "user_id": "u4",
            "username": "David",
            "review_count": 1
        },
        {
            "user_id": "u6",
            "username": "Frank",
            "review_count": 0
        }
    ]
}
```

## Примеры запросов
### Создать команду
```bash 
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }'
```
### Получить команду
```bash
curl "http://localhost:8080/team/get?team_name=backend"
```

### Изменить активность пользователя
```bash 
curl -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{"user_id": "u1", "is_active": false}'
  ```

### Получить PR пользователя как ревьювера  
```bash 
curl "http://localhost:8080/users/getReview?user_id=u2"
```

### Создать PR (автоназначение 2 активных ревьюверов)
```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1",
    "pull_request_name": "Add feature", 
    "author_id": "u1"
  }'
  ```

### Переназначить ревьювера
```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1",
    "old_user_id": "u2"
  }'
```
### Merge PR
```bash 
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id": "pr-1"}'
  ```

## Бизнес-логика
### Автоназначение ревьюверов
- Назначаются до 2 активных пользователей из команды автора

- Автор исключается из списка кандидатов

- Если кандидатов меньше 2 - назначается доступное количество

### Переназначение
- Новый ревьювер выбирается из той же команды что и старый

- Учитывается активность пользователей

- Нельзя переназначать в мержнутых PR

### Ограничения
- Пользователи с is_active=false не назначаются

- После мержа PR изменения запрещены

- Все операции идемпотентны

## Примечания
Все ошибки возвращаются в формате OpenAPI спецификации

Операция мержа PR идемпотентна

Добавлен эндпоинт статистики
