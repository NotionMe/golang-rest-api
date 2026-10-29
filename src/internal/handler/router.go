package handler

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"notion.me/restApi/src/internal/repository"
)

func NewRouter(repo repository.DeviceRepository) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, cors, recoverer)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Route does not exist")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "Method is not allowed for this route")
	})
	h := NewDeviceHandler(repo)
	r.Route("/api/v1/devices", func(r chi.Router) {
		r.Get("/", h.GetAll)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
	return r
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					panic(err)
				}
				log.Printf("panic: %v\n%s", err, debug.Stack())
				writeError(w, http.StatusInternalServerError, "An unexpected server error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
