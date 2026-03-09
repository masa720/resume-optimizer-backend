package service

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/lib/pq"
)

var nonWordRegex = regexp.MustCompile(`[^a-z0-9\s]+`)

var stopWords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "to": {}, "of": {}, "in": {}, "on": {},
	"with": {}, "for": {}, "is": {}, "are": {}, "be": {}, "as": {}, "at": {}, "by": {}, "from": {},
	"you": {}, "your": {}, "we": {}, "our": {}, "will": {}, "can": {}, "must": {}, "have": {},
	"has": {}, "had": {}, "that": {}, "this": {}, "it": {}, "they": {}, "their": {}, "who": {},
}

func ExtractKeywords(jobDescription string) pq.StringArray {
	words := strings.Fields(normalize(jobDescription))
	freq := map[string]int{}

	for _, word := range words {
		if len(word) < 3 {
			continue
		}
		if _, skip := stopWords[word]; skip {
			continue
		}
		freq[word]++
	}

	type item struct {
		word  string
		count int
	}
	items := make([]item, 0, len(freq))
	for k, v := range freq {
		items = append(items, item{word: k, count: v})
	}

	// Sort by frequency
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].word < items[j].word
		}

		return items[i].count > items[j].count
	})

	limit := 20
	if len(items) < limit {
		limit = len(items)
	}

	out := make(pq.StringArray, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, items[i].word)
	}

	return out
}

func MatchKeywords(keywords pq.StringArray, resumeText string) (pq.StringArray, pq.StringArray) {
	resumeSet := map[string]struct{}{}
	for _, word := range strings.Fields(normalize(resumeText)) {
		resumeSet[word] = struct{}{}
	}

	matched := make(pq.StringArray, 0)
	missing := make(pq.StringArray, 0)

	for _, kw := range keywords {
		if _, found := resumeSet[kw]; found {
			matched = append(matched, kw)
		} else {
			missing = append(missing, kw)
		}
	}

	return matched, missing
}

func CalculateMatchScore(matchedCount, total int) int {
	if total == 0 {
		return 0
	}
	score := int(math.Round((float64(matchedCount) / float64(total)) * 100))
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func GenerateSuggestions(missingKeywords pq.StringArray) []string {
	suggestions := []string{}
	for i, kw := range missingKeywords {
		if i >= 2 {
			break
		}
		suggestions = append(suggestions, "Add keyword '"+kw+"' with a concrete project/example.")
	}
	if len(suggestions) > 5 {
		return suggestions[:5]
	}
	return suggestions
}

func normalize(text string) string {
	l := strings.ToLower(text)
	c := nonWordRegex.ReplaceAllString(l, " ")
	return strings.TrimSpace(c)
}
