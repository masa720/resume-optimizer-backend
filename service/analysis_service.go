package service

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/lib/pq"
	"github.com/masa720/resume-optimizer-backend/domain"
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

// MatchStructuredSkills matches structured skills against resume text using substring matching.
func MatchStructuredSkills(skills []domain.StructuredSkill, resumeText string) (pq.StringArray, pq.StringArray) {
	resumeNorm := normalize(resumeText)
	matched := make(pq.StringArray, 0)
	missing := make(pq.StringArray, 0)

	for _, skill := range skills {
		if strings.Contains(resumeNorm, strings.ToLower(skill.Name)) {
			matched = append(matched, skill.Name)
		} else {
			missing = append(missing, skill.Name)
		}
	}

	return matched, missing
}

// CalculateWeightedMatchScore scores with required skills weighted 2x vs preferred skills.
func CalculateWeightedMatchScore(skills []domain.StructuredSkill, matchedSet map[string]bool) int {
	if len(skills) == 0 {
		return 0
	}

	totalWeight := 0
	matchedWeight := 0
	for _, s := range skills {
		w := 1
		if s.Importance == "required" {
			w = 2
		}
		totalWeight += w
		if matchedSet[strings.ToLower(s.Name)] {
			matchedWeight += w
		}
	}

	if totalWeight == 0 {
		return 0
	}

	score := int(math.Round(float64(matchedWeight) / float64(totalWeight) * 100))
	if score > 100 {
		return 100
	}
	return score
}

// CalculateSubScores computes category-level match scores from unified analysis skills.
func CalculateSubScores(skills []domain.StructuredSkill) domain.SubScores {
	type bucket struct{ matched, total int }
	hardReq := bucket{}
	hardPref := bucket{}
	soft := bucket{}

	for _, s := range skills {
		switch {
		case s.Category == "hard" && s.Importance == "required":
			hardReq.total++
			if s.Matched {
				hardReq.matched++
			}
		case s.Category == "hard" && s.Importance == "preferred":
			hardPref.total++
			if s.Matched {
				hardPref.matched++
			}
		default: // soft
			soft.total++
			if s.Matched {
				soft.matched++
			}
		}
	}

	pct := func(b bucket) int {
		if b.total == 0 {
			return 100
		}
		return CalculateMatchScore(b.matched, b.total)
	}

	hardReqScore := pct(hardReq)
	hardPrefScore := pct(hardPref)
	softScore := pct(soft)

	// Overall: required hard skills 50%, preferred hard 30%, soft 20%
	overall := int(math.Round(
		float64(hardReqScore)*0.50 +
			float64(hardPrefScore)*0.30 +
			float64(softScore)*0.20,
	))
	if overall > 100 {
		overall = 100
	}

	return domain.SubScores{
		HardSkillRequired:  hardReqScore,
		HardSkillPreferred: hardPrefScore,
		SoftSkill:          softScore,
		Overall:            overall,
	}
}

// ExtractMatchedMissing derives matched/missing keyword lists from unified skills.
func ExtractMatchedMissing(skills []domain.StructuredSkill) (matched, missing []string) {
	for _, s := range skills {
		if s.Matched {
			matched = append(matched, s.Name)
		} else {
			missing = append(missing, s.Name)
		}
	}
	return matched, missing
}

func normalize(text string) string {
	l := strings.ToLower(text)
	c := nonWordRegex.ReplaceAllString(l, " ")
	return strings.TrimSpace(c)
}
