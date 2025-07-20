// package game provides a wordcube game implementation.
package game

import (
	"bufio"
	"fmt"
	"iter"
	"maps"
	"math"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/danielkhasanov/wordcube/solver"
	"github.com/danielkhasanov/wordcube/trie"
)

type Direction struct {
	rowInc int
	colInc int
}

var (
	Horizontal = Direction{0, 1}
	Vertical   = Direction{1, 0}
)

type Options struct {
	// ValidWords is a list of valid words.
	ValidWords []string
}

// Game represents a wordcube game state.
type Game struct {
	// Grid represents a 2D grid, the first index is the row, the second index is the column.
	Grid           [][]rune
	ValidWordsMap  map[string]bool
	ValidWordsTrie *trie.Trie

	size          int
	idValid       bool
	id            solver.ID
	terminalValid bool
	terminal      bool
	// remainingSlots is a list of remaining slots for each row (first size elemnts), then column (second size elements).
	remainingSlots []int
	alphabet       map[rune]int
}

func New(opts *Options) (*Game, error) {
	if len(opts.ValidWords) == 0 {
		return nil, fmt.Errorf("no valid words provided")
	}
	size := len(opts.ValidWords[0])
	if size > 10 {
		return nil, fmt.Errorf("word lengths should be no more than 10")
	}
	remainingSlots := make([]int, 2*size)
	for i := 0; i < 2*size; i++ {
		remainingSlots[i] = size
	}
	globalTrie, err := trie.NewTrie(size)
	if err != nil {
		return nil, fmt.Errorf("error creating trie: %v", err)
	}
	game := &Game{
		Grid:           make([][]rune, size),
		ValidWordsMap:  make(map[string]bool),
		ValidWordsTrie: globalTrie,
		size:           size,
		idValid:        false,
		remainingSlots: remainingSlots,
		alphabet:       map[rune]int{},
	}
	for i := range size {
		game.Grid[i] = make([]rune, size)
	}
	game.alphabet[0] = 0
	for _, word := range opts.ValidWords {
		if len(word) != size {
			return nil, fmt.Errorf("word lengths should be equal, size=%d, wordlen=%d word=%s", size, len(word), word)
		}
		game.ValidWordsMap[word] = true
		err := game.ValidWordsTrie.Insert(word)
		if err != nil {
			return nil, fmt.Errorf("error inserting word %s: %v", word, err)
		}
		for _, char := range word {
			if _, exists := game.alphabet[char]; !exists {
				game.alphabet[char] = len(game.alphabet)
			}
		}
	}
	return game, nil
}

// AddWord adds a word to the game at the specified index and direction and returns cross-indexes that were modified.
func (g *Game) AddWord(word *trie.RuneSlice, idx int, dir Direction) ([]int, error) {
	if idx < 0 || idx >= g.size {
		return nil, fmt.Errorf("invalid starting position %d, game is size %d", idx, g.size)
	}
	if _, exists := g.ValidWordsMap[strings.ToLower(string(word.Runes))]; !exists {
		return nil, fmt.Errorf("invalid word: %v", word)
	}
	x, y := idx, 0
	if dir == Vertical {
		x, y = 0, idx
	}
	ret := []int{}
	for _, char := range word.Runes {
		if g.Grid[x][y] != 0 && g.Grid[x][y] != char {
			return nil, fmt.Errorf("collision detected at (%d, %d)", x, y)
		}
		if g.Grid[x][y] == 0 {
			g.Grid[x][y] = char
			g.remainingSlots[x]--
			g.remainingSlots[g.size+y]--
			if dir == Horizontal {
				ret = append(ret, y)
			} else {
				ret = append(ret, x)
			}
		}
		x += dir.rowInc
		y += dir.colInc
	}
	g.idValid = false
	return ret, nil
}

// SubtractIndices removes the output of AddWord from the game.
func (g *Game) SubtractIndices(crossIndices []int, idx int, dir Direction) error {
	if idx < 0 || idx >= g.size {
		return fmt.Errorf("invalid starting position %d, game is size %d", idx, g.size)
	}
	for _, i := range crossIndices {
		if dir == Horizontal {
			g.Grid[idx][i] = 0
			g.remainingSlots[idx]++
			g.remainingSlots[g.size+i]++
		} else {
			g.Grid[i][idx] = 0
			g.remainingSlots[i]++
			g.remainingSlots[g.size+idx]++
		}
	}
	g.idValid = false
	return nil
}

func (g *Game) Copy() *Game {
	grid := make([][]rune, g.size)
	for i := range g.size {
		grid[i] = make([]rune, g.size)
		copy(grid[i], g.Grid[i])
	}
	remainingSlots := make([]int, 2*g.size)
	copy(remainingSlots, g.remainingSlots)
	return &Game{
		Grid:           grid,
		ValidWordsMap:  g.ValidWordsMap,
		ValidWordsTrie: g.ValidWordsTrie,
		remainingSlots: remainingSlots,
		size:           g.size,
		id:             g.id,
		alphabet:       g.alphabet,
	}
}

// Id returns a unique identifier for the game state based on the grid and alphabet.
func (g *Game) Id() solver.ID {
	if g.idValid {
		return g.id
	}
	base := len(g.alphabet)
	hash := 0
	for i, row := range g.Grid {
		for j, r := range row {
			positionIndex := i*g.size + j
			runeValue := g.alphabet[r]
			hash += runeValue * int(math.Pow(float64(base), float64(positionIndex)))
		}
	}
	g.id = solver.ID(hash)
	g.idValid = true
	return g.id
}

func (g *Game) Terminal() bool {
	if g.terminalValid {
		return g.terminal
	}
	for _, remaining := range g.remainingSlots {
		if remaining != 0 {
			g.terminal = false
			g.terminalValid = true
			return false
		}
	}
	fmt.Printf("Terminal game:\n%s\n", g.String())
	g.terminal = true
	g.terminalValid = true
	return true
}

func (g *Game) extractPattern(idx int, dir Direction) (*trie.RuneSlice, error) {
	if idx < 0 || idx >= g.size {
		return nil, fmt.Errorf("invalid starting position %d, game is size %d", idx, g.size)
	}
	x, y := idx, 0
	if dir == Vertical {
		x, y = 0, idx
	}
	pattern := make([]rune, g.size)
	for i := 0; i < g.size; i++ {
		if g.Grid[x+i*dir.rowInc][y+i*dir.colInc] == 0 {
			pattern[i] = '.'
		} else {
			pattern[i] = rune(g.Grid[x+i*dir.rowInc][y+i*dir.colInc])
		}
	}
	return trie.NewRuneSlice(pattern), nil
}

func (g *Game) Children() iter.Seq[*Game] {
	var cMap = make(map[solver.ID]*Game)
	for idx, remaining := range g.remainingSlots {
		// Set direction and starting position from remaining slots.
		if remaining == 0 {
			continue
		}
		dir := Horizontal
		if idx >= g.size {
			dir = Vertical
			idx -= g.size
		}
		pattern, err := g.extractPattern(idx, dir)
		if err != nil {
			panic(fmt.Errorf("initial search extractPattern(%d, %v): %v\n%s", idx, dir, err, g.String()))
		}
		// Apply matching words to children.
		words, _ := g.ValidWordsTrie.SearchWithWildcard(pattern)
		for _, word := range words {
			// child := g.copy()
			crossIndices, err := g.AddWord(word, idx, dir)
			if err != nil {
				panic(fmt.Errorf("AddWord(%v, %d, %v): %v\n%s", word, idx, dir, err, g.String()))
			}
			crossDir := Vertical
			if dir == Vertical {
				crossDir = Horizontal
			}
			valid := true
			for _, crossIdx := range crossIndices {
				pattern, err := g.extractPattern(crossIdx, crossDir)
				if err != nil {
					panic(fmt.Errorf("cross search extractPattern(%d, %v): %v\n%s", crossIdx, crossDir, err, g.String()))
				}
				words, err := g.ValidWordsTrie.SearchWithWildcard(pattern)
				if err != nil {
					panic(fmt.Errorf("SearchWithWildcard(%v): %v\n%s", pattern, err, g.String()))
				}
				if len(words) == 0 {
					valid = false
					break
				}
			}
			if valid {
				cMap[g.Id()] = g.Copy()
			}
			err = g.SubtractIndices(crossIndices, idx, dir)
			if err != nil {
				panic(fmt.Errorf("SubtractIndices(%v, %d, %v): %v\n%s", crossIndices, idx, dir, err, g.String()))
			}
		}
	}
	return maps.Values(cMap)
}

func (g *Game) String() string {
	var sb strings.Builder
	sb.WriteString("  ")
	for i := 0; i < g.size; i++ {
		if i < g.size-1 {
			sb.WriteString(fmt.Sprintf("%d ", g.remainingSlots[i+g.size]))
		} else {
			sb.WriteString(fmt.Sprintf("%d\n", g.remainingSlots[i+g.size]))
		}
	}
	for i, row := range g.Grid {
		sb.WriteString(fmt.Sprintf("%d ", g.remainingSlots[i]))
		for _, cell := range row {
			if cell == 0 {
				sb.WriteString("_")
			} else {
				sb.WriteString(string(cell))
			}
			sb.WriteString(" ")
		}
		if i < len(g.Grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func main() {
	fmt.Println("Hello, find all words in the wordcube!")
	allWords := []string{}

	// Print the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}
	fmt.Printf("Current directory: %s\n", currentDir)

	filename := "game/data/all_words_5.txt"
	fmt.Printf("Loading words from %s\n", filename)

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allWords = append(allWords, scanner.Text())
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	fmt.Printf("Loaded %d words\n", len(allWords))

	g, err := New(&Options{ValidWords: allWords})
	if err != nil {
		fmt.Printf("Error creating game: %v\n", err)
		return
	}
	fmt.Printf("Game created:\n%s\n", g.String())

	// Start CPU profiling
	cpuProfile, err := os.Create("cpu_profile.prof")
	if err != nil {
		fmt.Printf("Error creating CPU profile: %v\n", err)
		return
	}
	defer cpuProfile.Close()
	if err := pprof.StartCPUProfile(cpuProfile); err != nil {
		fmt.Printf("Error starting CPU profile: %v\n", err)
		return
	}
	defer pprof.StopCPUProfile()

	// Start memory profiling
	memProfile, err := os.Create("mem_profile.prof")
	if err != nil {
		fmt.Printf("Error creating memory profile: %v\n", err)
		return
	}
	defer memProfile.Close()

	s := solver.New[*Game]()
	go s.CollectTerminals(g)

	// Collect memory profile after some time
	time.Sleep(5 * time.Second)
	if err := pprof.WriteHeapProfile(memProfile); err != nil {
		fmt.Printf("Error writing memory profile: %v\n", err)
		return
	}

	for i := 0; i < 100; i++ {
		hits := g.ValidWordsTrie.CacheHits
		misses := len(g.ValidWordsTrie.Cache)
		rate := float64(hits) / float64(hits+misses)
		visited := s.VisitedCount
		fmt.Printf("VisitedCount: %d, Cache Hits: %d, Misses %d, Rate %v\n", visited, hits, misses, rate)
		time.Sleep(10 * time.Second)
	}
}
