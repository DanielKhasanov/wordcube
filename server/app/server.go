// Package main provides the API routing for the wordsquare application.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/danielkhasanov/wordcube/psolve"
	"github.com/danielkhasanov/wordcube/reader"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

const (
	defaultPort = "1323"

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second

	// shutdownTimeout bounds how long a graceful shutdown waits for
	// in-flight requests (including streaming /solutions responses) to
	// finish before forcibly closing remaining connections.
	shutdownTimeout = 10 * time.Second

	// defaultSolutionsRateLimit and defaultSolutionsRateBurst bound how
	// often a single client IP may hit the expensive /solutions search
	// endpoint (each request walks the inverted index and streams
	// results), guarding against accidental or malicious hammering.
	defaultSolutionsRateLimit = 5
	defaultSolutionsRateBurst = 10
)

type (
	Solution struct {
		ID   int        `json:"id"`
		Grid [][]string `json:"grid"`
	}

	BoardRequest struct {
		Board [][]string `json:"board"`
	}
)

var (
	// Global searcher for finding solutions
	searcher *psolve.Searcher
	// Flag to track if searcher is ready
	searcherReady bool
)

// initializeSearcher loads solutions from textpb file and creates a searcher
func initializeSearcher() error {
	solutionsFile := "app/data/solutions.textpb"

	fmt.Println("Loading solutions from", solutionsFile)
	solutionSet, err := reader.ParseSolutionSetFile(solutionsFile)
	if err != nil {
		return fmt.Errorf("failed to load solutions: %v", err)
	}

	searcher = psolve.NewSearcher(solutionSet)
	searcherReady = true
	fmt.Printf("Searcher initialized with %d solutions\n", len(solutionSet.GetSolutions()))
	return nil
}

// allowedOrigins returns the CORS origin allowlist. It defaults to "*" to
// preserve existing behavior (the frontend may be served from an arbitrary
// dev-environment URL), but operators can lock this down in production via
// the ALLOWED_ORIGINS env var (comma-separated). Entries are trimmed so a
// stray space after a comma (e.g. "https://a.com, https://b.com") doesn't
// silently produce an origin string that will never match a real Origin
// header, which would otherwise leave the second entry permanently unusable.
func allowedOrigins() []string {
	v := os.Getenv("ALLOWED_ORIGINS")
	if v == "" {
		return []string{"*"}
	}
	origins := strings.Split(v, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return origins
}

// solutionsRateLimiterMiddleware rate limits POST /solutions per client IP
// using a token-bucket algorithm (golang.org/x/time/rate, via Echo's
// RateLimiterMemoryStore), since each request runs a real search against
// the in-memory index rather than serving static/cheap content. Limits are
// configurable via RATE_LIMIT_RPS/RATE_LIMIT_BURST for operators who need to
// tune them, defaulting to a conservative per-IP allowance.
func solutionsRateLimiterMiddleware() echo.MiddlewareFunc {
	rps := defaultSolutionsRateLimit
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			rps = parsed
		}
	}
	burst := defaultSolutionsRateBurst
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			burst = parsed
		}
	}
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:  rate.Limit(rps),
		Burst: burst,
	})
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: store,
	})
}

// ipExtractor returns how Echo (and the /solutions rate limiter, which keys
// off c.RealIP()) determines a client's IP. It defaults to the raw TCP
// connection address (echo.ExtractIPDirect): trusting the X-Forwarded-For/
// X-Real-IP headers by default would let any client bypass per-IP rate
// limiting simply by sending a different header value on every request,
// since this app has no reverse proxy in front of it out of the box (see
// DEPLOYMENT.md). Operators who do deploy behind a trusted reverse proxy on
// their private network can opt in via TRUST_PROXY_HEADERS=true.
func ipExtractor() echo.IPExtractor {
	if os.Getenv("TRUST_PROXY_HEADERS") == "true" {
		return echo.ExtractIPFromXFFHeader()
	}
	return echo.ExtractIPDirect()
}

// newTLSConfig returns the tls.Config used for the server's HTTPS listener.
// It is factored out from main() so tests can exercise the exact production
// config (rather than a hand-copied approximation) against a real TLS
// handshake. MinVersion excludes TLS 1.0/1.1, which are both deprecated
// (RFC 8996) and vulnerable to downgrade/BEAST-style attacks.
func newTLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

// listenAddr returns the ":<port>" address the server binds to. It honors
// the PORT env var (docker-compose.yml has set this since the container was
// first written, but server.go never read it - the listener was always
// hardcoded to :1323 regardless of PORT's value) and falls back to
// defaultPort when PORT is unset or not a valid port number.
func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		return ":" + defaultPort
	}
	if n, err := strconv.Atoi(port); err != nil || n <= 0 || n > 65535 {
		log.Printf("PORT=%q is not a valid port number; falling back to :%s", port, defaultPort)
		return ":" + defaultPort
	}
	return ":" + port
}

// runWithGracefulShutdown runs startFn (an e.Start/e.StartTLS call, which
// blocks until the listener stops or errors) and, when ctx is canceled (a
// SIGINT/SIGTERM), calls e.Shutdown so in-flight requests - including
// streaming /solutions responses - get up to shutdownTimeout to finish
// instead of having their connections killed outright. This matters because
// container orchestrators (Docker, Kubernetes) send SIGTERM on stop/restart,
// and without this the default net/http behavior is to close all
// connections immediately.
func runWithGracefulShutdown(ctx context.Context, e *echo.Echo, startFn func() error) error {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- startFn()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-serveErr
	}
}

// newEcho builds the Echo instance with all middleware and routes
// registered, without starting a listener. This keeps route/middleware
// wiring independently testable via httptest.
func newEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.IPExtractor = ipExtractor()

	// Recover must run first so a panic in any handler returns a 500
	// instead of crashing the whole process.
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
		HSTSMaxAge:         31536000,
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderContentType},
	}))
	// The board payload is a small 5x5 grid of single characters; cap the
	// request body well above that to block oversized-payload abuse.
	e.Use(middleware.BodyLimit("64K"))

	e.POST("/solutions", solutionsHandler, solutionsRateLimiterMiddleware())
	e.GET("/status", statusHandler)

	// Serve static files from the frontend build
	e.Static("/", "static")

	return e
}

func solutionsHandler(c echo.Context) error {
	if !searcherReady {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Solutions are still being loaded. Please try again in a moment.")
	}
	var req BoardRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid board format")
	}
	// Validate board dimensions
	if len(req.Board) != 5 {
		return echo.NewHTTPError(http.StatusBadRequest, "Board must be 5x5")
	}
	for _, row := range req.Board {
		if len(row) != 5 {
			return echo.NewHTTPError(http.StatusBadRequest, "Board must be 5x5")
		}
	}
	return streamSolutions(c, stringToRuneBoard(req.Board), 1000)
}

// Status endpoint to check if searcher is ready
func statusHandler(c echo.Context) error {
	status := map[string]interface{}{
		"ready": searcherReady,
	}
	if searcherReady {
		status["solutions_count"] = len(searcher.GetSolutionSet().GetSolutions())
	}
	return c.JSON(http.StatusOK, status)
}

func main() {
	// Initialize the searcher in the background
	go func() {
		if err := initializeSearcher(); err != nil {
			log.Printf("Failed to initialize searcher: %v", err)
		}
	}()

	e := newEcho()

	// Guard against slow/stalled clients holding connections open
	// (e.g. slowloris-style attacks); net/http's zero-value timeouts are
	// unbounded.
	e.Server.ReadHeaderTimeout = readHeaderTimeout
	e.Server.ReadTimeout = readTimeout
	e.Server.WriteTimeout = writeTimeout
	e.Server.IdleTimeout = idleTimeout

	addr := listenAddr()

	var startFn func() error
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	if certFile != "" && keyFile != "" {
		e.TLSServer.ReadHeaderTimeout = readHeaderTimeout
		e.TLSServer.ReadTimeout = readTimeout
		e.TLSServer.WriteTimeout = writeTimeout
		e.TLSServer.IdleTimeout = idleTimeout
		e.TLSServer.TLSConfig = newTLSConfig()
		startFn = func() error { return e.StartTLS(addr, certFile, keyFile) }
	} else {
		log.Println("TLS_CERT_FILE/TLS_KEY_FILE not set; serving plain HTTP. Terminate TLS here or at a reverse proxy in production.")
		startFn = func() error { return e.Start(addr) }
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runWithGracefulShutdown(ctx, e, startFn); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Fatal(err)
	}
}

func gridFromSolutionIndex(idx int) [][]rune {
	solution := searcher.GetSolutionSet().GetSolutions()[idx]
	dictionary := searcher.GetSolutionSet().GetDictionary()
	grid := make([][]rune, len(solution.GetWord()))
	for i, row := range solution.GetWord() {
		word := dictionary.GetWord()[row]
		grid[i] = make([]rune, len(word))
		for j, char := range word {
			grid[i][j] = char
		}
	}
	return grid
}

func runeToStringBoard(board [][]rune) [][]string {
	stringBoard := make([][]string, len(board))
	for i, row := range board {
		stringRow := make([]string, len(row))
		for j, char := range row {
			if char == 0 {
				stringRow[j] = "_"
			} else {
				stringRow[j] = string(char)
			}
		}
		stringBoard[i] = stringRow
	}
	return stringBoard
}

func stringToRuneBoard(board [][]string) [][]rune {
	runeBoard := make([][]rune, len(board))
	for i, row := range board {
		runeRow := make([]rune, len(row))
		for j, char := range row {
			if char == "_" || char == "" {
				runeRow[j] = 0
			} else {
				runeRow[j] = []rune(strings.ToLower(char))[0]
			}
		}
		runeBoard[i] = runeRow
	}
	return runeBoard
}

func streamSolutions(c echo.Context, boardState [][]rune, limit int) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)
	// Generate solutions based on board state
	ctx, cancel := context.WithTimeout(c.Request().Context(), time.Second*2)
	defer cancel()
	solutionChan, err := searcher.FindMatchingSolutions(ctx, boardState)
	if err != nil {
		fmt.Printf("Failed to find solutions: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find solutions: %v", err))
	}
	enc := json.NewEncoder(c.Response())
	i := 0
	for solutionIdx := range solutionChan {
		grid := gridFromSolutionIndex(solutionIdx)
		solution := Solution{
			ID:   i + 1,
			Grid: runeToStringBoard(grid),
		}
		if err := enc.Encode(solution); err != nil {
			return err
		}
		c.Response().Flush()
		i++
		if i >= limit {
			cancel()
		}
	}
	return nil
}
