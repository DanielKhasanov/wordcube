package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	cpb "github.com/danielkhasanov/wordcube/gen/proto/v1"
	"github.com/danielkhasanov/wordcube/psolve"
	"github.com/labstack/echo/v4"
)

// newFixtureSearcher builds a tiny real Searcher over a hand-built,
// symmetric 5x5 word square (grid[i][j] == grid[j][i], so rows and columns
// read as the same words) so /solutions can be exercised end-to-end without
// depending on the real app/data/solutions.textpb corpus.
func newFixtureSearcher() *psolve.Searcher {
	words := []string{"afghi", "fbjkl", "gjcmn", "hkmdo", "ilnoe"}
	ss := &cpb.SolutionSet{
		Dictionary: &cpb.Dictionary{Word: words},
		Solutions:  []*cpb.Square{{Word: []int32{0, 1, 2, 3, 4}}},
	}
	return psolve.NewSearcher(ss)
}

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

// withEnv sets an env var for the duration of the test, restoring whatever
// value (or absence) preceded it.
func withEnv(t *testing.T, key, value string) {
	t.Helper()
	prev, hadPrev := os.LookupEnv(key)
	t.Cleanup(func() {
		if hadPrev {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
	os.Setenv(key, value)
}

func TestAllowedOrigins(t *testing.T) {
	tests := []struct {
		name  string
		value string
		unset bool
		want  []string
	}{
		{name: "unset defaults to wildcard", unset: true, want: []string{"*"}},
		{name: "empty defaults to wildcard", value: "", want: []string{"*"}},
		{name: "single origin", value: "https://example.com", want: []string{"https://example.com"}},
		{
			name:  "comma-separated list",
			value: "https://a.example.com,https://b.example.com",
			want:  []string{"https://a.example.com", "https://b.example.com"},
		},
		{
			name:  "whitespace around entries is trimmed",
			value: "https://a.example.com, https://b.example.com , https://c.example.com",
			want:  []string{"https://a.example.com", "https://b.example.com", "https://c.example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				prev, hadPrev := os.LookupEnv("ALLOWED_ORIGINS")
				t.Cleanup(func() {
					if hadPrev {
						os.Setenv("ALLOWED_ORIGINS", prev)
					} else {
						os.Unsetenv("ALLOWED_ORIGINS")
					}
				})
				os.Unsetenv("ALLOWED_ORIGINS")
			} else {
				withEnv(t, "ALLOWED_ORIGINS", tt.value)
			}
			got := allowedOrigins()
			if len(got) != len(tt.want) {
				t.Fatalf("allowedOrigins() = %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("allowedOrigins() = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

// TestCORSHeaders_RestrictedOriginAllowlist proves that setting
// ALLOWED_ORIGINS actually changes the middleware's decision, not just its
// configured value: a request from an allowed origin gets the CORS header
// echoed back, while a request from an origin absent from the allowlist gets
// no Access-Control-Allow-Origin header at all (which is how browsers learn
// to block the cross-origin response). Every other CORS test in this file
// only exercises the default "*" wildcard, so this was previously untested.
func TestCORSHeaders_RestrictedOriginAllowlist(t *testing.T) {
	resetSearcherState(t)
	withEnv(t, "ALLOWED_ORIGINS", "https://allowed.example.com")
	e := newEcho()

	t.Run("allowed origin is echoed back", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/solutions", nil)
		req.Header.Set("Origin", "https://allowed.example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example.com" {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://allowed.example.com")
		}
	})

	t.Run("disallowed origin gets no CORS header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/solutions", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty (origin not in allowlist)", got)
		}
	})
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

func TestSolutionsHandler_StreamsMatchingSolutionAsNDJSON(t *testing.T) {
	resetSearcherState(t)
	searcher = newFixtureSearcher()
	searcherReady = true
	e := newEcho()

	// Only constrain the top-left cell; the fixture has exactly one
	// solution, and this exercises the real key-index/intersection path
	// (not just the "no keys" / not-ready shortcuts other tests cover).
	body := strings.NewReader(`{"board":[["a","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"]]}`)
	req := httptest.NewRequest(http.MethodPost, "/solutions", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var solutions []Solution
	scanner := bufio.NewScanner(rec.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var sol Solution
		if err := json.Unmarshal([]byte(line), &sol); err != nil {
			t.Fatalf("failed to unmarshal NDJSON line %q: %v", line, err)
		}
		solutions = append(solutions, sol)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error scanning response body: %v", err)
	}

	if len(solutions) != 1 {
		t.Fatalf("got %d streamed solutions, want 1: %+v", len(solutions), solutions)
	}

	wantGrid := [][]string{
		{"a", "f", "g", "h", "i"},
		{"f", "b", "j", "k", "l"},
		{"g", "j", "c", "m", "n"},
		{"h", "k", "m", "d", "o"},
		{"i", "l", "n", "o", "e"},
	}
	got := solutions[0]
	if got.ID != 1 {
		t.Errorf("solution ID = %d, want 1", got.ID)
	}
	if len(got.Grid) != len(wantGrid) {
		t.Fatalf("grid has %d rows, want %d", len(got.Grid), len(wantGrid))
	}
	for i := range wantGrid {
		if strings.Join(got.Grid[i], "") != strings.Join(wantGrid[i], "") {
			t.Errorf("row %d = %v, want %v", i, got.Grid[i], wantGrid[i])
		}
	}
}

func TestSolutionsHandler_RateLimited(t *testing.T) {
	resetSearcherState(t)
	// searcherReady must not be set true without a real searcher: statusHandler
	// dereferences the global searcher once ready, so a nil searcher here would
	// panic (Recover would mask it as an unrelated 500 in the assertions below).
	searcher = newFixtureSearcher()
	searcherReady = true
	// Burst of 1 and a very low refill rate means the second request from
	// the same client IP within the test's lifetime must be denied.
	t.Setenv("RATE_LIMIT_RPS", "1")
	t.Setenv("RATE_LIMIT_BURST", "1")
	e := newEcho()

	newReq := func() *http.Request {
		body := strings.NewReader(`{"board":[["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"]]}`)
		req := httptest.NewRequest(http.MethodPost, "/solutions", body)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		return req
	}

	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, newReq())
	if rec1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request unexpectedly rate limited (status %d)", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, newReq())
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d (burst=1 should reject an immediate second request)", rec2.Code, http.StatusTooManyRequests)
	}

	// A request from a different client IP is a distinct rate-limit bucket
	// and should not be affected by the first client's burst.
	rec3 := httptest.NewRecorder()
	req3 := newReq()
	req3.RemoteAddr = "203.0.113.5:1234"
	e.ServeHTTP(rec3, req3)
	if rec3.Code == http.StatusTooManyRequests {
		t.Fatalf("request from a different IP unexpectedly rate limited (status %d)", rec3.Code)
	}

	// /status is not rate limited by the /solutions-specific middleware.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("/status unexpectedly rate limited on request %d", i)
		}
	}
}

func TestSolutionsHandler_RateLimitIgnoresSpoofedForwardedForHeader(t *testing.T) {
	resetSearcherState(t)
	searcher = newFixtureSearcher()
	searcherReady = true
	t.Setenv("RATE_LIMIT_RPS", "1")
	t.Setenv("RATE_LIMIT_BURST", "1")
	// TRUST_PROXY_HEADERS is intentionally left unset: by default this app
	// has no reverse proxy in front of it, so the rate limiter must key off
	// the actual TCP connection address rather than a client-supplied
	// X-Forwarded-For header, or any client could bypass per-IP limiting by
	// sending a different header value on every request.
	e := newEcho()

	newReq := func(xff string) *http.Request {
		body := strings.NewReader(`{"board":[["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"]]}`)
		req := httptest.NewRequest(http.MethodPost, "/solutions", body)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderXForwardedFor, xff)
		req.RemoteAddr = "198.51.100.1:1234"
		return req
	}

	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, newReq("1.1.1.1"))
	if rec1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request unexpectedly rate limited (status %d)", rec1.Code)
	}

	// Same underlying connection, but a different (attacker-controlled)
	// X-Forwarded-For value on every request. If the extractor trusted this
	// header, each request would land in its own fresh bucket and never be
	// limited.
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, newReq("2.2.2.2"))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request with spoofed X-Forwarded-For status = %d, want %d (rate limit must key off RemoteAddr, not client-supplied headers)", rec2.Code, http.StatusTooManyRequests)
	}
}

func TestSolutionsHandler_TrustProxyHeadersHonorsXFFFromTrustedProxy(t *testing.T) {
	resetSearcherState(t)
	searcher = newFixtureSearcher()
	searcherReady = true
	t.Setenv("RATE_LIMIT_RPS", "1")
	t.Setenv("RATE_LIMIT_BURST", "1")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	e := newEcho()

	newReq := func(xff string) *http.Request {
		body := strings.NewReader(`{"board":[["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"]]}`)
		req := httptest.NewRequest(http.MethodPost, "/solutions", body)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderXForwardedFor, xff)
		// The direct TCP peer is loopback, standing in for a trusted reverse
		// proxy running on the same host/private network - echo.ExtractIPFromXFFHeader
		// only honors X-Forwarded-For when the immediate peer is itself
		// trusted (loopback/link-local/private-net by default), so this is
		// what makes the opt-in actually take the XFF branch below.
		req.RemoteAddr = "127.0.0.1:1234"
		return req
	}

	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, newReq("9.9.9.9"))
	if rec1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request unexpectedly rate limited (status %d)", rec1.Code)
	}

	// Same downstream client IP relayed via X-Forwarded-For, through the same
	// trusted proxy: should share the first request's bucket and be denied.
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, newReq("9.9.9.9"))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from same X-Forwarded-For client status = %d, want %d (TRUST_PROXY_HEADERS=true should key off the XFF client IP)", rec2.Code, http.StatusTooManyRequests)
	}

	// A different downstream client IP via the same trusted proxy is a
	// distinct bucket and must not be limited by the first client's burst.
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, newReq("8.8.8.8"))
	if rec3.Code == http.StatusTooManyRequests {
		t.Fatalf("request from a different X-Forwarded-For client unexpectedly rate limited (status %d)", rec3.Code)
	}
}

func TestSolutionsHandler_TrustProxyHeadersIgnoresXFFFromUntrustedRemoteAddr(t *testing.T) {
	resetSearcherState(t)
	searcher = newFixtureSearcher()
	searcherReady = true
	t.Setenv("RATE_LIMIT_RPS", "1")
	t.Setenv("RATE_LIMIT_BURST", "1")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	e := newEcho()

	newReq := func(xff string) *http.Request {
		body := strings.NewReader(`{"board":[["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"],["_","_","_","_","_"]]}`)
		req := httptest.NewRequest(http.MethodPost, "/solutions", body)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderXForwardedFor, xff)
		// A public, non-private direct peer: even with TRUST_PROXY_HEADERS=true,
		// echo.ExtractIPFromXFFHeader only trusts X-Forwarded-For when the
		// immediate connection is itself loopback/link-local/private-net, so
		// this simulates a client talking directly to the app (or through an
		// untrusted intermediary) rather than a real deployed reverse proxy.
		req.RemoteAddr = "203.0.113.9:1234"
		return req
	}

	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, newReq("1.1.1.1"))
	if rec1.Code == http.StatusTooManyRequests {
		t.Fatalf("first request unexpectedly rate limited (status %d)", rec1.Code)
	}

	// Different spoofed X-Forwarded-For, same untrusted direct peer: must
	// still land in the same bucket (RemoteAddr fallback), proving
	// TRUST_PROXY_HEADERS=true doesn't blindly trust XFF from any client -
	// only from ones connecting through a recognized private/loopback peer.
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, newReq("2.2.2.2"))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request with different X-Forwarded-For from an untrusted peer status = %d, want %d (untrusted direct peer must fall back to RemoteAddr, not the client-supplied header)", rec2.Code, http.StatusTooManyRequests)
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

func TestListenAddr(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{name: "unset falls back to default", port: "", want: ":1323"},
		{name: "valid port is honored", port: "8080", want: ":8080"},
		{name: "non-numeric falls back to default", port: "not-a-port", want: ":1323"},
		{name: "out-of-range falls back to default", port: "70000", want: ":1323"},
		{name: "zero falls back to default", port: "0", want: ":1323"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev, hadPrev := os.LookupEnv("PORT")
			t.Cleanup(func() {
				if hadPrev {
					os.Setenv("PORT", prev)
				} else {
					os.Unsetenv("PORT")
				}
			})
			if tt.port == "" {
				os.Unsetenv("PORT")
			} else {
				os.Setenv("PORT", tt.port)
			}
			if got := listenAddr(); got != tt.want {
				t.Fatalf("listenAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

// tlsTestServer starts a real TLS listener serving newEcho(), using the
// exact tls.Config production returns from newTLSConfig() (httptest fills in
// a generated self-signed cert since TLS.Certificates is left empty). This
// exercises the actual TLS handshake negotiation rather than just asserting
// on the tls.Config struct's fields.
func tlsTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	resetSearcherState(t)
	ts := httptest.NewUnstartedServer(newEcho())
	ts.TLS = newTLSConfig()
	ts.StartTLS()
	t.Cleanup(ts.Close)
	return ts
}

func TestTLSConfig_RejectsPreTLS12Handshakes(t *testing.T) {
	ts := tlsTestServer(t)

	for _, tt := range []struct {
		name       string
		maxVersion uint16
	}{
		{name: "TLS 1.0", maxVersion: tls.VersionTLS10},
		{name: "TLS 1.1", maxVersion: tls.VersionTLS11},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true, //nolint:gosec // test-only, dialing our own self-signed cert
						MaxVersion:         tt.maxVersion,
					},
				},
			}
			_, err := client.Get(ts.URL + "/status")
			if err == nil {
				t.Fatalf("handshake capped at %s unexpectedly succeeded; MinVersion=TLS1.2 should reject it", tt.name)
			}
		})
	}
}

// TestRunWithGracefulShutdown_WaitsForInFlightRequestThenStops proves that
// canceling the context (standing in for a SIGTERM from Docker/Kubernetes)
// makes runWithGracefulShutdown call e.Shutdown rather than dropping
// connections outright: a request already in flight is given time to finish
// and receives its full response, and the listener is closed afterward so
// new connections are refused.
func TestRunWithGracefulShutdown_WaitsForInFlightRequestThenStops(t *testing.T) {
	resetSearcherState(t)
	e := newEcho()

	inHandler := make(chan struct{})
	release := make(chan struct{})
	e.GET("/slow-for-test", func(c echo.Context) error {
		close(inHandler)
		<-release
		return c.String(http.StatusOK, "done")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	e.Listener = ln
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() {
		runErr <- runWithGracefulShutdown(ctx, e, func() error { return e.Start("") })
	}()

	reqDone := make(chan struct{})
	var reqErr error
	var reqStatus int
	go func() {
		defer close(reqDone)
		resp, err := http.Get("http://" + addr + "/slow-for-test")
		if err != nil {
			reqErr = err
			return
		}
		defer resp.Body.Close()
		reqStatus = resp.StatusCode
	}()

	select {
	case <-inHandler:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was never invoked")
	}

	// Cancel while the handler is still blocked mid-request, then let it
	// finish shortly after. A non-graceful shutdown would have severed the
	// connection the moment ctx was canceled, before the handler releases.
	cancel()
	time.Sleep(50 * time.Millisecond)
	close(release)

	select {
	case <-reqDone:
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request never completed")
	}
	if reqErr != nil {
		t.Fatalf("in-flight request failed instead of completing gracefully: %v", reqErr)
	}
	if reqStatus != http.StatusOK {
		t.Fatalf("in-flight request status = %d, want %d", reqStatus, http.StatusOK)
	}

	select {
	case err := <-runErr:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("runWithGracefulShutdown error = %v, want http.ErrServerClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runWithGracefulShutdown never returned")
	}

	if _, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		t.Fatal("listener still accepting connections after shutdown")
	}
}

func TestTLSConfig_AcceptsTLS12AndAbove(t *testing.T) {
	ts := tlsTestServer(t)

	for _, tt := range []struct {
		name       string
		minVersion uint16
	}{
		{name: "TLS 1.2", minVersion: tls.VersionTLS12},
		{name: "TLS 1.3", minVersion: tls.VersionTLS13},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true, //nolint:gosec // test-only, dialing our own self-signed cert
						MinVersion:         tt.minVersion,
					},
				},
			}
			resp, err := client.Get(ts.URL + "/status")
			if err != nil {
				t.Fatalf("handshake with MinVersion=%s failed: %v", tt.name, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
			}
		})
	}
}
