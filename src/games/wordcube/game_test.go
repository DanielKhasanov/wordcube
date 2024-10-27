package game_test

import (
	"strings"
	"testing"

	"github.com/dkhasanov/games/wordcube/game"
)

func createGameFromGridString(gridStr string) *game.Game {
	lines := strings.Split(gridStr, "\n")
	g := &game.Game{}
	for i, line := range lines {
		cells := strings.Fields(line)
		for j, cell := range cells {
			if cell != "_" {
				g.Grid[i][j] = cell
			}
		}
	}
	return g
}

func TestCreateGameFromGridString(t *testing.T) {
	gridStr := `a p p l e
				b _ _ _ _
				c _ _ _ _
				d _ _ _ _
				e _ _ _ _`

	g := createGameFromGridString(gridStr)

	expectedGrid := [5][5]string{
		{"a", "p", "p", "l", "e"},
		{"b", "", "", "", ""},
		{"c", "", "", "", ""},
		{"d", "", "", "", ""},
		{"e", "", "", "", ""},
	}

	for i := range expectedGrid {
		for j := range expectedGrid[i] {
			if g.Grid[i][j] != expectedGrid[i][j] {
				t.Errorf("Expected %s at (%d, %d), but got %s", expectedGrid[i][j], i, j, g.Grid[i][j])
			}
		}
	}
}
