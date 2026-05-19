package main

import (
	"PoolManagerVM/backend/config"
	ss "PoolManagerVM/backend/grpc"
	"PoolManagerVM/backend/internal"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {

	// loading .env
	config.LoadEnvConfig()
	if err := models.CreateParams(); err != nil {
		log.Fatalf("Failed to initialize OpenStack clients: %v", err)
	}

	// creating context to stop cleanly
	ctx, cancel := context.WithCancel(context.Background())

	//starting database
	config.Start_DB()
	go config.Sync_DB(ctx)

	//preparing workers
	var wg sync.WaitGroup
	worker.LaunchWorkers(5, &wg, ctx)

	// 	//starting goroutines
	go internal.Monitor(ctx)

	go ss.Start_grpc()

	// bloc instruction to shutdown cleanly
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received")
	cancel()
	wg.Wait()

	log.Println("Program exited cleanly")
}
