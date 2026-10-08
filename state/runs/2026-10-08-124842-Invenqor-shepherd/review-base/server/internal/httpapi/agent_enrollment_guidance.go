package httpapi

// Enrollment failures are the one class of error whose audience is never
// looking at this process' stdout: the Agent is on another machine and the
// operator is in the console. Every code therefore carries both a operator
// readable summary and the concrete next action.
var enrollmentGuidance = map[string][2]string{
	"AGENT_ENROLLMENT_READY": {
		"이 출처에서 Agent가 등록할 수 있습니다.",
		"조치가 필요하지 않습니다.",
	},
	"AGENT_ENROLLMENT_SUCCEEDED": {
		"Agent가 등록되었고 호스트 자산이 생성되었습니다.",
		"조치가 필요하지 않습니다.",
	},
	"AGENT_PREFLIGHT_READY": {
		"Agent가 등록 경로를 정상적으로 확인했습니다.",
		"조치가 필요하지 않습니다.",
	},
	"AGENT_AUTO_ENROLLMENT_DISABLED": {
		"Agent 자동 등록이 꺼져 있습니다.",
		"설정 > Agent 등록 을 URL 전용 또는 토큰 모드로 바꾸거나, " +
			"Agent 화면에서 해당 장비를 직접 등록하십시오.",
	},
	"AGENT_SOURCE_NOT_ALLOWED": {
		"등록 정책이 Agent의 출발지 IP를 거부했습니다.",
		"보고된 출발지 IP 또는 그 CIDR을 등록 허용 목록에 추가하거나, " +
			"앞단의 프록시를 신뢰 프록시로 등록하십시오.",
	},
	"AGENT_ENROLLMENT_UNAUTHORIZED": {
		"공용 등록 자격증명이 거부되었습니다.",
		"콘솔에서 등록 토큰을 발급하고 같은 값을 Agent의 " +
			"server.enrollment_token_file 에 기록하십시오.",
	},
	"AGENT_ENROLLMENT_RATE_LIMITED": {
		"Agent 등록 시도가 너무 많이 수신되었습니다.",
		"Agent가 너무 빠르게 재시도하고 있습니다. 호스트당 Agent 프로세스가 " +
			"하나인지 확인하고 재시도 창이 초기화될 때까지 기다리십시오.",
	},
	"AGENT_ALREADY_CLAIMED": {
		"Agent 식별자가 다른 장비의 device claim에 묶여 있습니다.",
		"Agent 상태 디렉터리가 이미지에 복제되었습니다. 복제본에서 agent-id 와 " +
			"enrollment-claim.json 을 삭제하고 다시 시작하십시오.",
	},
	"AGENT_BLOCKED": {
		"Agent가 차단되어 있습니다.",
		"해당 장비를 다시 신뢰한다면 Agent 화면에서 차단을 해제하십시오.",
	},
	"INVALID_AGENT_IDENTITY": {
		"Agent 등록 신원이 올바르지 않습니다.",
		"Agent가 잘못된 형식의 agent_id 또는 claim 을 보냈습니다. Agent를 " +
			"업그레이드하고 손으로 편집한 agent-id 파일을 삭제하십시오.",
	},
	"INVALID_AGENT_SOURCE_ADDRESS": {
		"Agent의 출발지 주소를 확인할 수 없습니다.",
		"프록시가 해석할 수 없는 X-Forwarded-For 값을 보냈습니다. 프록시를 " +
			"바로잡거나 신뢰 프록시 목록에서 제거하십시오.",
	},
	"INVALID_AGENT_HOSTNAME": {
		"Agent 호스트명이 지원 길이를 초과했습니다.",
		"호스트명을 255자 이하로 줄이십시오.",
	},
	"INVALID_AGENT_REQUEST": {
		"Agent 등록 요청 본문을 해석할 수 없습니다.",
		"프록시나 WAF가 요청 본문을 다시 쓰고 있습니다. /v1/agent/ 경로에 " +
			"대해서는 우회하도록 설정하십시오.",
	},
	"AGENT_ENROLLMENT_POLICY_UNAVAILABLE": {
		"자동 등록 정책을 불러오지 못했습니다.",
		"메타데이터 데이터베이스에 연결할 수 없습니다. 설정 > PostgreSQL 과 " +
			"데이터베이스 health 엔드포인트를 확인하십시오.",
	},
	"AGENT_ENROLLMENT_FAILED": {
		"Agent 또는 호스트 자산을 생성하는 중 Server에서 실패했습니다.",
		"기록된 오류 상세를 확인하십시오. 대개 데이터베이스 권한이나 " +
			"마이그레이션 문제입니다.",
	},
	"AGENT_ENDPOINT_NOT_FOUND": {
		"Agent가 이 Server에 없는 경로를 호출했습니다.",
		"Agent의 server.url 이 잘못된 경로나 다른 제품을 가리킵니다. " +
			"스킴·호스트·포트만 설정하십시오.",
	},
	"AGENT_ENDPOINT_METHOD_NOT_ALLOWED": {
		"Agent가 Agent 엔드포인트에 잘못된 HTTP 메서드를 사용했습니다.",
		"프록시가 메서드를 다시 쓰고 있습니다. /v1/agent/ 경로에 POST를 " +
			"허용하십시오.",
	},
	"AGENT_UNAUTHORIZED": {
		"Agent의 device 자격증명이 거부되었습니다.",
		"Agent 상태 디렉터리의 device-credential.json 을 삭제해 Agent가 다시 " +
			"등록하게 하거나, Agent 화면에서 토큰을 교체하십시오.",
	},
	"AGENT_PREFLIGHT_BLOCKED": {
		"Agent가 등록 경로를 확인했고 사용할 수 없는 상태입니다.",
		"기록된 사유를 여십시오. 이 Agent를 거부하는 정확한 정책이 적혀 " +
			"있습니다.",
	},
}

// enrollmentSummary is what a code means, or "" for a code with no entry. The
// two callers want different wording when there is none, so the fallback is
// theirs and the lookup is in one place.
//
// The summary used to be written out again at every recording site as well.
// Twelve of those matched this table exactly and three had drifted from it, so
// the same failure read one way in the diagnostic log and another in the
// enrollment panel. A site still passes its own sentence where it genuinely
// knows more - a blocked Agent trying to enrol and a blocked Agent trying to
// send inventory are different events sharing one code.
func enrollmentSummary(code string) string {
	if guidance, found := enrollmentGuidance[code]; found {
		return guidance[0]
	}
	return ""
}

func enrollmentRejectionMessage(code string) string {
	if summary := enrollmentSummary(code); summary != "" {
		return summary
	}
	return "The Agent enrollment attempt was rejected."
}

func enrollmentRemediation(code string) string {
	if guidance, found := enrollmentGuidance[code]; found {
		return guidance[1]
	}
	return "Open the recorded diagnostic detail for this event code."
}
