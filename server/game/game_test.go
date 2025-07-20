package game_test

import (
	"testing"

	"github.com/danielkhasanov/wordcube/game"
	"github.com/danielkhasanov/wordcube/trie"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type addWordArgsType = struct {
	word string
	idx  int
	dir  game.Direction
}

func TestAddWord(t *testing.T) {
	tests := []struct {
		desc             string
		addWordArgs      []addWordArgsType
		wantGridStr      string
		wantCrossIndices [][]int
	}{
		{
			desc: "Empty Grid",
			wantGridStr: `  5 5 5 5 5
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _`,
		},
		{
			desc: "Partial Fill",
			addWordArgs: []addWordArgsType{
				{"apple", 2, game.Horizontal},
			},
			wantCrossIndices: [][]int{{0, 1, 2, 3, 4}},
			wantGridStr: `  4 4 4 4 4
5 _ _ _ _ _
5 _ _ _ _ _
0 a p p l e
5 _ _ _ _ _
5 _ _ _ _ _`,
		},
		{
			desc: "Add Order Agnostic 1",
			addWordArgs: []addWordArgsType{
				{"apple", 0, game.Horizontal},
				{"abcde", 0, game.Vertical},
			},
			wantCrossIndices: [][]int{{0, 1, 2, 3, 4}, {1, 2, 3, 4}},
			wantGridStr: `  0 4 4 4 4
0 a p p l e
4 b _ _ _ _
4 c _ _ _ _
4 d _ _ _ _
4 e _ _ _ _`,
		},
		{
			desc: "Add Order Agnostic 2",
			addWordArgs: []addWordArgsType{
				{"abcde", 0, game.Vertical},
				{"apple", 0, game.Horizontal},
			},
			wantCrossIndices: [][]int{{0, 1, 2, 3, 4}, {1, 2, 3, 4}},
			wantGridStr: `  0 4 4 4 4
0 a p p l e
4 b _ _ _ _
4 c _ _ _ _
4 d _ _ _ _
4 e _ _ _ _`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := &game.Options{ValidWords: []string{"apple", "abcde"}}
			g, err := game.New(opts)
			if err != nil {
				t.Fatalf("New() = %v, want nil", err)
			}
			for i, args := range test.addWordArgs {
				crossIndices, err := g.AddWord(trie.NewRuneSlice(args.word), args.idx, args.dir)
				if err != nil {
					t.Errorf("AddWord(%q, %d, %v) = %v, want nil", args.word, args.idx, args.dir, err)
				}
				if diff := cmp.Diff(test.wantCrossIndices[i], crossIndices); diff != "" {
					t.Errorf("AddWord(%q, %d, %v) mismatch (-want +got):\n%s", args.word, args.idx, args.dir, diff)
				}
			}
			output := g.String()
			if output != test.wantGridStr {
				t.Errorf("String() =\n%s\n,want\n%s", output, test.wantGridStr)
			}
		})
	}
}

func TestSubtractIndices(t *testing.T) {
	tests := []struct {
		desc         string
		addWordArgs  []addWordArgsType
		wantGridStrs []string
	}{
		{
			desc: "Order Memory 1",
			addWordArgs: []addWordArgsType{
				{"apple", 0, game.Horizontal},
				{"abcde", 0, game.Vertical},
			},

			wantGridStrs: []string{
				`  5 5 5 5 5
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _`,
				`  4 4 4 4 4
0 a p p l e
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _`,
				`  0 4 4 4 4
0 a p p l e
4 b _ _ _ _
4 c _ _ _ _
4 d _ _ _ _
4 e _ _ _ _`,
			},
		},
		{
			desc: "Order Memory 2",
			addWordArgs: []addWordArgsType{
				{"abcde", 0, game.Vertical},
				{"apple", 0, game.Horizontal},
			},
			wantGridStrs: []string{
				`  5 5 5 5 5
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _
5 _ _ _ _ _`,
				`  0 5 5 5 5
4 a _ _ _ _
4 b _ _ _ _
4 c _ _ _ _
4 d _ _ _ _
4 e _ _ _ _`,
				`  0 4 4 4 4
0 a p p l e
4 b _ _ _ _
4 c _ _ _ _
4 d _ _ _ _
4 e _ _ _ _`,
			},
		}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := &game.Options{ValidWords: []string{"apple", "abcde"}}
			g, err := game.New(opts)
			if err != nil {
				t.Fatalf("New() = %v, want nil", err)
			}
			crossIndices := [][]int{}
			for i, args := range test.addWordArgs {
				cI, err := g.AddWord(args.word, args.idx, args.dir)
				if err != nil {
					t.Errorf("AddWord(%q, %d, %v) = %v, want nil", args.word, args.idx, args.dir, err)
				}
				crossIndices = append([][]int{cI}, crossIndices...)
				output := g.String()
				if output != test.wantGridStrs[i+1] {
					t.Errorf("AddWord String() =\n%s\n,want\n%s", output, test.wantGridStrs[i+1])
				}
			}
			for i, cI := range crossIndices {
				ri := len(test.addWordArgs) - i - 1
				err := g.SubtractIndices(cI, test.addWordArgs[ri].idx, test.addWordArgs[ri].dir)
				if err != nil {
					t.Errorf("SubtractIndices(%v, %d, %v) = %v, want nil", cI, test.addWordArgs[ri].idx, test.addWordArgs[ri].dir, err)
				}
				output := g.String()
				if output != test.wantGridStrs[ri] {
					t.Errorf("SubtractIndices String() =\n%s\n,want\n%s", output, test.wantGridStrs[ri])
				}
			}
		})
	}
}

func TestChildren(t *testing.T) {
	tests := []struct {
		desc         string
		validWords   []string
		addWordArgs  []addWordArgsType
		wantChildren []string
	}{
		{
			desc:       "Do Not Collect Duplicate Children From Different Actions",
			validWords: []string{"a"},
			wantChildren: []string{
				`  0
0 a`,
			},
		},
		{
			desc:       "First word all permutations",
			validWords: []string{"aa"},
			wantChildren: []string{
				`  0 2
1 a _
1 a _`,
				`  2 0
1 _ a
1 _ a`,
				`  1 1
0 a a
2 _ _`,
				`  1 1
2 _ _
0 a a`,
			},
		},
		{
			desc:       "Use All Valid Words",
			validWords: []string{"a", "b"},
			wantChildren: []string{
				`  0
0 a`,
				`  0
0 b`,
			},
		},
		{
			desc:       "TODO - Do Not Collect Skippable Children",
			validWords: []string{"aaa"},
			addWordArgs: []addWordArgsType{
				{"aaa", 1, game.Horizontal},
				{"aaa", 1, game.Vertical},
				{"aaa", 2, game.Vertical},
			},
			wantChildren: []string{
				`  1 0 0
0 a a a
0 a a a
1 _ a a`,
				`  1 0 0
1 _ a a
0 a a a
0 a a a`,
				`  0 0 0
0 a a a
0 a a a
0 a a a`,
			},
		},
		{
			desc:       "Only collect terminals or valid children.",
			validWords: []string{"aa", "bb"},
			addWordArgs: []addWordArgsType{
				{"aa", 0, game.Vertical},
			},
			wantChildren: []string{
				`  0 1
0 a a
1 a _`,
				`  0 1
1 a _
0 a a`,
				`  0 0
0 a a
0 a a`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := &game.Options{ValidWords: test.validWords}
			g, err := game.New(opts)
			if err != nil {
				t.Fatalf("New() = %v, want nil", err)
			}
			for _, args := range test.addWordArgs {
				_, err := g.AddWord(args.word, args.idx, args.dir)
				if err != nil {
					t.Errorf("AddWord(%q, %d, %v) = %v, want nil", args.word, args.idx, args.dir, err)
				}
			}
			children := g.Children()
			got := []string{}
			for child := range children {
				got = append(got, child.String())
			}
			if diff := cmp.Diff(test.wantChildren, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("Children() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
