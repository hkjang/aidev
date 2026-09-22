package server

import (
	"errors"
	"net/http"
)

type aiDocumentAction struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Instruction string `json:"-"`
}

// These are writing/research suggestions, never an authority to run a tool or
// edit a document. Applying an answer uses the ordinary versioned document API.
var aiDocumentActions = []aiDocumentAction{
	{"ask", "지식에 질문", "권한 있는 문서의 근거와 함께 답변을 받습니다.", "이 문서에서 중요한 내용을 설명해 주세요.", "질문에 직접 답하고 근거와 불확실성을 구분하세요."},
	{"write", "문서 초안", "목적·대상·구성을 입력해 Markdown 초안을 만듭니다.", "참고 자료를 바탕으로 목적, 핵심 내용, 다음 단계가 있는 문서 초안을 작성해 주세요.", "사용자의 목적과 대상에 맞는 Markdown 초안을 작성하세요. 참조에 없는 사실을 확정하지 말고 확인 필요로 표시하세요."},
	{"rewrite", "문장 개선", "의미를 유지하며 문장과 구조를 다듬습니다.", "핵심 의미와 수치, 코드, 링크를 유지하면서 문장을 명료하게 다듬어 주세요.", "의미·수치·인용·코드·링크를 임의로 바꾸지 말고 표현과 구조를 개선하세요. 중요한 변경 의도를 별도로 설명하세요."},
	{"summarize", "핵심 요약", "핵심 내용·결정·주의사항을 요약합니다.", "핵심 내용, 결정 사항, 주의할 점을 짧게 요약해 주세요.", "참고 자료의 핵심 주장·결정·주의사항을 요약하고 각 주장에 근거를 연결하세요."},
	{"translate", "번역", "대상 언어를 지정해 원문의 뜻과 형식을 유지합니다.", "이 문서를 영어로 번역해 주세요. 코드, 링크, 고유명사와 Markdown 구조는 유지해 주세요.", "사용자가 요청한 언어로 번역하세요. 코드·링크·수치·Markdown 구조를 보존하고 모호한 원문은 추측하지 마세요."},
	{"tags", "태그 추천", "중복 없는 주제 태그와 선택 이유를 제안합니다.", "문서를 설명하는 태그를 최대 8개 추천하고 각 태그의 이유를 알려 주세요.", "문서에서 근거를 찾을 수 있는 간결한 태그를 최대 8개 추천하세요. 태그는 제안일 뿐이며 자동 저장되지 않는다고 명시하세요."},
	{"links", "관련 링크 추천", "실제 검색된 문서 중 연결할 지식을 제안합니다.", "연결하면 도움이 될 문서와 연결 이유를 추천해 주세요.", "제공된 실제 참조 문서만 연결 대상으로 추천하세요. 문서 ID·URL·제목을 만들어내지 마세요. 부족하면 검색어 확장을 제안하세요. 관계는 사용자 확인 전 확정되지 않습니다."},
	{"duplicates", "중복 내용 검토", "찾은 문서 간 중복·차이·통합 후보를 검토합니다.", "검색된 문서에서 중복되는 내용과 차이를 비교하고 통합할 후보를 제안해 주세요.", "제공된 검색 결과 범위에서만 중복 및 차이를 검토하세요. 전체 워크스페이스를 검사한 것처럼 말하지 말고 자동 삭제나 통합을 제안 실행하지 마세요."},
	{"meeting", "회의록 정리", "논의·결정·담당자·실행 항목을 정리합니다.", "회의 내용을 논의, 결정, 실행 항목으로 정리해 주세요. 담당자와 기한이 없으면 미정으로 표시해 주세요.", "회의록을 논의·결정·실행 항목으로 정리하세요. 명시되지 않은 담당자·기한·동의·승인은 미정으로 표시하고 Markdown 체크리스트를 제안하세요."},
	{"template", "템플릿 만들기", "반복 업무에 쓸 문서 구조와 입력 안내를 만듭니다.", "이 문서와 같은 업무에 재사용할 템플릿을 만들어 주세요. 실제 개인정보 대신 입력 안내를 사용해 주세요.", "재사용 가능한 Markdown 템플릿과 입력 안내를 작성하세요. 개인·비밀 데이터는 예시로 복제하지 마세요. 실행 가능한 코드나 자동화가 아니라 문서 초안입니다."},
	{"gaps", "부족한 지식 찾기", "질문에 답하기 위해 보완할 지식을 제안합니다.", "이 주제에서 현재 자료만으로 답하기 어려운 질문과 보완할 문서를 제안해 주세요.", "현재 제공된 자료 범위의 지식 공백과 보완 질문을 제안하세요. 검색되지 않은 문서가 존재하지 않는다고 단정하지 마세요."},
}

func documentAIAction(id string) (aiDocumentAction, error) {
	if id == "" {
		id = "ask"
	}
	for _, action := range aiDocumentActions {
		if action.ID == id {
			return action, nil
		}
	}
	return aiDocumentAction{}, errors.New("지원하는 AI 작업을 선택하세요")
}

func (s *Server) listAIActions(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, 200, map[string]any{"actions": aiDocumentActions, "automatic_apply": false, "max_answer_bytes": 4 << 20})
}
