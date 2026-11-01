package importer

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	spaceRegex      = regexp.MustCompile(`\s+`)
	apostropheRegex = regexp.MustCompile(`['‘’ʻʼ` + "`" + `]`)
)

// NormalizeClassName normalizes class name strings.
// It trims whitespace, converts Cyrillic lookalike letters to uppercase Latin,
// strips hyphens and spaces, so:
// "5-а", "5 A", "5А", " 5 - a " all become "5A".
func NormalizeClassName(raw string) (normalized string, display string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ""
	}

	// First pass: replace Cyrillic lookalikes
	var sb strings.Builder
	for _, r := range trimmed {
		switch r {
		// Cyrillic uppercase lookalikes to Latin
		case 'А', 'а':
			sb.WriteRune('A')
		case 'В', 'в':
			sb.WriteRune('B')
		case 'С', 'с':
			sb.WriteRune('C')
		case 'Е', 'е':
			sb.WriteRune('E')
		case 'Н', 'н':
			sb.WriteRune('H')
		case 'К', 'к':
			sb.WriteRune('K')
		case 'М', 'м':
			sb.WriteRune('M')
		case 'О', 'о':
			sb.WriteRune('O')
		case 'Р', 'р':
			sb.WriteRune('P')
		case 'Т', 'т':
			sb.WriteRune('T')
		case 'Х', 'х':
			sb.WriteRune('X')
		case 'Г', 'г':
			sb.WriteRune('G')
		case 'Д', 'д':
			sb.WriteRune('D')
		default:
			sb.WriteRune(unicode.ToUpper(r))
		}
	}

	str := sb.String()

	// Strip hyphens, dashes, spaces, underscores, periods
	var clean strings.Builder
	for _, r := range str {
		if r == '-' || r == '–' || r == '—' || r == ' ' || r == '_' || r == '.' || r == ',' {
			continue
		}
		clean.WriteRune(r)
	}

	normalized = clean.String()
	if normalized == "" {
		return "", ""
	}

	// For display name, format cleanly like "5A" or "5-A"
	// Extract leading digits and trailing letters
	digits := ""
	letters := ""
	for i, r := range normalized {
		if unicode.IsDigit(r) {
			digits += string(r)
		} else {
			letters = normalized[i:]
			break
		}
	}

	if digits != "" && letters != "" {
		display = digits + "-" + letters
	} else {
		display = normalized
	}

	return normalized, display
}

// NormalizeStudentName normalizes a student's full name:
// 1. Replaces various apostrophe characters (ʻ, ’, ‘, `, ') with a standard single quote (')
// 2. Collapses multiple spaces to single space and trims
// 3. Converts to Title Case for proper Uzbek Latin names
// 4. Returns display name ("Aliyev Vali G'anisher o'g'li") and normalized key for unique comparisons.
func NormalizeStudentName(raw string) (display string, normalizedKey string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ""
	}

	// Standardize apostrophes
	standardized := apostropheRegex.ReplaceAllString(trimmed, "'")

	// Collapse multiple spaces
	collapsed := spaceRegex.ReplaceAllString(standardized, " ")

	// Split words and Title-Case each word while preserving o' and g'
	words := strings.Fields(collapsed)
	var formattedWords []string
	for _, w := range words {
		formattedWords = append(formattedWords, titleCaseUzbekWord(w))
	}

	display = strings.Join(formattedWords, " ")
	normalizedKey = strings.ToLower(display)

	return display, normalizedKey
}

func titleCaseUzbekWord(w string) string {
	lower := strings.ToLower(w)
	// Special Uzbek auxiliary words like o'g'li, qizi
	if lower == "o'g'li" || lower == "og'li" || lower == "ogli" {
		return "o'g'li"
	}
	if lower == "qizi" {
		return "qizi"
	}

	runes := []rune(lower)
	if len(runes) == 0 {
		return ""
	}

	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
