package preprocess

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	whitespaceRe = regexp.MustCompile(`\s+`)
	urlRe        = regexp.MustCompile(`https?://\S+|www\.\S+`)
	htmlTagRe    = regexp.MustCompile(`<[^>]+>`)
	emailRe      = regexp.MustCompile(`\S+@\S+\.\S+`)
)

type Preprocessor struct{}

func NewPreprocessor() *Preprocessor {
	return &Preprocessor{}
}

func (p *Preprocessor) Process(text string) string {
	text = norm.NFC.String(text)
	text = urlRe.ReplaceAllString(text, "")
	text = htmlTagRe.ReplaceAllString(text, "")
	text = emailRe.ReplaceAllString(text, "")
	text = expandContractions(text)
	text = expandNumbers(text)
	text = expandCurrency(text)
	text = expandTime(text)
	text = expandOrdinals(text)
	text = normalizeQuotes(text)
	text = normalizePunctuation(text)
	text = whitespaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

var contractions = map[string]string{
	"won't":     "will not",
	"can't":     "cannot",
	"n't":       " not",
	"'re":       " are",
	"'s":        " is",
	"'d":        " would",
	"'ll":       " will",
	"'ve":       " have",
	"'m":        " am",
	"let's":     "let us",
	"i'm":       "i am",
	"you're":    "you are",
	"he's":      "he is",
	"she's":     "she is",
	"it's":      "it is",
	"we're":     "we are",
	"they're":   "they are",
	"i've":      "i have",
	"you've":    "you have",
	"we've":     "we have",
	"they've":   "they have",
	"i'd":       "i would",
	"you'd":     "you would",
	"he'd":      "he would",
	"she'd":     "she would",
	"we'd":      "we would",
	"they'd":    "they would",
	"i'll":      "i will",
	"you'll":    "you will",
	"he'll":     "he will",
	"she'll":    "she will",
	"we'll":     "we will",
	"they'll":   "they will",
	"isn't":     "is not",
	"aren't":    "are not",
	"wasn't":    "was not",
	"weren't":   "were not",
	"haven't":   "have not",
	"hasn't":    "has not",
	"hadn't":    "had not",
	"doesn't":   "does not",
	"don't":     "do not",
	"didn't":    "did not",
	"wouldn't":  "would not",
	"shouldn't": "should not",
	"couldn't":  "could not",
	"mustn't":   "must not",
	"shan't":    "shall not",
}

func expandContractions(text string) string {
	lower := strings.ToLower(text)
	for contraction, expansion := range contractions {
		lower = strings.ReplaceAll(lower, contraction, expansion)
	}
	if len(text) > 0 && unicode.IsUpper(rune(text[0])) {
		runes := []rune(lower)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		return string(runes)
	}
	return lower
}

var onesWords = []string{
	"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
	"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen",
	"seventeen", "eighteen", "nineteen",
}

var tensWords = []string{
	"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety",
}

var scaleWords = []string{"", "thousand", "million", "billion", "trillion"}

func numberToWords(n int64) string {
	if n == 0 {
		return "zero"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	var parts []string
	scaleIndex := 0

	for n > 0 {
		chunk := n % 1000
		if chunk > 0 {
			chunkWords := chunkToWords(int(chunk))
			if scaleIndex > 0 && scaleIndex < len(scaleWords) {
				chunkWords += " " + scaleWords[scaleIndex]
			}
			parts = append([]string{chunkWords}, parts...)
		}
		n /= 1000
		scaleIndex++
	}

	result := strings.Join(parts, " ")
	if negative {
		result = "negative " + result
	}
	return result
}

func chunkToWords(n int) string {
	if n == 0 {
		return ""
	}
	if n < 20 {
		return onesWords[n]
	}
	if n < 100 {
		tens := tensWords[n/10]
		ones := n % 10
		if ones == 0 {
			return tens
		}
		return tens + " " + onesWords[ones]
	}
	hundreds := onesWords[n/100] + " hundred"
	remainder := n % 100
	if remainder == 0 {
		return hundreds
	}
	return hundreds + " " + chunkToWords(remainder)
}

var numberRe = regexp.MustCompile(`\b(\d{1,15})\b`)

func expandNumbers(text string) string {
	return numberRe.ReplaceAllStringFunc(text, func(match string) string {
		var n int64
		for _, c := range match {
			n = n*10 + int64(c-'0')
		}
		return numberToWords(n)
	})
}

var currencyRe = regexp.MustCompile(`\$(\d+(?:\.\d{2})?)`)

func expandCurrency(text string) string {
	return currencyRe.ReplaceAllStringFunc(text, func(match string) string {
		match = strings.TrimPrefix(match, "$")
		parts := strings.Split(match, ".")
		var n int64
		for _, c := range parts[0] {
			n = n*10 + int64(c-'0')
		}
		result := numberToWords(n)
		if n == 1 {
			result += " dollar"
		} else {
			result += " dollars"
		}
		if len(parts) == 2 && parts[1] != "00" {
			var cents int64
			for _, c := range parts[1] {
				cents = cents*10 + int64(c-'0')
			}
			result += " and " + numberToWords(cents)
			if cents == 1 {
				result += " cent"
			} else {
				result += " cents"
			}
		}
		return result
	})
}

var timeRe = regexp.MustCompile(`\b(\d{1,2}):(\d{2})\s*(am|pm|AM|PM)?\b`)

func expandTime(text string) string {
	return timeRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := timeRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		var hour, minute int64
		for _, c := range parts[1] {
			hour = hour*10 + int64(c-'0')
		}
		for _, c := range parts[2] {
			minute = minute*10 + int64(c-'0')
		}
		var result string
		result = numberToWords(hour)
		if minute == 0 {
			if len(parts) > 3 && parts[3] != "" {
				result += " " + strings.ToLower(parts[3])
			} else {
				result += " o'clock"
			}
		} else if minute < 10 {
			result += " oh " + numberToWords(minute)
		} else {
			result += " " + numberToWords(minute)
		}
		if len(parts) > 3 && parts[3] != "" && minute != 0 {
			result += " " + strings.ToLower(parts[3])
		}
		return result
	})
}

var ordinalRe = regexp.MustCompile(`\b(\d+)(st|nd|rd|th)\b`)

var ordinalWords = map[int64]string{
	1: "first", 2: "second", 3: "third", 4: "fourth", 5: "fifth",
	6: "sixth", 7: "seventh", 8: "eighth", 9: "ninth", 10: "tenth",
	11: "eleventh", 12: "twelfth", 13: "thirteenth", 14: "fourteenth",
	15: "fifteenth", 16: "sixteenth", 17: "seventeenth", 18: "eighteenth",
	19: "nineteenth", 20: "twentieth", 30: "thirtieth", 40: "fortieth",
	50: "fiftieth", 60: "sixtieth", 70: "seventieth", 80: "eightieth",
	90: "ninetieth",
}

func expandOrdinals(text string) string {
	return ordinalRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := ordinalRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		var n int64
		for _, c := range parts[1] {
			n = n*10 + int64(c-'0')
		}
		if word, ok := ordinalWords[n]; ok {
			return word
		}
		if n > 20 && n < 100 {
			tens := (n / 10) * 10
			ones := n % 10
			if ones == 0 {
				if word, ok := ordinalWords[tens]; ok {
					return word
				}
			} else if word, ok := ordinalWords[ones]; ok {
				return tensWords[n/10] + " " + word
			}
		}
		return numberToWords(n) + "th"
	})
}

func normalizeQuotes(text string) string {
	text = strings.ReplaceAll(text, "\u201c", "\"")
	text = strings.ReplaceAll(text, "\u201d", "\"")
	text = strings.ReplaceAll(text, "\u2018", "'")
	text = strings.ReplaceAll(text, "\u2019", "'")
	text = strings.ReplaceAll(text, "\u00ab", "\"")
	text = strings.ReplaceAll(text, "\u00bb", "\"")
	return text
}

func normalizePunctuation(text string) string {
	text = strings.ReplaceAll(text, "\u2014", ", ")
	text = strings.ReplaceAll(text, "\u2013", ", ")
	text = strings.ReplaceAll(text, "\u2026", "...")
	text = strings.ReplaceAll(text, "\u2022", ",")
	return text
}
