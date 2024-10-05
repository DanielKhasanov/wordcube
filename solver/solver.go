// Package solver provides functionality to solve word puzzles using backtracking search.
package solver

import (
	"math/rand"
)

// Value interface defines the methods that a value should implement.
type Value interface {
	Evaluate() float64
}

// Game interface defines the methods that a game should implement.
type Game interface {
	LegalActions() []Action
	Value() Value
}

// Action interface defines the methods that an action should implement.
type Action interface {
	Apply(g Game) NodeSelector
}

// NodeSelector represents a class that can select a random action according to their probabilities.
type NodeSelector struct {
	probabilityMap map[Game]float64
}

// NewNodeSelector creates a new NodeSelector.
func NewNodeSelector(probabilityMap map[Game]float64) *NodeSelector {
	return &NodeSelector{probabilityMap: probabilityMap}
}

// SelectRandom selects a random game state according to their probabilities.
func (ns *NodeSelector) SelectRandom() Game {
	total := 0.0
	for _, prob := range ns.probabilityMap {
		total += prob
	}
	r := rand.Float64() * total
	for game, prob := range ns.probabilityMap {
		r -= prob
		if r <= 0 {
			return game
		}
	}
	return nil // Should not reach here if probabilities are correctly normalized
}
