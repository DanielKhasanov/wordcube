package psolve

import (
	"context"
	"testing"

	cpb "github.com/danielkhasanov/wordcube/gen/proto/v1"
)

// newTestSearcher builds a Searcher over two small, independent 2x2 word
// squares so a query can be aimed at a letter that exists in the dictionary
// but never appears at the queried row/column (exercising the "key exists,
// keyVal doesn't" branch of filterSolutionsByKey) as well as a letter that
// is entirely absent from the dictionary (the "key doesn't exist at all"
// branch).
func newTestSearcher() *Searcher {
	ss := &cpb.SolutionSet{
		Dictionary: &cpb.Dictionary{Word: []string{"ab", "ba", "cd", "dc"}},
		Solutions: []*cpb.Square{
			{Word: []int32{0, 1}}, // "ab"/"ba" square
			{Word: []int32{2, 3}}, // "cd"/"dc" square
		},
	}
	return NewSearcher(ss)
}

func TestFindMatchingSolutions_ExactMatch(t *testing.T) {
	s := newTestSearcher()
	grid := [][]rune{{'a', 0}, {0, 0}}

	c, err := s.FindMatchingSolutions(context.Background(), grid)
	if err != nil {
		t.Fatalf("FindMatchingSolutions returned error: %v", err)
	}
	var got []int
	for solution := range c {
		got = append(got, solution)
	}
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("solutions = %v, want [0]", got)
	}
}

// TestFindMatchingSolutions_LetterAbsentFromRowReturnsNoError covers a
// dictionary letter that exists (so its alphabet key type is indexed), but
// never appears at the queried row - "key exists, keyVal doesn't". Before
// the fix, filterSolutionsByKey treated this as a search error instead of a
// legitimate "zero results" outcome.
func TestFindMatchingSolutions_LetterAbsentFromRowReturnsNoError(t *testing.T) {
	s := newTestSearcher()
	// 'b' is a real dictionary letter (row 1 of the "ab"/"ba" square) but
	// never appears in row 0 of any solution (row 0 is always "ab" or "cd"),
	// so the row-0 key type is indexed but this specific letter's value
	// isn't.
	grid := [][]rune{{'b', 0}, {0, 0}}

	c, err := s.FindMatchingSolutions(context.Background(), grid)
	if err != nil {
		t.Fatalf("FindMatchingSolutions returned error: %v, want nil (zero matches is not an error)", err)
	}
	var got []int
	for solution := range c {
		got = append(got, solution)
	}
	if len(got) != 0 {
		t.Fatalf("solutions = %v, want none", got)
	}
}

// TestFindMatchingSolutions_LetterUnknownToDictionaryReturnsNoError covers a
// letter with no entry in the dictionary's alphabet at all - "key doesn't
// exist in the index". Before the fix this was also misclassified as a
// search error rather than zero results.
func TestFindMatchingSolutions_LetterUnknownToDictionaryReturnsNoError(t *testing.T) {
	s := newTestSearcher()
	grid := [][]rune{{'z', 0}, {0, 0}}

	c, err := s.FindMatchingSolutions(context.Background(), grid)
	if err != nil {
		t.Fatalf("FindMatchingSolutions returned error: %v, want nil (zero matches is not an error)", err)
	}
	var got []int
	for solution := range c {
		got = append(got, solution)
	}
	if len(got) != 0 {
		t.Fatalf("solutions = %v, want none", got)
	}
}
