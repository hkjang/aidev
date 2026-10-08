package intelligence

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"unicode"
)

const Dimensions = 192

// Embed creates a deterministic, language-agnostic character n-gram embedding.
// It works fully offline and is intentionally replaceable by a configured model later.
func Embed(text string) []float32 {
	normalized := []rune(strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, text)))
	features := make([]float32, Dimensions)
	add := func(token string, weight float32) {
		if strings.TrimSpace(token) == "" {
			return
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		features[h.Sum64()%Dimensions] += weight
	}
	words := strings.Fields(string(normalized))
	for _, word := range words {
		add("w:"+word, 1.8)
		r := []rune(word)
		for n := 1; n <= 3; n++ {
			for i := 0; i+n <= len(r); i++ {
				add(string(r[i:i+n]), float32(n)*.35)
			}
		}
	}
	var norm float64
	for _, v := range features {
		norm += float64(v * v)
	}
	if norm == 0 {
		return features
	}
	scale := float32(math.Sqrt(norm))
	for i := range features {
		features[i] /= scale
	}
	return features
}

func Cosine(a, b []float32) float64 {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	var dot float64
	for i := 0; i < limit; i++ {
		dot += float64(a[i] * b[i])
	}
	return clampSimilarity(dot)
}

// clampSimilarity holds a score inside the range a similarity is defined on.
//
// Shared so that the dense path and the compressed one in neighbours.go cannot
// round or clamp differently: the two must agree exactly, because one is only
// ever a faster way of computing the other.
func clampSimilarity(dot float64) float64 {
	return math.Max(0, math.Min(1, dot))
}

func Keywords(text string, limit int) []string {
	stop := map[string]bool{"그리고": true, "그러나": true, "하지만": true, "대한": true, "위한": true, "에서": true, "으로": true, "하면": true, "있다": true, "생각": true, "어떻게": true}
	counts := map[string]int{}
	for _, word := range strings.Fields(strings.ToLower(text)) {
		word = strings.TrimFunc(word, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
		if len([]rune(word)) >= 2 && !stop[word] {
			counts[word]++
		}
	}
	out := make([]string, 0, len(counts))
	for word := range counts {
		out = append(out, word)
	}
	sort.Slice(out, func(i, j int) bool {
		if counts[out[i]] == counts[out[j]] {
			return out[i] < out[j]
		}
		return counts[out[i]] > counts[out[j]]
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
