package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notion.me/restApi/src/internal/domain"
	"notion.me/restApi/src/internal/repository"
)

type DeviceHandler struct{ repo repository.DeviceRepository }

type APIError struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func NewDeviceHandler(repo repository.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, APIError{http.StatusText(status), message, status, time.Now().UTC()})
}

func readDevice(w http.ResponseWriter, r *http.Request) (domain.Device, bool) {
	var d domain.Device
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "Request body must contain a valid JSON device (maximum 1 MiB)")
		return d, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "Request body must contain exactly one JSON object")
		return d, false
	}
	d.Name = strings.TrimSpace(d.Name)
	d.IpAddress = strings.TrimSpace(d.IpAddress)
	d.NodeType = strings.TrimSpace(d.NodeType)
	d.Status = strings.TrimSpace(d.Status)
	if d.Name == "" || d.IpAddress == "" || d.NodeType == "" {
		writeError(w, http.StatusBadRequest, "Fields name, ip_address and type are required")
		return d, false
	}
	if net.ParseIP(d.IpAddress) == nil {
		writeError(w, http.StatusBadRequest, "Field ip_address must be a valid IPv4 or IPv6 address")
		return d, false
	}
	if d.Status == "" {
		d.Status = "offline"
	}
	if d.Status != "online" && d.Status != "offline" {
		writeError(w, http.StatusBadRequest, "Field status must be online or offline")
		return d, false
	}
	d.UpdatedAt = time.Now().UTC()
	return d, true
}

func (h *DeviceHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.Filter(r.URL.Query().Get("status"), r.URL.Query().Get("type")))
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	d, ok := readDevice(w, r)
	if !ok {
		return
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "Unable to generate device ID")
		return
	}
	d.Id = "dev-" + hex.EncodeToString(id[:])
	if err := h.repo.Create(d); err != nil {
		writeError(w, http.StatusInternalServerError, "Unable to create device")
		return
	}
	w.Header().Set("Location", "/api/v1/devices/"+d.Id)
	writeJSON(w, http.StatusCreated, d)
}

func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, ok := readDevice(w, r)
	if !ok {
		return
	}
	d.Id = id
	if !h.repo.Update(id, d) {
		deviceNotFound(w, id)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.repo.Delete(id) {
		deviceNotFound(w, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DeviceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, ok := h.repo.GetById(id)
	if !ok {
		deviceNotFound(w, id)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func deviceNotFound(w http.ResponseWriter, id string) {
	writeError(w, http.StatusNotFound, fmt.Sprintf("Device with ID %q does not exist in registry", id))
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
