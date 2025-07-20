// Package parallel... TODO - describe the package
package parallel

import (
	"fmt"
	"sort"
	"testing"
)

type instance struct {
	start int
	end   int
}
type work struct {
	index int
}

func (i *instance) DoSomething(c chan *work) {
	for j := i.start; j < i.end; j++ {
		c <- &work{index: j}
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		desc           string
		instances      []*instance
		expectedSorted []int
	}{{
		desc:           "single instance",
		instances:      []*instance{{start: 0, end: 10}},
		expectedSorted: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	},
		{
			desc: "two instances",
			instances: []*instance{{start: 0, end: 5},
				{start: 5, end: 10}},
			expectedSorted: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			g := NewGroup((*instance).DoSomething, test.instances)
			g.Run()
			output := g.Output()
			var actual []int
			for _, w := range output {
				actual = append(actual, w.index)
			}
			sort.Ints(actual)
			if len(actual) != len(test.expectedSorted) {
				t.Errorf("expected %d items, got %d", len(test.expectedSorted), len(actual))
			}
			for i, expected := range test.expectedSorted {
				if i >= len(actual) || actual[i] != expected {
					t.Errorf("at index %d: expected %d, got %d", i, expected, actual[i])
				}
			}
		})
	}
}

func BenchmarkRun(b *testing.B) {
	workload := 1000
	for _, buffer := range []int{10, 100, 500, 1000} {
		for _, instanceCount := range []int{1000, 5000, 10000} {
			name := fmt.Sprintf("buffer=%d,instances=%d", buffer, instanceCount)
			b.Run(name, func(b *testing.B) {
				instances := make([]*instance, instanceCount)
				for i := 0; i < instanceCount; i++ {
					instances[i] = &instance{start: i * workload, end: (i + 1) * workload}
				}
				expectedSorted := make([]int, 0, instanceCount*workload)
				for i := 0; i < instanceCount*workload; i++ {
					expectedSorted = append(expectedSorted, i)
				}
				g := NewGroup((*instance).DoSomething, instances)
				g.Run()
				output := g.Output()
				if len(output) != len(expectedSorted) {
					b.Errorf("expected %d items, got %d", len(expectedSorted), len(output))
				}
				var actual []int
				for _, w := range output {
					actual = append(actual, w.index)
				}
				sort.Ints(actual)
				for i, expected := range expectedSorted {
					if i >= len(actual) || actual[i] != expected {
						b.Errorf("at index %d: expected %d, got %d", i, expected, actual[i])
					}
				}
				fmt.Printf("test %s Duration()=%v\n", name, g.Duration())
			})
		}

	}

}
