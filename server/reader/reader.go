// Package reader reads game state from files or other sources.
package reader

import (
	"fmt"
	"os"

	pb "github.com/danielkhasanov/wordcube/gen/proto/v1"
	"google.golang.org/protobuf/encoding/prototext"
)

func ParseSolutionSetFile(f string) (*pb.SolutionSet, error) {
	fmt.Printf("Loading solutions from %s\n", f)
	file, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()
	solutionSet := &pb.SolutionSet{}
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	if err := prototext.Unmarshal(content, solutionSet); err != nil {
		return nil, fmt.Errorf("error parsing SolutionSet: %v", err)
	}
	fmt.Printf("Loaded %d solutions\n", len(solutionSet.GetSolutions()))
	return solutionSet, nil
}
