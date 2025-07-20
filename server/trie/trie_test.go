// Package trie_test contains unit tests for the trie package.
package trie_test

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/danielkhasanov/wordcube/trie"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTrie(t *testing.T) {
	tests := []struct {
		desc          string
		wordSet       []string
		wordLength    int
		pattern       string
		expected      []*trie.RuneSlice
		wantInsertErr error
		wantSearchErr error
	}{
		{
			desc:       "Word Missing",
			wordSet:    []string{"berry"},
			wordLength: 5,
			pattern:    "apple",
			expected:   nil,
		},
		{
			desc:       "End Suffix Match",
			wordSet:    []string{"abz", "aba", "acz", "aca", "acb", "acc", "acd"},
			wordLength: 3,
			pattern:    "ab.",
			expected:   []*trie.RuneSlice{trie.NewRuneSlice([]rune("aba")), trie.NewRuneSlice([]rune("abz"))},
		},
		{
			desc:          "Non-Word Characters Insert Error",
			wordSet:       []string{"1"},
			wordLength:    1,
			wantInsertErr: trie.ErrInvalidChar,
		},
		{
			desc:          "Non-Word Characters Search Error",
			wordSet:       []string{"a"},
			wordLength:    1,
			pattern:       "1",
			wantSearchErr: trie.ErrInvalidChar,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tr, err := trie.NewTrie(tc.wordLength)
			if err != nil {
				t.Fatalf("NewTrie(%d) = %v, want nil", tc.wordLength, err)
			}
			for _, word := range tc.wordSet {
				err := tr.Insert(word)
				if !errors.Is(err, tc.wantInsertErr) {
					t.Errorf("Insert(%q) = %v, want %v", word, err, tc.wantInsertErr)
				}
				if err != nil {
					return
				}
			}
			for i := 0; i < 2; i++ {
				pattern := trie.NewRuneSlice([]rune(tc.pattern))
				result, err := tr.SearchWithWildcard(pattern)
				if !errors.Is(err, tc.wantSearchErr) {
					t.Errorf("%dth SearchWithWildcard(%q) = %v, want %v", i, tc.pattern, err, tc.wantSearchErr)
				}
				if err != nil {
					return
				}
				if diff := cmp.Diff(tc.expected, result, cmpopts.SortSlices(func(a, b *trie.RuneSlice) bool {
					return string(a.Runes) < string(b.Runes)
				})); diff != "" {
					t.Errorf("%dth SearchWithWildcard(%q) %v", i, tc.pattern, diff)
				}
			}
		})
	}
}

func generateAllWords(length int, position int, word []rune, seen map[uint64]string, t *testing.T) uint64 {
	if position == length {
		rs := trie.NewRuneSlice(word)
		hash := rs.Hash()
		if _, found := seen[hash]; found {
			t.Errorf("Hash collision: %s and %s have the same hash %d", seen[hash], string(word), hash)
		}
		seen[hash] = string(word)
		return hash
	}
	maxHash := uint64(0)
	for j := 0; j < 27; j++ {
		if j == 0 {
			word[position] = '.'
		} else {
			word[position] = rune('a' + j - 1)
		}
		maxHash = max(maxHash, generateAllWords(length, position+1, word, seen, t))
	}
	return maxHash
}

func TestRuneSliceHash(t *testing.T) {
	for length := 1; length <= 5; length++ {
		t.Run(fmt.Sprintf("length=%d", length), func(t *testing.T) {
			// Generate all possible words of the given length
			word := make([]rune, length)
			seen := make(map[uint64]string)
			maxHash := generateAllWords(length, 0, word, seen, t)
			t.Logf("Generated %d words of length %d, maxHash = %v", len(seen), length, maxHash)
		})
	}
}

func BenchmarkTrieVsSet(b *testing.B) {
	tests := []struct {
		desc            string
		numWords        int
		numDistractions int
		wordLength      int
		wildcardRatio   float32
	}{
		{
			desc:            "Small Words Few Wildcards",
			numWords:        10000,
			numDistractions: 10000,
			wordLength:      5,
			wildcardRatio:   0.1,
		},
		{
			desc:            "Large Words Few Wildcards",
			numWords:        1000000,
			numDistractions: 1000000,
			wordLength:      10,
			wildcardRatio:   0.1,
		},
		{
			desc:            "Small Words Many Wildcards",
			numWords:        10000,
			numDistractions: 10000,
			wordLength:      5,
			wildcardRatio:   0.4,
		},
		{
			desc:            "Large Words Many Wildcards",
			numWords:        1000000,
			numDistractions: 1000000,
			wordLength:      10,
			wildcardRatio:   0.4,
		},
	}
	for _, tc := range tests {
		// Generate random strings
		words := make([]string, tc.numWords)
		for i := 0; i < tc.numWords; i++ {
			words[i] = randomString(tc.wordLength)
		}

		// Generate random distractions
		distractions := make([]string, tc.numDistractions)
		d := 4
		for j := 0; j < tc.numDistractions; j += d {
			for i := 0; i < d; i++ {
				distractions[j+i] = randomString(tc.wordLength - d/2 + i)
			}
		}
		// Benchmark Trie
		b.Run(tc.desc+" Trie", func(b *testing.B) {
			patterns := make([]string, b.N)
			for i := 0; i < b.N; i++ {
				patterns[i] = randomPattern(tc.wordLength, tc.wildcardRatio)
			}
			b.ResetTimer()
			tr, err := trie.NewTrie(tc.wordLength)
			if err != nil {
				b.Fatalf("NewTrie(%d) = %v, want nil", tc.wordLength, err)
			}
			for _, word := range words {
				tr.Insert(word)
			}
			for _, distraction := range distractions {
				tr.Insert(distraction)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tr.SearchWithWildcard(trie.NewRuneSlice([]rune(patterns[i])))
			}
		})
		// Benchmark Set (map)
		b.Run(tc.desc+" Set", func(b *testing.B) {
			patterns := make([]string, b.N)
			for i := 0; i < b.N; i++ {
				patterns[i] = randomPattern(tc.wordLength, tc.wildcardRatio)
			}
			b.ResetTimer()
			set := make(map[string]bool)
			for _, word := range words {
				set[word] = true
			}
			for _, distraction := range distractions {
				set[distraction] = true
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				searchSetWithWildcard(set, patterns[i])
			}
		})
	}
}

// randomString generates a random string of the given length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// randomPattern generates a random pattern with wildcards of the given length.
func randomPattern(length int, wildcard_ratio float32) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		if rand.Float32() < wildcard_ratio {
			b[i] = '.'
			continue
		}
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// searchSetWithWildcard searches for words in the set that match the given wildcard pattern.
func searchSetWithWildcard(set map[string]bool, pattern string) []string {
	var results []string
	for word := range set {
		if matchWildcard(word, pattern) {
			results = append(results, word)
		}
	}
	return results
}

// matchWildcard checks if a word matches the given wildcard pattern.
func matchWildcard(word, pattern string) bool {
	if len(word) != len(pattern) {
		return false
	}
	for i := range word {
		if pattern[i] != '.' && word[i] != pattern[i] {
			return false
		}
	}
	return true
}
