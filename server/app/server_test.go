package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// resetSearcherState ensures tests don't leak global searcher state into
// each other; searcher/searcherReady are process-wide globals set by
// initializeSearcher in production.
func resetSearcherState(t *testing.T) {
	t.Helper()
	prevReady := searcherReady
	prevSearcher := searcher
	t.Cleanup(func() {
		searcherReady = prevReady
		searcher = prevSearcher
	})
	searcherReady = false
	searcher = nil
}

func TestStatusHandler_NotReady(t *testing.T) {
	resetSearcherState(t)
	e := newEcho()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if ready, _ := body["ready"].(bool); ready {
		t.Fatalf("ready = %v, want false", body["ready"])
	}
	if _, ok := body["solutions_count"]; ok {
		t.Fatalf("solutions_count should be absent when not ready, got %v", body["solutions_count"])
	}
}

func TestSolutionsHandler_ServiceUnavailableWhenNotReady(t *testing.T) {
	resetSearcherState(t)
	e := newEcho()

	body := strings.NewReader(`{"board":[["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"]]}`)
	req := httptest.NewRequest(http.MethodPost, "/solutions", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestSolutionsHandler_RejectsMalformedBoard(t *testing.T) {
	resetSearcherState(t)
	// The handler checks readiness before validating the board shape, so
	// mark the searcher ready (without a real searcher) to exercise the
	// validation branch specifically.
	searcherReady = true
	e := newEcho()

	tests := []struct {
		desc string
		body string
	}{
		{desc: "not json", body: `not json`},
		{desc: "too few rows", body: `{"board":[["a","b","c","d","e"]]}`},
		{desc: "too few columns", body: `{"board":[["a"],["a"],["a"],["a"],["a"]]}`},
		{desc: "empty body", body: ``},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/solutions", strings.NewReader(test.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status code = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestSolutionsHandler_BodyTooLarge(t *testing.T) {
	resetSearcherState(t)
	searcherReady = true
	e := newEcho()

	// BodyLimit is set to 64K; send a payload well over that.
	oversized := strings.Repeat("a", 100*1024)
	req := httptest.NewRequest(http.MethodPost, "/solutions", strings.NewReader(oversized))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.ContentLength = int64(len(oversized))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestCORSHeaders_Preflight(t *testing.T) {
	resetSearcherState(t)
	e := newEcho()

	req := httptest.NewRequest(http.MethodOptions, "/solutions", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Errorf("Access-Control-Allow-Methods = %q, want it to contain POST", got)
	}
}

func TestSecurityHeaders_Present(t *testing.T) {
	resetSearcherState(t)
	e := newEcho()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	wantHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}
	for header, want := range wantHeaders {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
}

func TestPanicRecovery(t *testing.T) {
	resetSearcherState(t)
	e := newEcho()
	e.GET("/panic-for-test", func(c echo.Context) error {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic-for-test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d (Recover middleware should convert panics to 500s)", rec.Code, http.StatusInternalServerError)
	}
}
