package pockettts

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type Tokenizer struct {
	vocabPath    string
	scoresPath   string
	tokenToID    map[string]int64
	idToToken    map[int64]string
	tokenScores  map[string]float32
	padID        int64
	bosID        int64
	eosID        int64
	unkID        int64
	sortedTokens []string
}

func NewTokenizer(vocabPath, scoresPath string) (*Tokenizer, error) {
	t := &Tokenizer{
		vocabPath:   vocabPath,
		scoresPath:  scoresPath,
		tokenToID:   make(map[string]int64),
		idToToken:   make(map[int64]string),
		tokenScores: make(map[string]float32),
		padID:       0,
		bosID:       1,
		eosID:       2,
		unkID:       3,
	}

	vocabData, err := os.ReadFile(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vocab file: %w", err)
	}

	var vocab map[string]int64
	if err := json.Unmarshal(vocabData, &vocab); err != nil {
		return nil, fmt.Errorf("failed to parse vocab JSON: %w", err)
	}

	for token, id := range vocab {
		t.tokenToID[token] = id
		t.idToToken[id] = token
	}

	if scoresPath != "" {
		scoresData, err := os.ReadFile(scoresPath)
		if err == nil {
			var scores map[string]float32
			if err := json.Unmarshal(scoresData, &scores); err == nil {
				t.tokenScores = scores
			}
		}
	}

	t.sortedTokens = make([]string, 0, len(t.tokenToID))
	for token := range t.tokenToID {
		t.sortedTokens = append(t.sortedTokens, token)
	}
	sort.Slice(t.sortedTokens, func(i, j int) bool {
		return len(t.sortedTokens[i]) > len(t.sortedTokens[j])
	})

	return t, nil
}

func (t *Tokenizer) Encode(text string) []int64 {
	tokens := make([]int64, 0, len(text)*2)
	tokens = append(tokens, t.bosID)

	remaining := text
	for len(remaining) > 0 {
		found := false
		for _, token := range t.sortedTokens {
			if len(token) <= len(remaining) && remaining[:len(token)] == token {
				if id, ok := t.tokenToID[token]; ok {
					tokens = append(tokens, id)
					remaining = remaining[len(token):]
					found = true
					break
				}
			}
		}
		if !found {
			char := string([]rune(remaining)[0])
			if id, ok := t.tokenToID[char]; ok {
				tokens = append(tokens, id)
			} else {
				tokens = append(tokens, t.unkID)
			}
			remaining = remaining[len(char):]
		}
	}

	tokens = append(tokens, t.eosID)
	return tokens
}

func (t *Tokenizer) VocabSize() int {
	return len(t.tokenToID)
}

func (t *Tokenizer) PadID() int64 {
	return t.padID
}

func (t *Tokenizer) BosID() int64 {
	return t.bosID
}

func (t *Tokenizer) EosID() int64 {
	return t.eosID
}
