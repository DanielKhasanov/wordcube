package psolve

import (
	"context"
	"runtime"
	"testing"
	"time"

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

// newManySolutionsSearcher builds a Searcher with n duplicate 2x2 squares
// (word 0 in row 0, word 1 in row 1), all matching the same single-letter
// query key. This lets tests force the intersection/filter pipeline to
// produce more matches than a channel's buffer size, so a consumer that
// stops reading mid-stream can actually observe backpressure.
func newManySolutionsSearcher(n int) *Searcher {
	ss := &cpb.SolutionSet{
		Dictionary: &cpb.Dictionary{Word: []string{"ax", "by"}},
	}
	for i := 0; i < n; i++ {
		ss.Solutions = append(ss.Solutions, &cpb.Square{Word: []int32{0, 1}})
	}
	return NewSearcher(ss)
}

// TestFindMatchingSolutions_AbandonedConsumerDoesNotLeakGoroutines reproduces
// a goroutine leak: when a caller cancels ctx and stops reading from the
// returned SolutionChan (exactly what streamSolutions in server/app does
// when a client disconnects mid-stream, or an encode/write to the response
// fails), the pipeline's internal goroutines (noCopyIntersection and
// filterSolutionsAbsolutely) should exit instead of blocking forever on a
// channel send nobody will ever read.
func TestFindMatchingSolutions_AbandonedConsumerDoesNotLeakGoroutines(t *testing.T) {
	// More than the internal channels' buffer size (100) so the producer
	// goroutines are guaranteed to still have unsent solutions once the
	// buffer fills and the consumer stops reading.
	s := newManySolutionsSearcher(500)
	grid := [][]rune{{'a', 0}, {0, 0}}

	before := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	c, err := s.FindMatchingSolutions(ctx, grid)
	if err != nil {
		t.Fatalf("FindMatchingSolutions returned error: %v", err)
	}
	// Never read from c at all (like a client that disconnects before
	// consuming any streamed result) and give the pipeline goroutines ample
	// time to run all the way to filling both internal channels' buffers and
	// blocking on a send, before abandoning the channel entirely.
	_ = c
	time.Sleep(100 * time.Millisecond)
	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for {
		// Give the runtime a chance to schedule the leaked goroutines so
		// NumGoroutine reflects their exit.
		runtime.Gosched()
		if runtime.NumGoroutine() <= before+1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines leaked: before=%d, after=%d", before, runtime.NumGoroutine())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
