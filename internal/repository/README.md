# internal/repository

Этот пакет содержит реализацию работы с базой данных, а также со внешними сервисами.

Важно, чтобы репозиторий не содержал бизнес-логику.

Репозиторий реализует паттерн Repository и служит абстракцией над различными источниками данных, такими как:
- базы данных (PostgreSQL, MySQL и др.)
- внешние API
- файловые системы
- кэши (Redis, Memcached)
- другие источники данных.

---


## Тестирование

Юнит-тесты с детектором гонок:

```
go test ./... -race -count=1
```

Покрытие:

```
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out   # сводка по функциям
go tool cover -html=coverage.out   # HTML-отчёт
```

Интеграционные тесты (`internal/repository`) по умолчанию пропускаются. Для запуска поднимите PostgreSQL и передайте DSN в переменной окружения `TEST_DATABASE_DSN` — миграции из `migrations/` применятся автоматически:

```
docker run --name metrics-test-pg -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:17

TEST_DATABASE_DSN="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
    go test ./internal/repository/ -run Integration -v -count=1
```

- Хендлеры тестируются через `httptest`, хранилище PostgreSQL — через `go-sqlmock` (без реальной БД).
- Используйте отдельную тестовую базу: интеграционные тесты применяют миграции и пишут данные.