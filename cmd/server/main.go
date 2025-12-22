package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gRPCvsREST/internal/grpcapi"
	"gRPCvsREST/internal/httpapi"
	"gRPCvsREST/internal/todo"
)

func main() {
	store := todo.NewStore()
	svc := todo.NewService(store)

	restServer := &http.Server{
		Addr:    ":8080",
		Handler: httpapi.NewHandler(svc),
	}

	grpcListener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("grpc listen error: %v", err)
	}
	grpcServer := grpcapi.NewServer(svc)

	errCh := make(chan error, 2)

	go func() {
		log.Printf("rest listening on %s", restServer.Addr)
		if err := restServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		log.Printf("grpc listening on %s", grpcListener.Addr().String())
		if err := grpcServer.Serve(grpcListener); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("shutdown signal: %s", sig)
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := restServer.Shutdown(ctx); err != nil {
		log.Printf("rest shutdown error: %v", err)
	}
	grpcServer.GracefulStop()
	log.Printf("shutdown complete")
}
