// Package psolve provides a solver specifically optimized for wordcube.
package psolve

import (
	"errors"
	"fmt"
	"iter"
	"math/big"
	"strings"

	"github.com/danielkhasanov/wordcube/trie"

	cpb "github.com/danielkhasanov/wordcube/gen/proto/v1"
)

var ErrBaseTooSmall = errors.New("base must be greater than each value")
var ErrExpectedSizeTooSmall = errors.New("expected size must be greater than extracted values'")

type Options struct {
	// ValidWords is a list of valid words.
	ValidWords []string
	// Checkpoint is a checkpoint to restore the solver from.
	Checkpoint *cpb.Checkpoint
}

type Indexes struct {
	values []int
}

func NewIndexes(size int) Indexes {
	return Indexes{values: make([]int, size)}
}

func (i Indexes) Len() int {
	return len(i.values)
}

func (i *Indexes) Emplace(index int, value int) {
	if index >= 0 && index < len(i.values) {
		i.values[index] = value
	}
}

func (i *Indexes) At(index int) int {
	return i.values[index]
}

func (i Indexes) Less(other Indexes) bool {
	for idx, val := range i.values {
		// No length check, dangerous.
		if val < other.values[idx] {
			return true
		} else if val > other.values[idx] {
			return false
		}
	}
	return false
}

func (i Indexes) Values() iter.Seq[int] {
	return func(yield func(int) bool) {
		for _, v := range i.values {
			if !yield(v) {
				return
			}
		}
	}
}

type State struct {
	size          int
	validWords    [][]rune
	verticalTries []*trie.Trie
	// rowWordIndexes[i] = j -> The word at row i is validWords[j]. i > currentRow -> j = 0.
	rowWordIndexes Indexes
	// index of largest populated row.
	currentRow int
	// The following are used to partition the state.
	// startRowWordIndexes[i] = j -> The first word this search state started at in row i was j.
	startRowWordIndexes Indexes
	// endRowWordIndexes[i] = j -> The last word this search state will examine at row i is j.
	endRowWordIndexes Indexes
	// Optimization for skipping words that share a common prefix when the prefix yields no valid words.
	// prefixSkipList[i][j] = k -> k is the smallest such that validWords[i][:j+1] != validWords[k][:j+1]
	prefixSkipList [][]int
}

func buildPrefixSkipList(words [][]rune) [][]int {
	prefixSkipList := make([][]int, len(words))
	if len(words) == 0 {
		return prefixSkipList
	}
	lastWord := words[0]
	size := len(lastWord)
	prefixIndices := make([]int, size)
	for i := 1; i < len(words)+1; i++ {
		prefixSkipList[i-1] = make([]int, size)
		for j := 0; j < size; j++ {
			word := make([]rune, size)
			if i < len(words) {
				word = words[i]
			}
			if string(word[:j+1]) != string(lastWord[:j+1]) {
				prefixIndex := prefixIndices[j]
				for k := prefixIndex; k < i; k++ {
					prefixSkipList[k][j] = i
				}
				prefixIndices[j] = i
			}
		}
		if i < len(words) {
			lastWord = words[i]
		}
	}
	return prefixSkipList
}

func (s *State) buildFromWords(words []string) error {
	if len(words) == 0 {
		return fmt.Errorf("no valid words provided")
	}
	size := len(words[0])
	if size > 10 {
		return fmt.Errorf("word lengths should be no more than 10")
	}
	s.size = size
	s.currentRow = -1
	s.validWords = make([][]rune, len(words))
	s.rowWordIndexes = NewIndexes(size)
	s.startRowWordIndexes = NewIndexes(size)
	s.endRowWordIndexes = NewIndexes(size)
	for i := range size {
		verticalTrie, err := trie.NewTrie(size)
		if err != nil {
			return fmt.Errorf("error creating trie: %v", err)
		}
		s.verticalTries = append(s.verticalTries, verticalTrie)
		s.rowWordIndexes.Emplace(i, 0)
		s.startRowWordIndexes.Emplace(i, 0)
		s.endRowWordIndexes.Emplace(i, len(words)-1)
	}
	for i, word := range words {
		s.validWords[i] = []rune(word)
		if len(word) != size {
			return fmt.Errorf("word lengths should be equal, size=%d, wordlen=%d word=%s", size, len(word), word)
		}
		for _, t := range s.verticalTries {
			err := t.Insert(word)
			if err != nil {
				return fmt.Errorf("error inserting word %s: %v", word, err)
			}
		}
	}
	s.prefixSkipList = buildPrefixSkipList(s.validWords)
	return nil
}

func New(opts *Options) (*State, error) {
	s := &State{}
	if opts.Checkpoint != nil {
		return s, s.FromCheckpoint(opts.Checkpoint)
	}
	return s, s.buildFromWords(opts.ValidWords)
}

// Int returns the Indexes as a big.Int if interepreted as each val in i is a digit in base.
func (i *Indexes) Int(base int) (*big.Int, error) {
	index := big.NewInt(0)
	for j := 0; j < i.Len(); j++ {
		val := i.At(j)
		if val >= base {
			return nil, fmt.Errorf("value %d at index %d base %d: %w", val, j, base, ErrBaseTooSmall)
		}
		index.Mul(index, big.NewInt(int64(base)))
		index.Add(index, big.NewInt(int64(val)))
	}
	return index, nil
}

// FromInt returns an Indexes from a big.Int interpreted as each digit in base is a val in i.
func FromInt(x *big.Int, base int, expectedSize int) (Indexes, error) {
	baseBig := big.NewInt(int64(base))
	valuesReverse := []int{}
	for x.Cmp(big.NewInt(0)) > 0 {
		_, value := x.DivMod(x, baseBig, new(big.Int))
		valuesReverse = append(valuesReverse, int(value.Int64()))
	}
	indexes := NewIndexes(expectedSize)
	if expectedSize < len(valuesReverse) {
		return indexes, fmt.Errorf("%d < %d: %w", expectedSize, len(valuesReverse), ErrExpectedSizeTooSmall)
	}
	for i, value := range valuesReverse {
		indexes.Emplace(expectedSize-i-1, value)
	}
	return indexes, nil
}

// Partition divides the state into n roughly equally sized States whose search spaces partition s's search space.
// The first partition starts where s starts.
// The last partition ends where s ends.
// Other partitions start at one increment after the end of the previous partition and are size |s|/n.
func (s *State) Partition(n int) ([]*State, error) {
	startInt, err := s.startRowWordIndexes.Int(len(s.validWords))
	if err != nil {
		return nil, fmt.Errorf("error converting start row indexes to int: %w", err)
	}
	endInt, err := s.endRowWordIndexes.Int(len(s.validWords))
	if err != nil {
		return nil, fmt.Errorf("error converting end row indexes to int: %w", err)
	}
	rangeSize := new(big.Int).Sub(endInt, startInt)
	if rangeSize.Cmp(big.NewInt(int64(n))) <= 0 {
		return nil, fmt.Errorf("cannot partition into %d partitions, space size is %v (%v - %v)", n, rangeSize, endInt, startInt)
	}
	rangeSize.Div(rangeSize, big.NewInt(int64(n)))
	partitions := make([]*State, n)
	startInts, endInts := make([]*big.Int, n), make([]*big.Int, n)
	for j := 0; j < n; j++ {
		startInt, err := s.startRowWordIndexes.Int(s.size)
		if err != nil {
			return nil, fmt.Errorf("error converting start row indexes to int: %w", err)
		}
		startInt = startInt.Add(startInt, new(big.Int).Mul(big.NewInt(int64(j)), rangeSize))
		if j != 0 {
			endInts[j-1] = startInt
			startInt = new(big.Int).Add(startInt, big.NewInt(1))
		}
		startInts[j] = startInt
	}
	endInts[n-1], err = s.endRowWordIndexes.Int(len(s.validWords))
	if err != nil {
		return nil, fmt.Errorf("error converting end row indexes to int: %w", err)
	}
	for j := 0; j < n; j++ {
		checkpoint := s.ToCheckpoint()
		start, err := FromInt(startInts[j], len(s.validWords), s.size)
		if err != nil {
			return nil, fmt.Errorf("error converting start int to indexes: %w", err)
		}
		checkpoint.Start = &cpb.Square{}
		checkpoint.Current = &cpb.Square{}
		checkpoint.End = &cpb.Square{}
		for word := range start.Values() {
			checkpoint.Start.Word = append(checkpoint.Start.Word, int32(word))
			checkpoint.Current.Word = append(checkpoint.Current.Word, int32(word))
		}
		checkpoint.CurrentRow = int32(s.size)
		if j == 0 {
			checkpoint.CurrentRow = int32(s.currentRow)
		}
		end, err := FromInt(endInts[j], len(s.validWords), s.size)
		if err != nil {
			return nil, fmt.Errorf("error converting end int to indexes: %w", err)
		}
		for word := range end.Values() {
			checkpoint.End.Word = append(checkpoint.End.Word, int32(word))
		}
		newState := &State{}
		if err := newState.FromCheckpoint(checkpoint); err != nil {
			return nil, fmt.Errorf("error creating partition from checkpoint: %v", err)
		}
		partitions[j] = newState
	}
	return partitions, nil
}

// addRow adds a word to the game following preorder traversal.
func (s *State) addRow(wordIdx int) int {
	if s.currentRow >= s.size-1 {
		return -1
	}
	word := s.validWords[wordIdx]
	for i := 0; i < s.size; i++ {
		if !s.verticalTries[i].WalkIn(word[i]) {
			for j := 0; j < i; j++ {
				s.verticalTries[j].WalkOut()
			}
			return i
		}
	}
	s.currentRow++
	s.rowWordIndexes.Emplace(s.currentRow, wordIdx)
	return -1
}

// removeRow removes the word in the current row and returns true if we removed the first row.
func (s *State) removeRow() bool {
	if s.currentRow < 0 {
		return false
	}
	s.rowWordIndexes.Emplace(s.currentRow, 0)
	for i := 0; i < s.size; i++ {
		s.verticalTries[i].WalkOut()
	}
	s.currentRow--
	return s.currentRow < 0
}

// Next advances the state within the search space. Returns false if the search space is exhausted.
func (s *State) Next() bool {
	// The next word to try to add.
	proposedWordIndex := 0
	// Replacement mode. If true, we're replacing the current row.
	replace := false
	if s.currentRow == s.size-1 {
		// If all rows are currently set, propose the next word in the last row.
		proposedWordIndex = s.rowWordIndexes.At(s.currentRow) + 1
		replace = true
	}
	for {
		// The proposed index is out of bounds, so we must backtrack.
		for proposedWordIndex >= len(s.validWords) {
			// Backtracking in replacement mode means removing the current row.
			if replace && s.removeRow() {
				// Backtracking the first row indicates the total search space has been exhausted.
				return false
			}
			// Always enter replacement mode after backtracking since adding a row failed.
			proposedWordIndex = s.rowWordIndexes.At(s.currentRow) + 1
			replace = true
		}
		if replace {
			s.removeRow()
			replace = false
		}
		noWordsPrefixIndex := s.addRow(proposedWordIndex)
		if !s.rowWordIndexes.Less(s.endRowWordIndexes) {
			// The state search space has been exhausted.
			return false
		}
		if noWordsPrefixIndex == -1 {
			// Pruning optimization #1. Only add the word if all column prefixes have at least one potential valid word.
			return true
		}
		// Pruning optimization #2. Skip to the first word that doesn't share the shortest row prefix that invalidates the column.
		proposedWordIndex = s.prefixSkipList[proposedWordIndex][noWordsPrefixIndex]
	}
}

func (s *State) CollectTerminals(c chan *cpb.Square) {
	for {
		if s.Terminal() {
			c <- s.ToSolution()
		}
		if !s.Next() {
			return
		}
	}
}

// ToSolution returns a square proto message of the current state.
func (s *State) ToSolution() *cpb.Square {
	square := cpb.Square{}
	for index := range s.rowWordIndexes.Values() {
		square.Word = append(square.Word, int32(index))
	}
	return &square
}

// FromCheckpoint restores the state from a checkpoint.
func (s *State) FromCheckpoint(c *cpb.Checkpoint) error {
	if err := s.buildFromWords(c.Dictionary.Word); err != nil {
		return fmt.Errorf("error building from checkpointed words: %v", err)
	}
	for i, wordIndex := range c.Current.Word {
		s.rowWordIndexes.Emplace(i, int(wordIndex))
		if s.currentRow < int(c.CurrentRow) {
			s.addRow(int(wordIndex))
		}
	}
	for i, index := range c.Start.Word {
		s.startRowWordIndexes.Emplace(i, int(index))
	}
	for i, index := range c.End.Word {
		s.endRowWordIndexes.Emplace(i, int(index))
	}
	return nil
}

// ToCheckpoint returns a checkpoint proto message of the current state.
func (s *State) ToCheckpoint() *cpb.Checkpoint {
	c := &cpb.Checkpoint{}
	d := &cpb.Dictionary{}
	for _, word := range s.validWords {
		d.Word = append(d.Word, string(word))
	}
	c.Dictionary = d
	c.CurrentRow = int32(s.currentRow)
	c.Current = &cpb.Square{}
	for index := range s.rowWordIndexes.Values() {
		c.Current.Word = append(c.Current.Word, int32(index))
	}
	c.Start = &cpb.Square{}
	for index := range s.startRowWordIndexes.Values() {
		c.Start.Word = append(c.Start.Word, int32(index))
	}
	c.End = &cpb.Square{}
	for index := range s.endRowWordIndexes.Values() {
		c.End.Word = append(c.End.Word, int32(index))
	}
	return c
}

func (s *State) Terminal() bool {
	return s.currentRow == s.size-1
}

func (s *State) String() string {
	var sb strings.Builder
	var grid [][]rune = make([][]rune, s.size)
	for i := 0; i < s.size; i++ {
		if i <= s.currentRow {
			grid[i] = s.validWords[s.rowWordIndexes.At(i)]
		} else {
			grid[i] = make([]rune, s.size)
		}
	}
	for i, row := range grid {
		for _, cell := range row {
			if cell == 0 {
				sb.WriteString("_")
			} else {
				sb.WriteString(string(cell))
			}
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%d", s.rowWordIndexes.At(i)))
		if i < len(grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
