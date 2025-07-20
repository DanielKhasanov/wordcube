// Package parallel provides an interface for for parallel execution of tasks.
package parallel

import (
	"fmt"
	"sync"
	"time"
)

var timeNow = time.Now // For testing purposes, can be mocked

type Group[T any, O any] struct {
	method     func(T, chan *O)
	instances  []T
	inputChans []chan *O
	chanBuffer int
	wg         sync.WaitGroup

	collectionChan chan *O
	output         []*O
	done           chan struct{} // Channel to signal completion
	mu             sync.Mutex    // Protects output slice

	startTime time.Time
	endTime   time.Time
}

func NewGroup[T any, O any](method func(T, chan *O), instances []T) *Group[T, O] {
	return &Group[T, O]{
		method:     method,
		instances:  instances,
		inputChans: make([]chan *O, len(instances)),
		chanBuffer: 1000,
		done:       make(chan struct{}, 1),
	}
}

func (g *Group[T, O]) Run() {
	g.wg.Add(len(g.instances))
	g.collectionChan = make(chan *O, g.chanBuffer)
	for i, instance := range g.instances {
		g.inputChans[i] = make(chan *O, g.chanBuffer)
		go func(i int, instance T) {
			g.method(instance, g.inputChans[i])
			close(g.inputChans[i])
		}(i, instance)
		go func(i int) {
			for output := range g.inputChans[i] {
				g.collectionChan <- output
			}
			g.wg.Done()
		}(i)
	}
	go func() {
		defer close(g.done)
		for output := range g.collectionChan {
			g.output = append(g.output, output)
		}
		g.done <- struct{}{}
	}()
	go func() {
		g.wg.Wait()
		close(g.collectionChan)
		g.endTime = timeNow()
	}()
	g.startTime = timeNow()
}

func (g *Group[T, O]) Output() []*O {
	<-g.done
	return g.output
}

func (g *Group[T, O]) Duration() time.Duration {
	if g.endTime.IsZero() || g.startTime.IsZero() {
		return 0
	}
	return g.endTime.Sub(g.startTime)
}

func main() {
	fmt.Println("Main method does nothing.")
}
