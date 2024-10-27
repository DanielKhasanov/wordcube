// package game provides a wordcube game implementation.
package game

import (
	"fmt"
	"strings"
)

type Direction struct {
	rowInc int
	colInc int
}

var (
	Horizontal = Direction{0, 1}
	Vertical   = Direction{1, 0}
)

type Game struct {
	Grid       [][]string
	ValidWords map[string]bool

	size int
}

func New(validWords []string) (*Game, error) {
	if len(validWords) == 0 {
		return nil, fmt.Errorf("no valid words provided")
	}
	size := len(validWords[0])
	if size > 10 {
		return nil, fmt.Errorf("word lengths should be no more than 10")
	}
	game := &Game{
		Grid:       make([][]string, size),
		ValidWords: make(map[string]bool),
		size:       size,
	}
	for i := range size {
		game.Grid[i] = make([]string, size)
	}
	for _, word := range validWords {
		if len(word) != size {
			return nil, fmt.Errorf("word lengths should be equal, size=%d, wordlen=%d word=%s", size, len(word), word)
		}
		game.ValidWords[word] = true
	}
	return game, nil
}

func (g *Game) IsValidAddWord(word string, row int, col int, dir Direction) error {
	if _, exists := g.ValidWords[strings.ToLower(word)]; !exists {
		return fmt.Errorf("invalid word: %s", word)
	}
	if (row < 0 || row >= g.size || col < 0 || col >= g.size) || ((row > 0 && dir == Vertical) || (col > 0 && dir == Horizontal)) {
		return fmt.Errorf("invalid starting position and direction: (%d, %d, %v)", row, col, dir)
	}
	for i, char := range word {
		if g.Grid[row+i*dir.rowInc][col+i*dir.colInc] != "" && g.Grid[row+i*dir.rowInc][col+i*dir.colInc] != string(char) {
			return fmt.Errorf("collision detected at (%d, %d)", row+i*dir.rowInc, col+i*dir.colInc)
		}
	}
	return nil
}

func (g *Game) AddWord(word string, row int, col int, dir Direction) error {
	if err := g.IsValidAddWord(word, row, col, dir); err != nil {
		return fmt.Errorf("invalid AddWord(%s, %d, %d, %v): %s", word, row, col, dir, err)
	}
	for i, char := range word {
		g.Grid[row+i*dir.rowInc][col+i*dir.colInc] = string(char)
	}
	return nil
}

func (g *Game) String() string {
	var sb strings.Builder
	for i, row := range g.Grid {
		for j, cell := range row {
			if cell == "" {
				sb.WriteString("_")
			} else {
				sb.WriteString(cell)
			}
			if j < len(row)-1 {
				sb.WriteString(" ")
			}
		}
		if i < len(g.Grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
