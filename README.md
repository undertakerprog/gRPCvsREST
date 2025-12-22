# gRPCvsREST

Minimal REST + gRPC example over the same `todo.Service`.

## Run

```
go run ./cmd/server
```

REST listens on `:8080`, gRPC listens on `:9090`.

## REST API

- GET `/health` -> `{"status":"ok"}`
- POST `/todos` -> creates a todo
  - body: `{"title":"test","done":false}`
- GET `/todos?limit=&offset=&payload_kb=` -> list todos
  - `payload_kb` adds a `payload` field of the given KB size in each item
- GET `/todos/{id}` -> single todo

### REST examples

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

## gRPC API

Proto: `api/proto/todo.proto`

### Install protoc plugins

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate stubs

```
protoc --proto_path=api/proto --go_out=. --go_opt=module=gRPCvsREST --go-grpc_out=. --go-grpc_opt=module=gRPCvsREST api/proto/todo.proto
```

### gRPC examples (grpcurl)

```
grpcurl -plaintext -import-path api/proto -proto todo.proto localhost:9090 todo.TodoService/CreateTodo -d "{\"title\":\"test\",\"done\":false,\"payload_kb\":2}"
```

```
grpcurl -plaintext -import-path api/proto -proto todo.proto localhost:9090 todo.TodoService/ListTodos -d "{\"limit\":10,\"offset\":0,\"payload_kb\":1}"
```
