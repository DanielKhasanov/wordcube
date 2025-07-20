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
	parent   *TrieNode
	depth    int
}

// RuneSlice wraps []rune to make it usable as a map key
type RuneSlice struct {
	Runes      []rune
	HashCached uint64
}

// NewRuneSlice creates a RuneSlice from []rune
func NewRuneSlice(r []rune) *RuneSlice {
	return &RuneSlice{Runes: r}
}

// Hash generates a hash for use as map key
func (r *RuneSlice) Hash() uint64 {
	if r.HashCached != 0 {
		return r.HashCached
	}
	h := uint64(1)     // Start with 1 to ensure we never return 0
	base := uint64(27) // Using 27 as base since we have 26 letters + '.'
	for i, c := range r.Runes {
		val := uint64(0)
		if c == '.' {
			val = 26 // Use 26 for '.'
		} else {
			val = uint64(c - 'a') // 0-25 for 'a'-'z'
		}
		h = h + val
		if i < len(r.Runes)-1 {
			h = h * base
		}
	}
	r.HashCached = h
	return h
}

// Updated Trie struct
type Trie struct {
	root       *TrieNode
	Cache      map[uint64][]*RuneSlice // key is Hash(), value includes full RuneSlice for equality check
	CacheHits  int
	wordlength int
	walkedNode *TrieNode
}

// NewTrie creates a new Trie.
func NewTrie(wordlength int) (*Trie, error) {
	if wordlength < 1 || wordlength > 10 {
		return nil, fmt.Errorf("word length should be between 1 and 10, got %d", wordlength)
	}
	root := &TrieNode{}
	root.parent = root
	return &Trie{
		root:       root,
		Cache:      make(map[uint64][]*RuneSlice),
		wordlength: wordlength,
		walkedNode: root,
	}, nil
}

// Insert inserts a word into the Trie.
func (t *Trie) Insert(word string) error {
	if len(word) != t.wordlength {
		return fmt.Errorf("word length should be %d, got %d", t.wordlength, len(word))
	}
	node := t.root
	for _, char := range word {
		index := char - 'a'
		if index < 0 || index >= 26 {
			return fmt.Errorf("%w: %s", ErrInvalidChar, string(char))
		}
		if node.children[index] == nil {
			node.children[index] = &TrieNode{parent: node, depth: node.depth + 1}
		}
		node = node.children[index]
	}
	node.isEnd = true
	return nil
}

// SearchWithWildcard searches for words in the Trie that match the given wildcard pattern.
func (t *Trie) SearchWithWildcard(pattern *RuneSlice) ([]*RuneSlice, error) {
	hash := pattern.Hash()
	if words, found := t.Cache[hash]; found {
		t.CacheHits++
		return words, nil
	}
	currentWord := make([]rune, 0, len(pattern.Runes))
	var results []*RuneSlice
	err := searchWithWildcardHelper(t.root, pattern, 0, &currentWord, &results)
	if err == nil {
		t.Cache[hash] = results
	}
	return results, err
}

// searchWithWildcardHelper is a recursive helper function for the SearchWithWildcard method.
func searchWithWildcardHelper(node *TrieNode, pattern *RuneSlice, index int, currentWord *[]rune, results *[]*RuneSlice) error {
	if index == len(pattern.Runes) {
		if node.isEnd {
			wordCopy := make([]rune, len(*currentWord))
			copy(wordCopy, *currentWord)
			*results = append(*results, NewRuneSlice(wordCopy))
		}
		return nil
	}
	char := pattern.Runes[index]
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

func (t *Trie) WalkIn(r rune) bool {
	node := t.walkedNode.children[r-'a']
	if node == nil {
		return false
	}
	t.walkedNode = node
	return true
}

func (t *Trie) WalkOut() {
	node := t.walkedNode.parent
	if node == nil {
		return
	}
	t.walkedNode = node
}

func (t *Trie) Depth() int {
	return t.walkedNode.depth
}
