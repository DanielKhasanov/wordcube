package solver_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/danielkhasanov/wordcube/solver"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type TestNode struct {
	id            solver.ID
	terminal      bool
	children      []*TestNode
	childrenCalls int
}

func (n *TestNode) Id() solver.ID {
	return n.id
}

func (n *TestNode) Terminal() bool {
	return n.terminal
}

func (n *TestNode) Children() []*TestNode {
	n.childrenCalls++
	return n.children
}

func TestCollectTerminals(t *testing.T) {
	tests := []struct {
		desc              string
		nodesNoChildren   map[solver.ID]*TestNode
		childPairs        []struct{ parent, child solver.ID }
		root              solver.ID
		want              []solver.ID
		wantChildrenCalls map[solver.ID]int
	}{
		{
			desc: "OneNode",
			nodesNoChildren: map[solver.ID]*TestNode{
				1: {id: 1, terminal: true},
			},
			root: 1,
			want: []solver.ID{1},
		},
		{
			desc: "MultipleReturns",
			nodesNoChildren: map[solver.ID]*TestNode{
				1: {id: 1, terminal: false},
				2: {id: 2, terminal: true},
				3: {id: 3, terminal: true},
			},
			childPairs: []struct{ parent, child solver.ID }{{1, 2}, {1, 3}},
			root:       1,
			want:       []solver.ID{2, 3},
		},
		{
			desc: "LoopTerminates",
			nodesNoChildren: map[solver.ID]*TestNode{
				1: {id: 1, terminal: false},
				2: {id: 2, terminal: true},
				3: {id: 3, terminal: false},
			},
			childPairs: []struct{ parent, child solver.ID }{{1, 2}, {2, 1}, {1, 3}, {3, 1}},
			root:       1,
			want:       []solver.ID{2},
		},
		{
			desc: "Unreachable",
			nodesNoChildren: map[solver.ID]*TestNode{
				1: {id: 1, terminal: false},
				2: {id: 2, terminal: true},
			},
			childPairs: []struct{ parent, child solver.ID }{{2, 1}},
			root:       1,
			want:       []solver.ID{},
		},
		{
			desc: "Memoization",
			nodesNoChildren: map[solver.ID]*TestNode{
				1: {id: 1, terminal: false},
				2: {id: 2, terminal: false},
				3: {id: 3, terminal: false},
				4: {id: 4, terminal: true},
			},
			childPairs: []struct{ parent, child solver.ID }{
				{1, 2},
				{1, 3},
				{2, 3},
				{3, 4},
			},
			root:              1,
			want:              []solver.ID{4},
			wantChildrenCalls: map[solver.ID]int{3: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fullGraph := graph.New(graph.IntHash, graph.Directed())
			for _, node := range test.nodesNoChildren {
				fillcolor := "1"
				if node.Terminal() {
					fillcolor = "2"
				}
				fullGraph.AddVertex(node.Id().Int(),
					graph.VertexAttribute("colorscheme", "blues3"),
					graph.VertexAttribute("style", "filled"),
					graph.VertexAttribute("fillcolor", fillcolor),
				)
			}
			for _, pair := range test.childPairs {
				parent := test.nodesNoChildren[pair.parent]
				child := test.nodesNoChildren[pair.child]
				parent.children = append(parent.children, child)
				fullGraph.AddEdge(parent.Id().Int(), child.Id().Int())
			}
			s := solver.New[*TestNode]()
			root := test.nodesNoChildren[test.root]
			terminals := s.CollectTerminals(root)
			got := []solver.ID{}
			for _, node := range terminals {
				got = append(got, node.Id())
			}
			if diff := cmp.Diff(test.want, got, cmpopts.SortSlices(func(a, b solver.ID) bool { return a < b })); diff != "" {
				t.Errorf("CollectTerminals() mismatch (-want +got):\n%s", diff)
			}
			for id, wantCalls := range test.wantChildrenCalls {
				if gotCalls := test.nodesNoChildren[id].childrenCalls; gotCalls != wantCalls {
					t.Errorf("Node %d: Children() calls = %d, want %d", id, gotCalls, wantCalls)
				}
			}
			full_file, _ := os.Create(fmt.Sprintf("test_graphs/%s_full.gv", test.desc))
			err := draw.DOT(fullGraph, full_file)
			if err != nil {
				t.Fatalf("Error saving full graph: %v", err)
			}
		})
	}
}
