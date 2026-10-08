package presentation_test

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/presentation"
	"github.com/hkjang/umm/internal/store"
)

func exampleID(n byte) uuid.UUID { var u uuid.UUID; u[0] = n; u[15] = n; return u }

// A space someone actually worked in: a question they wrote down, an answer,
// the evidence under it, a disagreement they kept rather than deleted, what
// they decided to do next, and one thought held back from analysis.
func ExampleCompile() {
	thoughts := []presentation.Thought{
		{ID: exampleID(1), Content: "격주 회고가 정말 더 나은가?", Kind: "question"},
		{ID: exampleID(2), Content: "회고 주기를 격주로 줄여 보자", X: 700},
		{ID: exampleID(3), Content: "주기가 짧으면 논의가 얕아진다", X: 700, Y: 300},
		{ID: exampleID(4), Content: "지난 분기 회고 3회가 일정 때문에 취소됐다", X: 700, Y: 600},
		{ID: exampleID(5), Content: "격주로 하면 사이 기간의 맥락을 잊는다", X: 1400},
		{ID: exampleID(6), Content: "다음 스프린트에 4주만 시험해 보고 정한다", X: 2100},
		{ID: exampleID(7), Content: "개인적으로 회고가 싫다", X: 2800, AIExcluded: true},
	}
	links := []presentation.Link{
		{From: exampleID(2), To: exampleID(1), Relation: store.RelationAnswers},
		{From: exampleID(3), To: exampleID(2), Relation: store.RelationSupports},
		{From: exampleID(4), To: exampleID(3), Relation: store.RelationSupports},
		{From: exampleID(2), To: exampleID(5), Relation: store.RelationContradicts},
		{From: exampleID(5), To: exampleID(6), Relation: store.RelationFollows},
	}

	story := presentation.Compile(thoughts, links, presentation.Options{Title: "회고 주기 재검토"})
	fmt.Println(story.Summary())
	fmt.Print(presentation.WriteSource(story, presentation.SourceOptions{}))
	// Output:
	// 4 slides from 6 thoughts, 1 left out
	// # 회고 주기 재검토
	// @cover
	// > 격주 회고가 정말 더 나은가?
	//
	// # 격주 회고가 정말 더 나은가?
	// @section
	//
	// # 회고 주기를 격주로 줄여 보자
	// @comparison
	// - 격주로 하면 사이 기간의 맥락을 잊는다
	//
	// # 주기가 짧으면 논의가 얕아진다
	// @content
	// - 지난 분기 회고 3회가 일정 때문에 취소됐다
	//
	// # 다음 스프린트에 4주만 시험해 보고 정한다
	// @content
}
