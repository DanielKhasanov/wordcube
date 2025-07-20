// Package main provides the API routing for the wordsquare application.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danielkhasanov/wordcube/psolve"
	"github.com/danielkhasanov/wordcube/reader"
	"github.com/labstack/echo/v4"
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

func main() {
	// Initialize the searcher in the background
	go func() {
		if err := initializeSearcher(); err != nil {
			log.Printf("Failed to initialize searcher: %v", err)
		}
	}()

	e := echo.New()

	// Enable CORS for frontend
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Access-Control-Allow-Origin", "*")
			c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if c.Request().Method == "OPTIONS" {
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	})

	// Solutions endpoint for word squares - supports both GET and POST
	e.GET("/solutions", func(c echo.Context) error {
		if !searcherReady {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "Solutions are still being loaded. Please try again in a moment.")
		}
		// Default board state for GET requests
		boardState := [][]rune{
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
		}
		return streamSolutions(c, boardState, 10000)
	})

	e.POST("/solutions", func(c echo.Context) error {
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
		return streamSolutions(c, stringToRuneBoard(req.Board), 10000)
	})

	// Status endpoint to check if searcher is ready
	e.GET("/status", func(c echo.Context) error {
		status := map[string]interface{}{
			"ready": searcherReady,
		}
		if searcherReady {
			status["solutions_count"] = len(searcher.GetSolutionSet().GetSolutions())
		}
		return c.JSON(http.StatusOK, status)
	})

	// Serve static files from the frontend build
	e.Static("/", "static")

	e.Logger.Fatal(e.Start(":1323"))
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
