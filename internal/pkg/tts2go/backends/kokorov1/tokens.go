package kokorov1

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Tokenizer struct {
	tokenToID map[string]int64
	idToToken map[int64]string
	padID     int64
}

func NewTokenizer(tokensPath string) (*Tokenizer, error) {
	f, err := os.Open(tokensPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tokens file: %w", err)
	}
	defer f.Close()

	t := &Tokenizer{
		tokenToID: make(map[string]int64),
		idToToken: make(map[int64]string),
		padID:     0,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		token := parts[0]
		var id int64
		if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
			continue
		}
		t.tokenToID[token] = id
		t.idToToken[id] = token
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read tokens file: %w", err)
	}

	return t, nil
}

func (t *Tokenizer) Encode(text string) []int64 {
	tokens := make([]int64, 0, len(text)*2)
	tokens = append(tokens, t.padID)

	for _, r := range text {
		char := string(r)
		if id, ok := t.tokenToID[char]; ok {
			tokens = append(tokens, id)
		}
	}

	return tokens
}

func (t *Tokenizer) EncodeWithLanguage(text string, lang string) []int64 {
	tokens := make([]int64, 0, len(text)*2+2)

	langToken := "[" + strings.ToUpper(lang) + "]"
	if id, ok := t.tokenToID[langToken]; ok {
		tokens = append(tokens, id)
	}

	tokens = append(tokens, t.padID)

	for _, r := range text {
		char := string(r)
		if id, ok := t.tokenToID[char]; ok {
			tokens = append(tokens, id)
		}
	}

	return tokens
}

func (t *Tokenizer) VocabSize() int {
	return len(t.tokenToID)
}
