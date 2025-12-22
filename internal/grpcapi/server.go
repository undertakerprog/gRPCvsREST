package grpcapi

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"gRPCvsREST/api/proto/todopb"
	"gRPCvsREST/internal/todo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	todopb.UnimplementedTodoServiceServer
	svc *todo.Service
}

func NewServer(svc *todo.Service) *grpc.Server {
	server := grpc.NewServer(grpc.UnaryInterceptor(loggingInterceptor))
	todopb.RegisterTodoServiceServer(server, &Handler{svc: svc})
	return server
}

func (h *Handler) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.Todo, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if req.PayloadKb < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid payload_kb")
	}

	item, err := h.svc.Create(req.Title, req.Done)
	if err != nil {
		return nil, mapServiceError(err)
	}

	resp := toProto(item)
	if req.PayloadKb > 0 {
		resp.Payload = payloadFromKB(req.PayloadKb)
	}

	return resp, nil
}

func (h *Handler) GetTodo(ctx context.Context, req *todopb.GetTodoRequest) (*todopb.Todo, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	item, err := h.svc.Get(req.Id)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return toProto(item), nil
}

func (h *Handler) ListTodos(ctx context.Context, req *todopb.ListTodosRequest) (*todopb.ListTodosResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if req.Limit < 0 || req.Offset < 0 || req.PayloadKb < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid pagination")
	}

	items, err := h.svc.List(int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, mapServiceError(err)
	}

	resp := &todopb.ListTodosResponse{
		Items: make([]*todopb.Todo, 0, len(items)),
	}
	var payload string
	if req.PayloadKb > 0 {
		payload = payloadFromKB(req.PayloadKb)
	}
	for _, item := range items {
		msg := toProto(item)
		if payload != "" {
			msg.Payload = payload
		}
		resp.Items = append(resp.Items, msg)
	}

	return resp, nil
}

func toProto(item todo.Todo) *todopb.Todo {
	return &todopb.Todo{
		Id:        item.ID,
		Title:     item.Title,
		Done:      item.Done,
		CreatedAt: item.CreatedAt,
		Payload:   item.Payload,
	}
}

func payloadFromKB(kb int32) string {
	if kb <= 0 {
		return ""
	}
	return strings.Repeat("a", int(kb)*1024)
}

func mapServiceError(err error) error {
	if errors.Is(err, todo.ErrInvalidInput) {
		return status.Error(codes.InvalidArgument, "invalid input")
	}
	if errors.Is(err, todo.ErrNotFound) {
		return status.Error(codes.NotFound, "not found")
	}

	log.Printf("internal error: %v", err)
	return status.Error(codes.Internal, "internal error")
}

func loggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	code := status.Code(err)
	log.Printf("%s %s %s", info.FullMethod, code.String(), time.Since(start))
	return resp, err
}
