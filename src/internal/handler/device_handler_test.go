package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"notion.me/restApi/src/internal/domain"
	"notion.me/restApi/src/internal/repository"
)

func request(t *testing.T, h http.Handler, method, path, body string, status int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != status {
		t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body.String())
	}
	if status != 204 && w.Header().Get("Content-Type") != "application/json" {
		t.Fatal("missing JSON content type")
	}
	return w
}

func TestDeviceLifecycleAndFiltering(t *testing.T) {
	h := NewRouter(repository.NewMemoryDeviceRepository())
	const base = "/api/v1/devices"
	w := request(t, h, "POST", base, `{"id":"ignored","name":"Sensor","ip_address":"192.168.1.50","type":"sensor","status":"online"}`, 201)
	var d domain.Device
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.Id == "" || d.Id == "ignored" || d.UpdatedAt.IsZero() {
		t.Fatalf("invalid generated fields: %+v", d)
	}
	path := base + "/" + d.Id
	if w.Header().Get("Location") != path {
		t.Fatal("invalid Location")
	}
	request(t, h, "GET", path, "", 200)
	request(t, h, "POST", base, `{"name":"Gateway","ip_address":"::1","type":"gateway"}`, 201)
	for query, count := range map[string]int{"": 2, "?status=online": 1, "?type=gateway": 1, "?status=online&type=sensor": 1, "?status=offline&type=sensor": 0} {
		w := request(t, h, "GET", base+query, "", 200)
		var devices []domain.Device
		if err := json.Unmarshal(w.Body.Bytes(), &devices); err != nil {
			t.Fatal(err)
		}
		if devices == nil || len(devices) != count {
			t.Fatalf("filter %q: got %s", query, w.Body.String())
		}
	}
	w = request(t, h, "PUT", path, `{"id":"other","name":"Updated","ip_address":"10.0.0.1","type":"sensor","status":"offline"}`, 200)
	var updated domain.Device
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Id != d.Id || updated.Name != "Updated" || updated.UpdatedAt.Before(d.UpdatedAt) {
		t.Fatalf("bad update: %+v", updated)
	}
	request(t, h, "GET", path, "", 200)
	w = request(t, h, "DELETE", path, "", 204)
	if w.Body.Len() != 0 {
		t.Fatal("204 response has a body")
	}
	request(t, h, "GET", path, "", 404)
	request(t, h, "DELETE", path, "", 404)
	request(t, h, "PUT", path, `{"name":"Missing","ip_address":"10.0.0.1","type":"sensor"}`, 404)
}

func TestValidationAndErrors(t *testing.T) {
	h := NewRouter(repository.NewMemoryDeviceRepository())
	for _, method := range []string{"POST", "PUT"} {
		path := "/api/v1/devices"
		if method == "PUT" {
			path += "/missing"
		}
		for _, body := range []string{`{`, `null`, `{}`, `{"name":" ","ip_address":"::1","type":"sensor"}`, `{"name":"x","ip_address":"bad","type":"sensor"}`, `{"name":"x","ip_address":"::1","type":"sensor","status":"bad"}`, `{"name":"x","ip_address":"::1","type":"sensor"} {}`} {
			w := request(t, h, method, path, body, 400)
			var problem APIError
			if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Status != 400 || problem.Error == "" || problem.Message == "" || problem.Timestamp.IsZero() {
				t.Fatalf("invalid problem: %+v", problem)
			}
		}
	}
	request(t, h, "GET", "/unknown", "", 404)
	request(t, h, "PATCH", "/api/v1/devices", "", 405)
	w := request(t, h, "GET", "/api/v1/devices", "", 200)
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal("invalid requests modified repository")
	}
}

func TestCORS(t *testing.T) {
	h := NewRouter(repository.NewMemoryDeviceRepository())
	r := httptest.NewRequest("OPTIONS", "/api/v1/devices", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "PUT")
	r.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 || w.Body.Len() != 0 || w.Header().Get("Access-Control-Allow-Origin") != "*" || !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "PUT") || !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("invalid preflight: %+v", w)
	}
	w = request(t, h, "GET", "/unknown", "", 404)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS on errors")
	}
}

type panicRepository struct{ repository.DeviceRepository }

func (panicRepository) Filter(string, string) []domain.Device { panic("test panic") }

func TestPanicRecovery(t *testing.T) {
	h := NewRouter(panicRepository{repository.NewMemoryDeviceRepository()})
	w := request(t, h, "GET", "/api/v1/devices", "", 500)
	var problem APIError
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Status != 500 || problem.Timestamp.IsZero() || strings.Contains(w.Body.String(), "test panic") {
		t.Fatal("invalid recovery response")
	}
	request(t, h, "GET", "/api/v1/devices/missing", "", 404)
}
