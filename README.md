# gRPCvsREST

Пример сравнения REST и gRPC поверх одного `todo.Service`.

## Запуск

```
go run ./cmd/server
```

Порты: REST `:8080`, gRPC `:9090`.

## Тестирование REST (curl + Postman)

### curl

```
curl http://localhost:8080/health
```

```
curl -X POST http://localhost:8080/todos ^
  -H "Content-Type: application/json" ^
  -d "{\"title\":\"test\",\"done\":false}"
```

```
curl "http://localhost:8080/todos?limit=10&offset=0&payload_kb=2"
```

```
curl http://localhost:8080/todos/1
```

### Postman

- GET `http://localhost:8080/health`
- POST `http://localhost:8080/todos`
  - Body → raw → JSON:
    ```
    {"title":"test","done":false}
    ```
- GET `http://localhost:8080/todos?limit=10&offset=0&payload_kb=2`
- GET `http://localhost:8080/todos/1`

Опционально: можно завести переменную окружения `base_url` и использовать `{{base_url}}` в URL.

## Тестирование gRPC (Postman + grpcurl)

Proto: `api/proto/todo.proto`

### Postman

1) New → gRPC Request.
2) Import Proto → выбрать `api/proto/todo.proto`.
3) Address: `localhost:9090`.
4) Примеры:
   - CreateTodo: `{"title":"test","done":false,"payload_kb":2}`
   - ListTodos: `{"limit":10,"offset":0,"payload_kb":4}`
   - GetTodo: `{"id":1}`

### grpcurl

Если reflection включен (здесь по умолчанию нет), можно:
```
grpcurl -plaintext localhost:9090 list
```

Без reflection (через proto):
```
grpcurl -plaintext -import-path api/proto -proto todo.proto localhost:9090 todo.TodoService/CreateTodo -d "{\"title\":\"test\",\"done\":false,\"payload_kb\":2}"
```

```
grpcurl -plaintext -import-path api/proto -proto todo.proto localhost:9090 todo.TodoService/ListTodos -d "{\"limit\":10,\"offset\":0,\"payload_kb\":4}"
```

```
grpcurl -plaintext -import-path api/proto -proto todo.proto localhost:9090 todo.TodoService/GetTodo -d "{\"id\":1}"
```

## Бенчмарк

Бенч использует `ListTodos` для REST и gRPC, чтобы сравнивать одинаковые ответы.

REST:
```
go run ./cmd/bench --mode=rest --base=http://localhost:8080 --n=20000 --c=50 --payload_kb=32 --limit=100
```

gRPC:
```
go run ./cmd/bench --mode=grpc --grpc=localhost:9090 --n=20000 --c=50 --payload_kb=32 --limit=100
```

Метрики:
- latency p50/p95 (ms)
- RPS
- bytes: REST = `len(body)`, gRPC = `len(proto.Marshal(response))`

Честное сравнение: прогрев + несколько прогонов, одинаковые параметры (`n`, `c`, `payload_kb`, `limit`).

## Рекомендуемые прогоны и что обычно наблюдается

Рекомендуемые `payload_kb`: `0` / `4` / `32` / `128`.

Обычно:
- bytes растут быстрее в REST (JSON), gRPC обычно компактнее (protobuf)
- gRPC чаще стабильнее по latency при росте payload

Важно: результаты зависят от машины, GC и фоновой нагрузки, но общий тренд обычно виден.

## Troubleshooting

- `protoc` не найден: установите Protocol Buffers и добавьте `protoc` в PATH.
- `protoc-gen-go` не найден: проверьте PATH, типичный путь для Go — `%USERPROFILE%\go\bin`.
- порт занят: остановите процесс на `:8080`/`:9090` или измените порты в коде.
