package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	workloadservice "github.com/google/go-tpm-tools/keymanager/workload_service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	socketPath := "/tmp/kmaserver.sock"
	os.Remove(socketPath)

	srv, err := workloadservice.New(ctx, socketPath)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		os.Exit(1)
	}

	go func() {
		if err := srv.Serve(); err != nil {
			fmt.Printf("Serve error: %v\n", err)
		}
	}()

	fmt.Printf("Server listening on %s\n", socketPath)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
	srv.Shutdown(ctx)
}
