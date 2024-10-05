package main

import (
	"fmt"

	"github.com/danielkhasanov/wordcube/game"
)

func main() {
	validWords := []string{"apple", "actor", "herry", "dates", "elder"}
	g, err := game.New(validWords)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	err = g.AddWord("apple", 0, 0, game.Horizontal)
	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Println("Added word 'apple' horizontally at (0,0)")
	}

	err = g.AddWord("actor", 0, 0, game.Vertical)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Added word 'actor' vertically at (0,0)")
	}

	fmt.Println("Current Grid:")
	fmt.Println(g.String())
}
