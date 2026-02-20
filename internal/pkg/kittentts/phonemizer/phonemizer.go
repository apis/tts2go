package phonemizer

import (
	"strings"

	"github.com/neurlang/goruut/lib"
	"github.com/neurlang/goruut/models/requests"
)

type Phonemizer struct {
	p *lib.Phonemizer
}

func NewPhonemizer() *Phonemizer {
	return &Phonemizer{
		p: lib.NewPhonemizer(nil),
	}
}

func (ph *Phonemizer) Phonemize(text string) string {
	resp := ph.p.Sentence(requests.PhonemizeSentence{
		Language: "English",
		Sentence: text,
	})

	var result strings.Builder
	for i, word := range resp.Words {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(word.Phonetic)
	}

	return result.String()
}

func (ph *Phonemizer) Close() error {
	return nil
}
