package intelligence

import "testing"

func TestEmbeddingSimilarity(t *testing.T) {
	a := Embed("사용자별 API 키 권한 정책")
	b := Embed("API Key 사용자 권한 모델")
	c := Embed("점심 메뉴로 비빔밥을 먹자")
	if Cosine(a, b) <= Cosine(a, c) {
		t.Fatalf("related thoughts should be more similar: related=%f unrelated=%f", Cosine(a, b), Cosine(a, c))
	}
}
func TestEmbeddingNormalized(t *testing.T) {
	v := Embed("AI Gateway 비용과 권한")
	score := Cosine(v, v)
	if score < .99 || score > 1.001 {
		t.Fatalf("embedding is not normalized: %f", score)
	}
}
