package game_test

import (
	"strings"
	"testing"

	"github.com/danielkhasanov/wordcube/game"
)

func createGameFromGridString(t *testing.T, gridStr string) *game.Game {
	lines := strings.Split(gridStr, "\n")
	g, err := game.New([]string{"apple", "abcde"})
	if err != nil {
		t.Fatalf("New() = %v, want nil", err)
	}
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

func TestString(t *testing.T) {
	tests := []struct {
		desc    string
		gridStr string
	}{
		{
			desc:    "Empty Grid",
			gridStr: "_ _ _ _ _\n_ _ _ _ _\n_ _ _ _ _\n_ _ _ _ _\n_ _ _ _ _",
		},
		{
			desc:    "Partially Filled Grid",
			gridStr: "a p p l e\nb _ _ _ _\nc _ _ _ _\nd _ _ _ _\ne _ _ _ _",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			g := createGameFromGridString(t, test.gridStr)
			output := g.String()
			if output != test.gridStr {
				t.Errorf("String() =\n%q\n,want\n%q", output, test.gridStr)
			}
		})
	}
}
