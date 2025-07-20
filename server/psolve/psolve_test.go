// Package psolve_test... TODO - describe the package
package psolve

import (
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/testing/protocmp"

	cpb "github.com/danielkhasanov/wordcube/gen/v1"
)

func mustReadCheckpoint(t *testing.T, path string) *cpb.Checkpoint {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read checkpoint file: %v", err)
	}
	var checkpoint cpb.Checkpoint
	if err := prototext.Unmarshal(data, &checkpoint); err != nil {
		t.Fatalf("failed to unmarshal checkpoint: %v", err)
	}
	return &checkpoint
}

func TestCheckpoints(t *testing.T) {
	tests := []struct {
		desc string
		c    *cpb.Checkpoint
	}{{
		desc: "Full Square",
		c:    mustReadCheckpoint(t, "testdata/5_letter.textproto"),
	},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := &Options{Checkpoint: test.c}
			s, err := New(opts)
			if err != nil {
				t.Fatalf("error creating game: %v", err)
			}
			gotC := s.ToCheckpoint()
			if diff := cmp.Diff(test.c, gotC, protocmp.Transform()); diff != "" {
				t.Errorf("checkpoint mismatch (-want +got):\n%v", diff)
			}
		})
	}
}

func TestBuildPrefixSkipList(t *testing.T) {
	tests := []struct {
		name       string
		validWords [][]rune
		expected   [][]int
	}{
		{
			name:       "Empty list",
			validWords: [][]rune{},
			expected:   [][]int{},
		},
		{
			name:       "Last Prefixes Point ahead of list",
			validWords: [][]rune{[]rune("aaa")},
			expected:   [][]int{{1, 1, 1}},
		},
		{
			name:       "Cycle Through Prefixes",
			validWords: [][]rune{[]rune("aaa"), []rune("aab"), []rune("abb"), []rune("bbb"), []rune("bbc")},
			expected: [][]int{
				{3, 2, 1},
				{3, 2, 2},
				{3, 3, 3},
				{5, 5, 4},
				{5, 5, 5},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildPrefixSkipList(test.validWords)
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("buildPrefixSkipList() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIndexesIntAndFromInt tests the Int and FromInt methods of the Indexes class.
func TestIndexesIntAndFromInt(t *testing.T) {
	tests := []struct {
		name       string
		values     []int
		base       int
		wantIntErr error
	}{
		{
			name:   "Empty indexes always OK",
			values: []int{},
			base:   10,
		},
		{
			name:   "Base exceeds value OK",
			values: []int{42},
			base:   50,
		},
		{
			name:       "Base less than val errors",
			values:     []int{42},
			base:       25,
			wantIntErr: ErrBaseTooSmall,
		},
		{
			name:   "Multiple values encode OK",
			values: []int{1, 2, 3, 4, 5},
			base:   50,
		},
		{
			name:   "Multiple values giant base encode OK",
			values: []int{3839120, 5628491, 9876, 1, 0},
			base:   1000000000,
		},
		{
			name:   "Many zeros OK",
			values: []int{0, 0, 1, 0, 0},
			base:   10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			indexes := NewIndexes(len(test.values))
			for i, v := range test.values {
				indexes.Emplace(i, v)
			}

			// Convert to int and back to Indexes
			intValue, err := indexes.Int(test.base)
			if !errors.Is(err, test.wantIntErr) {
				t.Errorf("Int() error = %v, wantErr %v", err, test.wantIntErr)
			}
			if err != nil {
				return
			}
			newIndexes, err := FromInt(intValue, test.base, len(test.values))
			if !errors.Is(err, test.wantIntErr) {
				t.Errorf("Int() error = %v, wantErr %v", err, test.wantIntErr)
			}
			if err != nil {
				return
			}
			// Compare the original and new Indexes
			if diff := cmp.Diff(indexes.values, newIndexes.values); diff != "" {
				t.Errorf("Int and FromInt mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
