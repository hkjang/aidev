package docs

import "testing"

func TestScoutListQuoteObservation(t *testing.T) {
 for _, input := range []string{"- > 인용입니다.", "* >> 인용입니다.", "- >", "- > # 소제목", "- 매출 > 목표"} {
  got, err := Read("월간 보고서.md", []byte("# 제목\n\n"+input+"\n"))
  if err != nil { t.Fatal(err) }
  t.Logf("input=%q source=%q warnings=%v", input, got.Source, got.Warnings)
 }
 got, err := Read("월간 보고서.md", []byte("# 제목\n\n- > 인용입니다.\n"))
 if err != nil {t.Fatal(err)}
 want, err := Read("월간 보고서.md", []byte("# 제목\n\n- 인용입니다.\n"))
 if err != nil {t.Fatal(err)}
 if got.Source == want.Source {t.Fatal("candidate already fixed")}
}

