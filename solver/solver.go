// Package solver provides functionality to solve word puzzles using backtracking search.
package solver

type ID int32

func (id ID) Int() int {
	return int(id)
}

type Node[T any] interface {
	// Children returns a list of child nodes of type T.
	Children() []T
	// Id returns the unique identifier of the node.
	Id() ID
	// Terminal returns true if the node is a terminal node.
	Terminal() bool
}

// Solver struct with a cache map and a mutex
type Solver[T Node[T]] struct {
	cache    map[ID]map[ID]T
	visiting map[ID]bool
}

// NewMemoize initializes the cache
func New[T Node[T]]() *Solver[T] {
	return &Solver[T]{cache: make(map[ID]map[ID]T), visiting: make(map[ID]bool)}
}

// CollectTerminals returns a list of terminal nodes in the tree rooted at the given node.
func (s *Solver[T]) CollectTerminals(node T) map[ID]T {
	if result, found := s.cache[node.Id()]; found {
		return result
	}
	var terminals map[ID]T = make(map[ID]T)
	if node.Terminal() {
		terminals[node.Id()] = node
	} else {
		s.visiting[node.Id()] = true
		for _, child := range node.Children() {
			if _, visiting := s.visiting[child.Id()]; visiting {
				continue
			}
			ct := s.CollectTerminals(child)
			for id, terminal := range ct {
				terminals[id] = terminal
			}
		}
	}
	s.cache[node.Id()] = terminals
	return terminals
}
