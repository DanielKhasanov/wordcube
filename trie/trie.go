package trie

import (
	"errors"
	"fmt"
)

var ErrInvalidChar = errors.New("invalid character")

// TrieNode represents a node in the Trie.
type TrieNode struct {
	children [26]*TrieNode
	isEnd    bool
}

// Trie represents the Trie data structure.
type Trie struct {
	root *TrieNode
}

// NewTrie creates a new Trie.
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{},
	}
}

// Insert inserts a word into the Trie.
func (t *Trie) Insert(word string) error {
	node := t.root
	for _, char := range word {
		index := char - 'a'
		if index < 0 || index >= 26 {
			return fmt.Errorf("%w: %s", ErrInvalidChar, string(char))
		}
		if node.children[index] == nil {
			node.children[index] = &TrieNode{}
		}
		node = node.children[index]
	}
	node.isEnd = true
	return nil
}

// SearchWithWildcard searches for words in the Trie that match the given wildcard pattern.
func (t *Trie) SearchWithWildcard(pattern string) ([]string, error) {
	var results []string
	currentWord := make([]rune, 0, len(pattern))
	return results, searchWithWildcardHelper(t.root, pattern, 0, &currentWord, &results)
}

// searchWithWildcardHelper is a recursive helper function for the SearchWithWildcard method.
func searchWithWildcardHelper(node *TrieNode, pattern string, index int, currentWord *[]rune, results *[]string) error {
	if index == len(pattern) {
		if node.isEnd {
			*results = append(*results, string(*currentWord))
		}
		return nil
	}
	char := rune(pattern[index])
	if char == '.' {
		for i, child := range node.children {
			if child != nil {
				*currentWord = append(*currentWord, 'a'+rune(i))
				if err := searchWithWildcardHelper(child, pattern, index+1, currentWord, results); err != nil {
					return err
				}
				*currentWord = (*currentWord)[:len(*currentWord)-1]
			}
		}
	} else if char < 'a' || char > 'z' {
		return fmt.Errorf("%w: %s", ErrInvalidChar, string(char))
	} else {
		child := node.children[char-'a']
		if child != nil {
			*currentWord = append(*currentWord, char)
			if err := searchWithWildcardHelper(child, pattern, index+1, currentWord, results); err != nil {
				return err
			}
			*currentWord = (*currentWord)[:len(*currentWord)-1]
		}
	}
	return nil
}
