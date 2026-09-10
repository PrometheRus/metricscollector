### 1. запустить сервер с миграциями и DSN
```bash
go run ./cmd/server -d "<твой DSN>" -m ./migrations
```

### 2. записать метрики
```bash
curl -X POST localhost:8080/update/gauge/temp/23.5
curl -X POST localhost:8080/update/counter/polls/5
curl -X POST localhost:8080/update/counter/polls/3
```

### 3. проверить чтение
```bash
curl localhost:8080/value/gauge/temp
curl localhost:8080/value/counter/polls             # должно быть 8
curl localhost:8080/
curl -X POST localhost:8080/update/gauge/unknown/x  # 404
```

### 4. перезапустить сервер

### 5. повторить чтение — данные должны пережить рестарт
```bash
curl localhost:8080/value/gauge/temp
curl localhost:8080/value/counter/polls   # должно быть 8
```