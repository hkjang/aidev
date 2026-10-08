package intelligence

import (
	"context"
	"fmt"
	"math"
)

// The embedding layer decides which thoughts umm calls related, which ones it
// clusters, and which pair a Dream is built from. Whether those decisions carry
// meaning or only vocabulary is not visible from the outside, so this file makes
// it measurable — in tests, and in the administrator screen where the person who
// can change the backend will actually see it.

// PairClass labels what kind of relationship a measurement pair represents.
type PairClass string

const (
	// ClassParaphrase: the same meaning in deliberately different words. These
	// must score high, or related thoughts miss the connections worth surfacing.
	ClassParaphrase PairClass = "paraphrase"
	// ClassRelated: different claims about the same subject.
	ClassRelated PairClass = "related"
	// ClassLexicalDecoy: unrelated meaning, shared vocabulary. The trap for any
	// algorithm that scores surface overlap.
	ClassLexicalDecoy PairClass = "lexical-decoy"
	// ClassUnrelated: no meaningful relationship. The floor.
	ClassUnrelated PairClass = "unrelated"
)

// SemanticPair is one labelled comparison.
type SemanticPair struct {
	Class PairClass
	A, B  string
}

// QualityPairs is a small hand-labelled set drawn from the kind of notes umm
// actually holds: working thoughts about a product, in Korean and English.
//
// It is not a benchmark of record. It is a fence against regression and a way to
// compare one embedding backend against another on identical ground, which is
// what an operator needs before deciding whether a model is worth configuring.
var QualityPairs = []SemanticPair{
	{ClassParaphrase, "외부 API 없이 사내망에서만 모델을 돌린다", "폐쇄망 내부 GPU 서버로 추론을 처리한다"},
	{ClassParaphrase, "메모를 고칠 때마다 이전 상태를 남긴다", "편집 시점마다 스냅샷을 보관한다"},
	{ClassParaphrase, "밤사이 쌓인 생각에서 새 연결을 찾아준다", "자는 동안 기록을 훑어 관계를 발견한다"},
	{ClassParaphrase, "로그인 실패가 반복되면 잠시 막는다", "인증 시도가 계속 틀리면 일시적으로 차단한다"},
	{ClassParaphrase, "the database stores every note revision", "each edit to a memo is kept as a snapshot"},
	{ClassParaphrase, "search should work without any network access", "lookups must succeed on a fully isolated machine"},
	{ClassParaphrase, "팀원마다 볼 수 있는 범위를 다르게 준다", "구성원별로 접근 권한을 구분한다"},
	{ClassParaphrase, "오래 안 본 기록을 다시 꺼내 보여준다", "잊고 있던 메모를 되살려 제시한다"},

	{ClassRelated, "임베딩 모델을 바꾸면 다시 계산해야 한다", "벡터 차원이 달라지면 기존 인덱스를 못 쓴다"},
	{ClassRelated, "캔버스에서 생각을 드래그해 배치한다", "포스트잇 위치를 자유롭게 옮길 수 있다"},
	{ClassRelated, "webhook payload is signed with HMAC", "receivers must verify the signature header"},
	{ClassRelated, "Dream은 하루에 한 번 생성된다", "야간 배치가 사용자별로 돌아간다"},

	{ClassLexicalDecoy, "PostgreSQL 백업 주기를 하루로 정했다", "PostgreSQL 로고 색상이 마음에 든다"},
	{ClassLexicalDecoy, "임베딩 모델 교체 계획을 세웠다", "모델하우스 임베딩 광고를 봤다"},
	{ClassLexicalDecoy, "the memory graph needs typed edges", "the graph on the wall of the memory ward"},
	{ClassLexicalDecoy, "검색 결과를 점수 순으로 정렬한다", "검색대에서 결과를 기다리며 줄을 섰다"},
	{ClassLexicalDecoy, "캔버스 확대 배율을 조정했다", "캔버스 천으로 만든 가방을 샀다"},
	{ClassLexicalDecoy, "token limit을 4096으로 올렸다", "지하철 token을 잃어버렸다"},

	{ClassUnrelated, "고객 인터뷰를 매주 정리한다", "쿠버네티스 인그레스 설정"},
	{ClassUnrelated, "점심에 김치찌개를 먹었다", "분산 트랜잭션의 격리 수준"},
	{ClassUnrelated, "the release pipeline builds an image", "he practised the violin for an hour"},
	{ClassUnrelated, "주말에 자전거를 탔다", "TLS 인증서 만료일 확인"},
}

// A backend can win on paraphrase pairs and still be useless for the two
// features people actually see. Related thoughts and clustering do not ask "are
// these two sentences the same claim" — they ask "of everything in this space,
// which belongs with which". paraphrase-multilingual scores 85.4% on the pairs
// above and still puts "라이딩이 끝나면 스트레칭을 꼭 한다" closer to an auth-token
// note than that note's own topic-mates, which collapses a two-topic space into
// one cluster. The pairs alone never see it, so the topics below are measured too.

// TopicGroup is a set of sentences that belong together.
type TopicGroup struct {
	Topic     string
	Sentences []string
}

// QualityTopics holds clearly distinct subjects, the shape a workspace has when
// someone has been thinking about more than one thing.
var QualityTopics = []TopicGroup{
	{"auth", []string{
		"인증 토큰 만료 시간을 24시간으로 정했다",
		"세션 쿠키는 HttpOnly와 SameSite를 함께 설정한다",
		"로그인 실패가 반복되면 계정을 일시적으로 잠근다",
		"비밀번호는 해시로만 저장하고 원문은 남기지 않는다",
	}},
	{"cycling", []string{
		"주말에 자전거를 타고 한강을 따라 달렸다",
		"자전거 체인에 기름을 새로 발랐다",
		"라이딩이 끝나면 스트레칭을 꼭 한다",
		"이번 주말에는 남산 업힐을 다녀올 생각이다",
	}},
	{"hiring", []string{
		"채용 공고에 필요한 경험을 더 구체적으로 적었다",
		"면접에서는 실제로 해 본 일을 묻는 편이 낫다",
		"레퍼런스 체크는 합격 통보 전에 끝낸다",
		"온보딩 첫 주에 무엇을 읽게 할지 정리했다",
	}},
	{"cooking", []string{
		"김치찌개는 돼지고기를 먼저 볶아야 깊은 맛이 난다",
		"파스타 면은 봉지에 적힌 시간보다 1분 덜 삶는다",
		"주말에 쓸 재료를 미리 손질해 냉장고에 넣어 둔다",
		"오븐은 예열이 끝난 뒤에 넣어야 겉이 마르지 않는다",
	}},
}

// ClassScore summarises one relationship class.
type ClassScore struct {
	Class PairClass `json:"class"`
	Mean  float64   `json:"mean"`
	Min   float64   `json:"min"`
	Max   float64   `json:"max"`
	Count int       `json:"count"`
}

// QualityReport is what the measurement produces. Discrimination and
// PairwiseAccuracy are the two numbers that matter: a backend that scores
// vocabulary above meaning has a negative discrimination and an accuracy near
// zero, and every feature built on it is lexical no matter what it is called.
type QualityReport struct {
	Algorithm string `json:"algorithm"`
	// Model is empty when the offline algorithm produced the vectors.
	Model   string       `json:"model"`
	Classes []ClassScore `json:"classes"`
	// Discrimination is mean(paraphrase) - mean(lexical-decoy). Positive means
	// the backend can tell meaning from shared vocabulary.
	Discrimination float64 `json:"discrimination"`
	// PairwiseAccuracy is the fraction of (paraphrase, decoy) comparisons ranked
	// the right way round. It does not depend on where a threshold sits.
	PairwiseAccuracy float64 `json:"pairwiseAccuracy"`
	Pairs            int     `json:"pairs"`
	// TopicSeparation is mean(same-topic similarity) - mean(different-topic
	// similarity) across the labelled topic groups.
	TopicSeparation float64 `json:"topicSeparation"`
	// NeighbourPurity is the fraction of sentences whose single closest match
	// belongs to the same topic. This is the question related thoughts asks, so a
	// backend that fails it produces visibly wrong suggestions however well it
	// scores on isolated pairs.
	NeighbourPurity float64 `json:"neighbourPurity"`
	Sentences       int     `json:"sentences"`
	// AccuracyBar and PurityBar are the thresholds this verdict was reached
	// against, so a reader can see what "semantic" meant on this deployment
	// rather than assuming the shipped defaults.
	AccuracyBar float64 `json:"accuracyBar"`
	PurityBar   float64 `json:"purityBar"`
	// Semantic reports whether this backend is usable as a semantic layer at all.
	Semantic bool `json:"semantic"`
}

// QualityBars is where a backend has to land before umm will treat it as a
// semantic layer. They are settings rather than constants because an operator
// running an unusual corpus — one language, one narrow domain — may legitimately
// need a different line, and because the only honest way to pick one is to
// measure. The defaults below are the values umm was measured against.
type QualityBars struct {
	// Accuracy is the fraction of (paraphrase, lexical-decoy) comparisons the
	// backend must rank the right way round.
	Accuracy float64
	// Purity is the share of sentences whose nearest match must share their
	// topic — the question related thoughts and clustering actually ask.
	Purity float64
}

// DefaultQualityBars are the measured values. The offline algorithm scores 4.2%
// accuracy and 18.8% purity; every sentence embedding model measured so far
// scores 81-85% and 69-88%. The bars sit in the empty space between them.
func DefaultQualityBars() QualityBars {
	return QualityBars{Accuracy: defaultSemanticAccuracyBar, Purity: defaultSemanticPurityBar}
}

// normalized keeps a stored setting inside the range where it means anything. A
// bar outside 0..1 is not a stricter rule, it is a rule that can never be met or
// never fails.
func (b QualityBars) normalized() QualityBars {
	if b.Accuracy <= 0 || b.Accuracy > 1 {
		b.Accuracy = defaultSemanticAccuracyBar
	}
	if b.Purity <= 0 || b.Purity > 1 {
		b.Purity = defaultSemanticPurityBar
	}
	return b
}

// defaultSemanticAccuracyBar is the point below which a backend is not telling
// meaning from vocabulary well enough for related thoughts, clustering or Dream
// pair selection to mean what their names say. The offline algorithm scores 4.2%.
const defaultSemanticAccuracyBar = 0.65

// defaultSemanticPurityBar separates a backend that groups subjects from one that
// does not. The gap is wide and unambiguous: the offline character n-gram algorithm
// lands at 18.8%, while every sentence embedding model measured so far scores
// between 68.8% and 87.5%. The bar sits in the empty space between them.
//
// It is not a quality ranking. Models that score well here still disagree with
// each other on individual sentences, and a higher number does not reliably mean
// better clusters in a given workspace — see docs/ADMIN_GUIDE.md, where the
// candidates are compared on this and on umm's end-to-end clustering test, which
// they do not rank the same way.
const defaultSemanticPurityBar = 0.6

// MeasureQuality scores every labelled pair with the supplied provider.
//
// It embeds all sides in a single batch so a remote backend is contacted once
// rather than once per pair, and so the report cannot be skewed by a provider
// that behaves differently on small and large requests.
func MeasureQuality(ctx context.Context, provider Provider, bars QualityBars) (QualityReport, error) {
	bars = bars.normalized()
	texts := make([]string, 0, len(QualityPairs)*2)
	for _, pair := range QualityPairs {
		texts = append(texts, pair.A, pair.B)
	}
	pairTexts := len(texts)
	// Topic sentences ride along in the same batch, so both halves of the report
	// describe one call to one backend.
	topicOf := []int{}
	for index, group := range QualityTopics {
		for _, sentence := range group.Sentences {
			texts = append(texts, sentence)
			topicOf = append(topicOf, index)
		}
	}
	vectors, algorithm := provider.Embed(ctx, texts)
	if len(vectors) != len(texts) {
		return QualityReport{}, fmt.Errorf("embedding backend returned %d vectors for %d texts", len(vectors), len(texts))
	}
	report := QualityReport{Algorithm: algorithm, Pairs: len(QualityPairs), Sentences: len(topicOf)}
	if algorithm != LocalAlgorithm {
		report.Model = provider.Model()
	}

	byClass := map[PairClass][]float64{}
	for index, pair := range QualityPairs {
		score := Cosine(vectors[index*2], vectors[index*2+1])
		byClass[pair.Class] = append(byClass[pair.Class], score)
	}
	for _, class := range []PairClass{ClassParaphrase, ClassRelated, ClassLexicalDecoy, ClassUnrelated} {
		values := byClass[class]
		if len(values) == 0 {
			continue
		}
		summary := ClassScore{Class: class, Min: values[0], Max: values[0], Count: len(values)}
		for _, value := range values {
			summary.Mean += value
			summary.Min = math.Min(summary.Min, value)
			summary.Max = math.Max(summary.Max, value)
		}
		summary.Mean /= float64(len(values))
		report.Classes = append(report.Classes, summary)
	}
	report.Discrimination = classMean(byClass, ClassParaphrase) - classMean(byClass, ClassLexicalDecoy)
	report.PairwiseAccuracy = pairwiseAccuracy(byClass)
	report.TopicSeparation, report.NeighbourPurity = topicScores(vectors[pairTexts:], topicOf)
	return report.WithBars(bars), nil
}

// WithBars re-decides the verdict against a different set of thresholds.
//
// The measurements themselves do not depend on where the bars sit, so a stored
// report can be re-judged without embedding anything again. That matters because
// the report is cached: an administrator who raises the bar must see the effect
// immediately, not once the cache expires, and a second instance that never saw
// the settings change must reach the same conclusion from the same numbers.
func (r QualityReport) WithBars(bars QualityBars) QualityReport {
	bars = bars.normalized()
	r.AccuracyBar, r.PurityBar = bars.Accuracy, bars.Purity
	r.Semantic = r.Discrimination > 0 &&
		r.PairwiseAccuracy >= bars.Accuracy &&
		r.NeighbourPurity >= bars.Purity
	return r
}

// topicScores measures what related thoughts and clustering actually do: how far
// apart the subjects sit, and whether each sentence's closest match belongs with
// it. Purity is the stricter of the two — a single wrong nearest neighbour is
// enough to merge two topics into one cluster.
func topicScores(vectors [][]float32, topicOf []int) (separation, purity float64) {
	if len(vectors) != len(topicOf) || len(topicOf) < 2 {
		return 0, 0
	}
	withinTotal, withinCount := 0.0, 0
	acrossTotal, acrossCount := 0.0, 0
	correct := 0
	for i := range vectors {
		bestScore, bestTopic := math.Inf(-1), -1
		for j := range vectors {
			if i == j {
				continue
			}
			score := float64(Cosine(vectors[i], vectors[j]))
			if topicOf[i] == topicOf[j] {
				withinTotal += score
				withinCount++
			} else {
				acrossTotal += score
				acrossCount++
			}
			if score > bestScore {
				bestScore, bestTopic = score, topicOf[j]
			}
		}
		if bestTopic == topicOf[i] {
			correct++
		}
	}
	if withinCount > 0 && acrossCount > 0 {
		separation = withinTotal/float64(withinCount) - acrossTotal/float64(acrossCount)
	}
	return separation, float64(correct) / float64(len(vectors))
}

func classMean(byClass map[PairClass][]float64, class PairClass) float64 {
	values := byClass[class]
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

// pairwiseAccuracy asks the only question that matters for retrieval: given one
// paraphrase and one lexical decoy, does the backend rank the paraphrase higher?
// Every pairing is compared, so the answer is independent of any threshold.
func pairwiseAccuracy(byClass map[PairClass][]float64) float64 {
	paraphrases, decoys := byClass[ClassParaphrase], byClass[ClassLexicalDecoy]
	if len(paraphrases) == 0 || len(decoys) == 0 {
		return 0
	}
	correct, total := 0, 0
	for _, paraphrase := range paraphrases {
		for _, decoy := range decoys {
			total++
			if paraphrase > decoy {
				correct++
			}
		}
	}
	return float64(correct) / float64(total)
}
