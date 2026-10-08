package dream

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// The label is the whole mechanism, so its wording is not a detail.
//
// It said "보류된 갈래" first. Run against a real model, that produced
// "아직 최종 결정되지 않고 보류되어 있습니다" for a line the person had decided
// against — 보류 reads as "on hold", so the label said the opposite of what
// happened. The word is banned here rather than merely changed, because the
// obvious rewording is to reach for it again.
func TestAbandonedMarkerDoesNotReadAsUndecided(t *testing.T) {
	marker := branchMarker(&store.BranchRef{
		ID: uuid.New(), Name: "젠킨스로 이전", Status: store.BranchAbandoned,
		Resolution: "플러그인 호환 부담이 직접 통제로 얻는 것보다 컸습니다",
	})
	if strings.Contains(marker, "보류") {
		t.Errorf("the marker says 보류, which a model reads as not yet decided: %q", marker)
	}
	if !strings.Contains(marker, "채택하지 않기로") {
		t.Errorf("the marker does not say the line was decided against: %q", marker)
	}
	if !strings.Contains(marker, "플러그인 호환 부담") {
		t.Errorf("the marker drops the reason, leaving the model to invent one: %q", marker)
	}
}

// An open or adopted line is the ordinary state of a thought. Marking every
// source would push the model to talk about branches instead of answering.
func TestOnlyAbandonedLinesAreMarked(t *testing.T) {
	for _, status := range []string{store.BranchOpen, store.BranchAdopted} {
		marker := branchMarker(&store.BranchRef{
			ID: uuid.New(), Name: "격주 회고", Status: status, Resolution: "두 번 해 보니 괜찮았습니다",
		})
		if marker != "" {
			t.Errorf("a %s line was marked: %q", status, marker)
		}
	}
	if branchMarker(nil) != "" {
		t.Error("a thought in no branch was marked")
	}
}

// A reason long enough to be a document must not push the thoughts out of the
// context window, and neither must a branch name.
func TestMarkerIsBounded(t *testing.T) {
	marker := branchMarker(&store.BranchRef{
		ID: uuid.New(), Status: store.BranchAbandoned,
		Name:       strings.Repeat("가", 500),
		Resolution: strings.Repeat("나", 5000),
	})
	if length := len([]rune(marker)); length > 220 {
		t.Errorf("marker is %d runes; one source's label should not crowd out the sources", length)
	}
}
