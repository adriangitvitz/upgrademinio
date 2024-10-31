package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"upgrademinio/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	basePath := "tmp/webhook"
	asbPath, err := filepath.Abs(basePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	contentService := handlers.NewRetrieveService(asbPath)
	defer contentService.Close()

	handler := handlers.NewHandler(contentService)

	router.POST("/create", handler.HandleRetrieveContent)
	router.GET("/:tag/:name", handler.HandleGetBinary)
	router.StaticFS("/update", http.Dir(asbPath))

	server := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
}
