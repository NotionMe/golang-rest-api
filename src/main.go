package main

import (
	"log"
	"net/http"
	"notion.me/restApi/src/internal/handler"
	"notion.me/restApi/src/internal/repository"
	"time"
)

func main() {
	repo := repository.NewMemoryDeviceRepository()
	server := &http.Server{
		Addr:              ":8090",
		Handler:           handler.NewRouter(repo),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Println("Device Registry API: сховище sync.RWMutex; middleware Logger, Recoverer, CORS")
	log.Println("HTTP-сервер: http://localhost:8080/api/v1/devices")
	log.Fatal(server.ListenAndServe())
}
