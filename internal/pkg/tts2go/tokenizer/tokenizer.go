package tokenizer

var symbols = []rune{
	'_', ';', ':', ',', '.', '!', '?', '¡', '¿', '—', '…', '"', '«', '»', '"', '"',
	' ', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
	'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'ɑ', 'ɐ', 'ɒ', 'æ', 'ɓ', 'ʙ', 'β', 'ɔ', 'ɕ', 'ç', 'ɗ', 'ɖ', 'ð', 'ʤ', 'ə',
	'ɘ', 'ɚ', 'ɛ', 'ɜ', 'ɝ', 'ɞ', 'ɟ', 'ʄ', 'ɡ', 'ɠ', 'ɢ', 'ʛ', 'ɦ', 'ɧ', 'ħ',
	'ɥ', 'ʜ', 'ɨ', 'ɪ', 'ʝ', 'ɭ', 'ɬ', 'ɫ', 'ɮ', 'ʟ', 'ɱ', 'ɯ', 'ɰ', 'ŋ', 'ɳ',
	'ɲ', 'ɴ', 'ø', 'ɵ', 'ɸ', 'θ', 'œ', 'ɶ', 'ʘ', 'ɹ', 'ɺ', 'ɾ', 'ɻ', 'ʀ', 'ʁ',
	'ɽ', 'ʂ', 'ʃ', 'ʈ', 'ʧ', 'ʉ', 'ʊ', 'ʋ', 'ⱱ', 'ʌ', 'ɣ', 'ɤ', 'ʍ', 'χ', 'ʎ',
	'ʏ', 'ʑ', 'ʐ', 'ʒ', 'ʔ', 'ʡ', 'ʕ', 'ʢ', 'ǀ', 'ǁ', 'ǂ', 'ǃ', 'ˈ', 'ˌ', 'ː',
	'ˑ', 'ʼ', 'ʴ', 'ʰ', 'ʱ', 'ʲ', 'ʷ', 'ˠ', 'ˤ', '˞', '↓', '↑', '→', '↗', '↘',
	'\'', 'ᵻ',
}

type Tokenizer struct {
	symbolToIndex map[rune]int64
	padIndex      int64
}

func NewTokenizer() *Tokenizer {
	symbolToIndex := make(map[rune]int64)
	for i, s := range symbols {
		symbolToIndex[s] = int64(i)
	}

	return &Tokenizer{
		symbolToIndex: symbolToIndex,
		padIndex:      0,
	}
}

func (t *Tokenizer) Encode(text string) []int64 {
	tokens := make([]int64, 0, len(text)+1)

	tokens = append(tokens, t.padIndex)

	for _, r := range text {
		if idx, ok := t.symbolToIndex[r]; ok {
			tokens = append(tokens, idx)
		}
	}

	return tokens
}

func (t *Tokenizer) VocabSize() int {
	return len(symbols) + 1
}
