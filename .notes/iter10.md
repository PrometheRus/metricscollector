# Сервер:

- Добавьте функциональность подключения к БД. В качестве СУБД используйте PostgreSQL не ниже 10 версии.
- Добавьте в сервер хендлер GET `/ping`, который при запросе проверяет соединение с БД. 
- При успешной проверке хендлер должен вернуть HTTP-статус `200 OK`, при неуспешной — `500 Internal Server Error`.

Строка с адресом подключения к БД должна получаться из переменной окружения `DATABASE_DSN` или флага командной строки `-d`.

```bash
host=host port=port user=myuser password=xxxx dbname=mydb sslmode=disable`
```

Для работы с БД используйте один из следующих пакетов:
- `database/sql`,
- `github.com/jackc/pgx`,
- `github.com/lib/pq`,
- `github.com/jmoiron/sqlx`.