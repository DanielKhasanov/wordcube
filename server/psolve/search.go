// Package psolve finds all solutions from a partially complete puzzle.
package psolve

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	cpb "github.com/danielkhasanov/wordcube/gen/proto/v1"
)

// timeNow is a variable to allow mocking of time.Now in tests.
var timeNow = time.Now

// SolutionSet is a set of solutions indexed by their key in a Dictionary.
type SolutionSet map[int]bool

// SolutionChan is a channel that can be used to send solutions.
type SolutionChan chan int

type Searcher struct {
	// KeyId : {KeyVal: [solutionSet]}
	keyIndexedSolutions map[int]map[int]SolutionSet

	// Maps letters to their KeyId component. Values are guaranteed to be less than 8 bits wide.
	alphabet        map[rune]int
	inverseAlphabet map[int]rune // Maps KeyId component to letter.

	ss *cpb.SolutionSet // The original solution set.
}

func NewSearcher(ss *cpb.SolutionSet) *Searcher {
	s := &Searcher{
		keyIndexedSolutions: make(map[int]map[int]SolutionSet),
		ss:                  ss,
	}
	s.populateAlphabet(ss)
	// go func() {
	time := timeNow()
	fmt.Printf("Extracting keys from %d solutions...\n", len(ss.GetSolutions()))
	s.keySolutionSet(ss)
	fmt.Printf("Extracted keys in %v\n", timeNow().Sub(time))
	// }()
	return s
}

func (s *Searcher) rowSingleLetterKey(row int, letter rune) int {
	// Game size is assumed to be < 8 bits wide.
	return (1 << 16) | (row << 8) | s.alphabet[letter]
}

func (s *Searcher) colSingleLetterKey(col int, letter rune) int {
	// Game size is assumed to be < 8 bits wide.
	return (2 << 16) | (col << 8) | s.alphabet[letter]
}

func (s *Searcher) keyString(i int) string {
	switch (i >> 16) & 0xFF {
	case 1:
		return fmt.Sprintf("Row %d contains letter %c", (i>>8)&0xFF, s.inverseAlphabet[i&0xFF])
	case 2:
		return fmt.Sprintf("Col %d contains letter %c", (i>>8)&0xFF, s.inverseAlphabet[i&0xFF])
	default:
		return fmt.Sprintf("Unknown key %d", i)
	}
}

func (s *Searcher) extractKeys(grid [][]rune) map[int]int {
	// Row i contains letter l
	// Column j contains letter l
	keys := make(map[int]int)
	columnLetters := make(map[int]map[rune]bool)
	for i, row := range grid {
		for j, letter := range row {
			if letter == 0 {
				continue
			}
			if _, exists := columnLetters[j]; !exists {
				columnLetters[j] = make(map[rune]bool)
			}
			columnLetters[j][letter] = true
			// Value is an indicator.
			keys[s.rowSingleLetterKey(i, letter)] = 1
		}
	}
	for j := range grid[0] {
		for letter := range columnLetters[j] {
			keys[s.colSingleLetterKey(j, letter)] = 1
		}
	}
	return keys
}

func (s *Searcher) populateAlphabet(ss *cpb.SolutionSet) {
	s.alphabet = make(map[rune]int)
	s.inverseAlphabet = make(map[int]rune)
	s.alphabet[0] = 0
	s.inverseAlphabet[0] = 0
	for _, word := range ss.GetDictionary().GetWord() {
		for _, char := range word {
			if _, exists := s.alphabet[char]; !exists {
				s.alphabet[char] = len(s.alphabet)
				s.inverseAlphabet[len(s.inverseAlphabet)] = char
			}
		}
	}
}

// gridFromProto converts a solution proto to a 2D grid of runes.
func gridFromProto(dictionary *cpb.Dictionary, solution *cpb.Square) [][]rune {
	grid := [][]rune{}
	for _, wordId := range solution.GetWord() {
		word := dictionary.GetWord()[wordId]
		row := make([]rune, len(word))
		for i, char := range word {
			row[i] = char
		}
		grid = append(grid, row)
	}
	return grid
}

// keySolutionSet initializes keyIndexedSolutions by extracting keys from each solution in the solutionSet.
func (s *Searcher) keySolutionSet(ss *cpb.SolutionSet) {
	i := 0
	for solutionIndex, solution := range ss.GetSolutions() {
		grid := gridFromProto(ss.GetDictionary(), solution)
		keys := s.extractKeys(grid)
		for key, keyVal := range keys {
			if _, exists := s.keyIndexedSolutions[key]; !exists {
				s.keyIndexedSolutions[key] = make(map[int]SolutionSet)
				s.keyIndexedSolutions[key][keyVal] = make(SolutionSet)
			}
			s.keyIndexedSolutions[key][keyVal][solutionIndex] = true
			i++
			if i%10000000 == 0 {
				fmt.Printf("Processed %d keys\n", i)
			}
		}
	}
}

func (s *Searcher) solutionInSets(solution int, solutionSets []SolutionSet) bool {
	for _, solutionSet := range solutionSets {
		if !solutionSet[solution] {
			return false // Solution not in this set.
		}
	}
	return true // Solution is in all sets.
}

// noCopyIntersection streams the intersection of solutionSets without copying the solutions.
func (s *Searcher) noCopyIntersection(ctx context.Context, solutionSets []SolutionSet, buf int) SolutionChan {
	slices.SortFunc(solutionSets, func(a, b SolutionSet) int {
		return len(a) - len(b)
	})
	// Copy the smallest solution and hash intersect the rest.
	solutionChan := make(SolutionChan, buf)
	go func(ctx context.Context) {
		defer close(solutionChan)
		if len(solutionSets) == 0 {
			log.Println("noCopyIntersection: No solution sets provided.")
			return // No solutions to intersect.
		}
		for solution := range solutionSets[0] {
			select {
			case <-ctx.Done():
				fmt.Printf("noCopyIntersection canceled: %v\n", ctx.Err())
				return
			default:
				// continue processing.
			}
			if s.solutionInSets(solution, solutionSets[1:]) {
				solutionChan <- solution
			}
		}
	}(ctx)
	return solutionChan
}

func (s *Searcher) filterSolutionsByKey(ctx context.Context, keys map[int]int) (SolutionChan, error) {
	solutionSets := []SolutionSet{}
	// TODO, when there are no keys, the grid is empty and we should stream all solutions.
	for key, keyVal := range keys {
		indexedSolutions, exists := s.keyIndexedSolutions[key]
		if !exists {
			return nil, fmt.Errorf("no indexed solutions for key: %s", s.keyString(key))
		}
		matchingSolutions, exists := indexedSolutions[keyVal]
		if !exists {
			return nil, fmt.Errorf("no matching keyVal for key %s", s.keyString(key))
		}
		solutionSets = append(solutionSets, matchingSolutions)
	}
	return s.noCopyIntersection(ctx, solutionSets, 100), nil
}

// TODO move to game package or a utility package.
func printGrid(grid [][]rune) {
	for _, row := range grid {
		for _, char := range row {
			if char == 0 {
				fmt.Print("_")
			} else {
				fmt.Print(string(char))
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

func (s *Searcher) possibleMatch(solution, grid [][]rune) bool {
	for i, row := range grid {
		for j, char := range row {
			if char != 0 && char != solution[i][j] {
				return false // Mismatch found.
			}
		}
	}
	return true
}

// filterSolutionsAbsolutely filters a stream of solutions to ones that.
func (s *Searcher) filterSolutionsAbsolutely(ctx context.Context, solutions SolutionChan, grid [][]rune) SolutionChan {
	c := make(SolutionChan, 100) // Buffered channel for solutions.
	go func() {
		defer close(c)
		for solution := range solutions {
			select {
			case <-ctx.Done():
				fmt.Printf("filterSolutionsAbsolutely canceled: %v\n", ctx.Err())
				return
			default:
				// continue processing.
			}
			solutionSquare := s.ss.GetSolutions()[solution]
			solutionGrid := gridFromProto(s.ss.GetDictionary(), solutionSquare)
			if s.possibleMatch(solutionGrid, grid) {
				c <- solution // Send solution if it matches the grid.
			}
		}
	}()
	return c
}

func (s *Searcher) FindMatchingSolutions(ctx context.Context, grid [][]rune) (SolutionChan, error) {
	fmt.Println("Finding matching solutions for the following grid...")
	printGrid(grid)
	keys := s.extractKeys(grid)
	c, err := s.filterSolutionsByKey(ctx, keys)
	if err != nil {
		return nil, err
	}
	return s.filterSolutionsAbsolutely(ctx, c, grid), nil
}

// GetSolutionSet returns the original solution set used to create this searcher
func (s *Searcher) GetSolutionSet() *cpb.SolutionSet {
	return s.ss
}
