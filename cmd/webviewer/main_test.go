package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("unexpected content type: %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "gotdx 网页查看器") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestHandleMethods(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/methods", nil)
	rec := httptest.NewRecorder()

	handleMethods(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var defs []methodDef
	if err := json.Unmarshal(rec.Body.Bytes(), &defs); err != nil {
		t.Fatalf("unmarshal methods: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("expected methods")
	}
	if defs[0].Key == "" {
		t.Fatal("expected method key")
	}
}

func TestHandleQueryRejectsUnknownMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"method":"missing","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleQuery(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "未知方法") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}
