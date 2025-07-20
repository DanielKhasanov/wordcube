package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"

	pb "github.com/danielkhasanov/wordcube/gen/proto/v1"
	"github.com/danielkhasanov/wordcube/parallel"
	"github.com/danielkhasanov/wordcube/psolve"
	"github.com/danielkhasanov/wordcube/reader"
	"google.golang.org/protobuf/encoding/prototext"
)

var mode = flag.String("mode", "find_solutions", "Mode of operation: 'find_solutions' to find all solutions, 'search' to search for a specific solution.")
var outputDir = flag.String("output_dir", "", "Path to the output directory.")

// search mode flags
var gameState = flag.String("game_state", "", "Path to the game state file. This is used only in 'search' mode to find a specific solution.")
var solutionsFile = flag.String("solutions_file", "", "Path to the solutions file. This is used only in 'search' mode to find a specific solution.")

// find_solutions mode flags
var wordList = flag.String("word_list", "", "Path to the word list file. Each word must be on a separate line, be of the same length, and use alphanumeric characters.")
var numPartitions = flag.Int("num_partitions", 2, "Number of partitions to create for the game.")

func readWordList() ([]string, error) {
	fmt.Printf("Loading words from %s\n", *wordList)
	file, err := os.Open(*wordList)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()
	allWords := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allWords = append(allWords, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	fmt.Printf("Loaded %d words\n", len(allWords))
	return allWords, nil
}

func findSolutions() {
	fmt.Println("Hello, let's find all words in the wordcube!")
	numCPU := runtime.NumCPU()
	fmt.Println("Number of CPUs:", numCPU)
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("error getting current directory: %v\n", err)
		return
	}
	fmt.Printf("Current directory: %s\n", currentDir)
	allWords, err := readWordList()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	s, err := psolve.New(&psolve.Options{
		ValidWords: allWords,
	})
	if err != nil {
		fmt.Printf("error creating game: %v\n", err)
		return
	}
	fmt.Printf("Game created:\n%s\n", s.String())
	partitions, err := s.Partition(*numPartitions)
	if err != nil {
		fmt.Printf("error partitioning game: %v\n", err)
		return
	}
	fmt.Printf("Created %d partitions\n", len(partitions))

	group := parallel.NewGroup((*psolve.State).CollectTerminals, partitions)
	group.Run()
	ss := pb.SolutionSet{
		Dictionary: &pb.Dictionary{
			Word: allWords,
		},
		Solutions: group.Output(),
	}
	outputPath := fmt.Sprintf("%s/solutions.textpb", *outputDir)
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("error creating output directory: %v\n", err)
		return
	}
	output, err := prototext.Marshal(&ss)
	if err != nil {
		fmt.Printf("error marshaling solutions: %v\n", err)
		return
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		fmt.Printf("error writing solutions file: %v\n", err)
		return
	}
	fmt.Printf("Solutions written to %s\n", outputPath)
	fmt.Printf("All partitions processed and solutions written in %v\n", group.Duration())
}

// TODO: move to game package.
func printGrid(grid [][]rune) {
	for _, row := range grid {
		for _, char := range row {
			if char == 0 {
				fmt.Print("_")
			} else {
				fmt.Print(string(char))
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

func readGameState() ([][]rune, error) {
	fmt.Printf("Loading game state from %s\n", *gameState)
	file, err := os.Open(*gameState)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()
	gameState := [][]rune{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		row := []rune{}
		for _, char := range line {
			if char == '_' {
				row = append(row, 0)
			} else {
				row = append(row, char)
			}
		}
		gameState = append(gameState, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	fmt.Printf("Loaded game state: \n")
	printGrid(gameState)
	return gameState, nil
}

func search() {
	if *gameState == "" {
		fmt.Println("Please provide a game state file using the --game_state flag.")
		return
	}
	gameState, err := readGameState()
	if err != nil {
		fmt.Printf("error reading game state: %v\n", err)
		return
	}
	ss, err := reader.ParseSolutionSetFile(*solutionsFile)
	if err != nil {
		fmt.Printf("error reading solution set: %v\n", err)
		return
	}
	searcher := psolve.NewSearcher(ss)
	fmt.Printf("Searching for matching solutions in the game state...\n")
	solutions := searcher.FindMatchingSolutions(gameState)
	fmt.Printf("Found %d matching solutions\n", len(solutions))
	outputPath := fmt.Sprintf("%s/matching_solutions.txt", *outputDir)
	fmt.Printf("Writing matching solutions to %s\n", outputPath)
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("error creating output directory: %v\n", err)
		return
	}
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("error opening output file: %v\n", err)
		return
	}
	defer file.Close()
	for solutionIdx := range solutions {
		fmt.Fprintf(file, "Solution %d:\n", solutionIdx+1)
		square := ss.GetSolutions()[solutionIdx]
		for _, row := range square.GetWord() {
			word := ss.GetDictionary().GetWord()[row]
			fmt.Fprintf(file, "%s\n", string(word))
		}
		fmt.Fprintf(file, "\n")
	}
	fmt.Println("")
}

func main() {
	flag.Parse()
	switch *mode {
	case "find_solutions":
		findSolutions()
	case "search":
		search()
	default:
		fmt.Printf("Unknown mode: %s. Please use 'find_solutions' or 'search'.\n", *mode)
	}
}
