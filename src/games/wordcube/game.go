// package game provides a wordcube game implementation.
package game

import (
	"fmt"
	"strings"
)

type Game struct {
	Grid       [5][5]string
	ValidWords map[string]bool
}

func NewGame(validWords []string) *Game {
	game := &Game{
		ValidWords: make(map[string]bool),
	}
	for _, word := range validWords {
		game.ValidWords[word] = true
	}
	return game
}

func (g *Game) AddWord(word string, row int, col int, dir string) error {
	if !g.IsValidWord(word) {
		return fmt.Errorf("invalid word: %s", word)
	}

	if dir == "horizontal" {
		if col+len(word) > 5 {
			return fmt.Errorf("word does not fit horizontally")
		}
		for i, char := range word {
			g.Grid[row][col+i] = string(char)
		}
	} else if dir == "vertical" {
		if row+len(word) > 5 {
			return fmt.Errorf("word does not fit vertically")
		}
		for i, char := range word {
			g.Grid[row+i][col] = string(char)
		}
	} else {
		return fmt.Errorf("invalid direction: %s", dir)
	}
	return nil
}

func (g *Game) IsValidWord(word string) bool {
	_, exists := g.ValidWords[strings.ToLower(word)]
	return exists
}

func (g *Game) PrintGrid() {
	for _, row := range g.Grid {
		for _, cell := range row {
			if cell == "" {
				fmt.Print("_ ")
			} else {
				fmt.Print(cell + " ")
			}
		}
		fmt.Println()
	}
}

func main() {
	validWords := []string{"apple", "berry", "cherry", "dates", "elder"}
	game := NewGame(validWords)

	err := game.AddWord("apple", 0, 0, "horizontal")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Added word 'apple' horizontally at (0,0)")
	}

	err = game.AddWord("berry", 0, 0, "vertical")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Added word 'berry' vertically at (0,0)")
	}

	fmt.Println("Current Grid:")
	for _, row := range game.Grid {
		fmt.Println(row)
	}
}
