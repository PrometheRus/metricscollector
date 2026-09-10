### Простой вариант — одна метрика gauge:
```bash
curl -v -X POST http://localhost:8080/updates/ \
  -H "Content-Type: application/json" \
  -d '[{"id":"testGauge","type":"gauge","value":42.5}]'
```
### Должен вернуть 400, а не 500
```bash
curl -v -X POST http://localhost:8080/updates/ \
  -H "Content-Type: application/json" \
  -d '[{"id":"testGauge","type":"gauge","value": "string"}]'
```

### Батч из разных типов:
```bash
curl -v -X POST http://localhost:8080/updates/ \
  -H "Content-Type: application/json" \
  -d '[
    {"id":"testGauge1","type":"gauge","value":42.5},
    {"id":"testGauge2","type":"gauge","value":-1.0},
    {"id":"PollCount","type":"counter","delta":10}
  ]'
```

Не забудь: у тебя подключен gzip — если декодер в middleware, проверь и так (агент шлёт сжатое):
```bash
curl -v -X POST http://localhost:8080/updates/ \
  -H "Content-Type: application/json" \
  -H "Content-Encoding: gzip" \
  --data-binary @<(echo '[{"id":"g","type":"gauge","value":1}]' | gzip)
```