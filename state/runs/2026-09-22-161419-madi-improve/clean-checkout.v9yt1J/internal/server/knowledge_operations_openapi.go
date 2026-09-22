package server

func knowledgeOperationsOpenAPI(paths, schemas map[string]any) {
	text := map[string]any{"type": "string"}
	boolean := map[string]any{"type": "boolean"}
	yes := map[string]any{"type": "boolean", "const": true}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	version := map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
	revision := map[string]any{"type": "integer", "minimum": 0, "maximum": 2147483647}
	object := func(p map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": p, "required": required}
	}
	choice := func(values ...string) map[string]any { return map[string]any{"type": "string", "enum": values} }
	array := func(item any, min, max int) map[string]any {
		return map[string]any{"type": "array", "minItems": min, "maxItems": max, "items": item}
	}
	docref := object(map[string]any{"id": uuid, "version": version}, "id", "version")
	schemas["EvidencePolicy"] = object(map[string]any{"version": version, "enabled": boolean, "retention_days": map[string]any{"type": "integer", "minimum": 1, "maximum": 3650}}, "version", "enabled", "retention_days")
	schemas["EvidenceSave"] = object(map[string]any{"ticket": text, "consent": yes}, "ticket", "consent")
	schemas["EvidenceReview"] = object(map[string]any{"version": version, "status": choice("unreviewed", "supported", "insufficient", "misinterpreted"), "note": map[string]any{"type": "string", "description": "UTF-8 최대8000바이트. 의견에도 현재 민감정보 정책 적용"}}, "version", "status")
	schemas["EvidenceDelete"] = object(map[string]any{"version": version, "confirmation": map[string]any{"const": "DELETE"}}, "version", "confirmation")
	schemas["KnowledgePackagePolicy"] = object(map[string]any{"version": version, "enabled": boolean, "retention_hours": map[string]any{"type": "integer", "minimum": 1, "maximum": 168}, "token_counter": choice("estimate", "responses"), "allow_http": boolean}, "version", "enabled", "retention_hours", "token_counter", "allow_http")
	schemas["KnowledgePolicyRestore"] = object(map[string]any{"version": version, "consent": yes}, "version", "consent")
	packageDoc := object(map[string]any{"id": uuid, "version": version, "mandatory": boolean, "reason": text}, "id", "version")
	schemas["KnowledgePackageAttachment"] = packageAttachmentOpenAPISchema()
	schemas["KnowledgePackageCreate"] = object(map[string]any{"workspace_id": uuid, "purpose": text, "allowed_scope": text, "model": text, "counter": choice("estimate", "responses"), "token_budget": map[string]any{"type": "integer", "minimum": 512, "maximum": 262144}, "documents": array(packageDoc, 0, 32), "attachments": array(map[string]any{"$ref": "#/components/schemas/KnowledgePackageAttachment"}, 0, 32), "consent": yes, "model_consent": boolean}, "workspace_id", "purpose", "model", "counter", "token_budget", "consent")
	schemas["KnowledgePackageCreate"].(map[string]any)["description"] = "documents와attachments 선택 합계는1~32입니다. 첨부는 자동포함되지 않습니다. 현재 위치/구간/hash/추출버전을 확인한 명시선택만 포함합니다."
	schemas["KnowledgePackageExport"] = object(map[string]any{"target": map[string]any{"type": "string", "description": "명시 전달 대상, UTF-8 1~200바이트"}, "consent": yes}, "target", "consent")
	schemas["KnowledgePackageDelete"] = object(map[string]any{"confirmation": map[string]any{"const": "DELETE"}}, "confirmation")
	schemas["KnowledgeImpactCheck"] = object(map[string]any{"workspace_id": uuid, "documents": array(docref, 1, 501), "packages": array(uuid, 0, 100), "agents": array(object(map[string]any{"id": uuid, "revision": version}, "id", "revision"), 0, 100)}, "workspace_id", "documents")
	schemas["KnowledgeImpactRequest"] = object(map[string]any{"source_version": version, "target_id": uuid, "target_version": version, "relation_type": choice("reference", "related", "policy", "execution", "data")}, "source_version", "target_id", "target_version", "relation_type")
	schemas["KnowledgeImpactDecision"] = object(map[string]any{"revision": version, "target_version": version, "status": choice("pending", "no_impact", "needs_change", "done", "exception_requested"), "note": map[string]any{"type": "string", "description": "필수, UTF-8 최대4000바이트"}}, "revision", "target_version", "status", "note")
	schemas["KnowledgeProposalCreate"] = object(map[string]any{"base_version": version, "markdown": map[string]any{"type": "string", "description": "최대2MiB, 기준+제안 JSON합계도2MiB 이하"}, "reason": map[string]any{"type": "string", "description": "UTF-8 1~4000바이트"}, "provenance": choice("human", "ai_assisted"), "consent": yes}, "base_version", "markdown", "reason", "provenance", "consent")
	schemas["KnowledgeProposalMerge"] = object(map[string]any{"revision": version, "consent": yes, "note": text}, "revision", "consent")
	schemas["KnowledgeProposalDecision"] = object(map[string]any{"revision": version, "status": choice("withdrawn", "rejected"), "note": text}, "revision", "status", "note")
	schemas["KnowledgeValidPeriod"] = object(map[string]any{"version": version, "from": map[string]any{"type": "string", "format": "date", "description": "시작일 포함"}, "until": map[string]any{"type": "string", "description": "YYYY-MM-DD 종료일 제외, 빈 문자열은 무기한"}}, "version", "from", "until")
	schemas["KnowledgeValiditySave"] = object(map[string]any{"revision": revision, "document_version": version, "periods": array(map[string]any{"$ref": "#/components/schemas/KnowledgeValidPeriod"}, 0, 100), "reason": text, "consent": yes}, "revision", "document_version", "periods", "reason", "consent")
	schemas["KnowledgeTimeCheck"] = object(map[string]any{"workspace_id": uuid, "protection_revision": version, "documents": array(object(map[string]any{"id": uuid, "version": version, "revision": revision}, "id", "version", "revision"), 0, 52)}, "workspace_id", "protection_revision", "documents")
	for _, entry := range []struct {
		path, method, summary, schema, description string
		personal                                   bool
	}{
		{"/admin/evidence-policy", "put", "근거 보관 정책 CAS 저장", "EvidencePolicy", "기본 비활성. 보존 축소는 기존 만료시간에도 적용하며 연장은 만료 기록을 부활시키지 않습니다.", true},
		{"/ai/evidence", "post", "완료 답변의 당시 원문 근거 별도 보관", "EvidenceSave", "실제 완료 티켓의 현재 공급자·출처 버전·바이트 범위·해시·권한을 검증합니다. 같은 티켓 중복 저장은 멱등. 근거 없는 답변은422. 원문·모델·답변 암호화 사본 최대2MiB.", true},
		{"/ai/evidence", "get", "본인의 현재 접근 가능한 근거 목록", "", "workspace_id 필수, 최근100개. 현재 모든 출처 ACL·TTL·정책을 만족하는 기록만 조회합니다.", true},
		{"/ai/evidence/{id}", "get", "당시 근거·최신성·사람 검토를 분리하여 조회", "", "현재 원문이 변경돼도 보관된 구간을 과거 버전으로 표시합니다. 현재 출처 접근권한은 우회하지 않습니다. 해시 일치는 사실성 인증이 아닙니다.", true},
		{"/ai/evidence/{id}/reviews", "post", "사람의 근거 검토 CAS 기록", "EvidenceReview", "본인 근거에 검토 결과와 암호화 의견을 별도로 기록합니다.", true},
		{"/ai/evidence/{id}", "delete", "본인 근거 삭제", "EvidenceDelete", "보존/법적 보존 정책상 제거 불가능하면409. 삭제 동의·현재 version 필수.", true},
		{"/admin/knowledge-packages/policy", "put", "지식 패키지 운영 정책 CAS 저장", "KnowledgePackagePolicy", "estimate는 UTF-8 byte 기반 로컬 추정. responses는 실제 공급자 /responses/input_tokens API를 사용합니다. HTTP는 별도 관리자 허용이 필요합니다.", true},
		{"/admin/evidence-policy/history/{version}/restore", "post", "근거 정책 이력을 새 revision으로 재적용", "KnowledgePolicyRestore", "경로는 복원할 과거 version, 본문 version은 현재 CAS. 보존기간 복원으로 만료 사본이 부활하지 않습니다.", true},
		{"/admin/knowledge-packages/policy/history/{version}/restore", "post", "패키지 정책 이력을 새 revision으로 재적용", "KnowledgePolicyRestore", "경로는 복원할 과거 version, 본문 version은 현재 CAS. 보존기간 복원으로 만료 사본이 부활하지 않습니다.", true},
		{"/knowledge/packages", "post", "근거·필수 정책·토큰 예산으로 지식 패키지 구성", "KnowledgePackageCreate", "document:read 필수. responses 계산은 ai:execute 및 model_consent:true 추가 필요. 실제 최종 입력 전체를 최대10회 계산하고 조용히 추정으로 대체하지 않습니다. 필수 문서 전체가 예산을 초과하면 거절. 원문 후보≤2MiB, 보관 JSON≤4MiB. 도구 실행 권한은 부여하지 않습니다.", false},
		{"/knowledge/packages", "get", "본인의 현재 권한 범위 패키지 목록", "", "workspace_id 필수, 최근100개. TTL/모든 출처 현재 ACL 적용.", false},
		{"/knowledge/packages/{id}", "get", "패키지 만료·최신성·근거 검사", "", "API키는 원문 없이 metadata만 받습니다. 원문 전달은 명시한 export로 분리합니다. 브라우저 개인 조회도 현재 ACL을 적용합니다.", false},
		{"/knowledge/packages/{id}/export", "post", "현재 권한·버전 검증 후 패키지 명시 반출", "KnowledgePackageExport", "기준 원문 변경은409. 원문과 전달 대상에 동의해야 합니다. 이미 반출된 사본은 후속 권한 변경으로 회수할 수 없습니다.", false},
		{"/knowledge/packages/{id}", "delete", "본인 지식 패키지 삭제", "KnowledgePackageDelete", "현재 보존 정책을 만족하는 본인 기록만 삭제합니다. 다른 사본의 원격 삭제를 보장하지 않습니다.", false},
		{"/documents/{id}/impact", "get", "현재 권한 안의 문서 변경 영향 후보", "", "from=이전 version 필수, depth=1~5. 직접 등록된 역방향 관계를 각 경로의 양쪽 현재 ACL로 탐색. 문서500·단계관계2000·본인패키지100·허용Agent100 한도. 규칙 기반 수치/절차/표현 후보이며 의미·영향을 확정하지 않습니다.", false},
		{"/knowledge/impact-check", "post", "표시 중 영향 목록의 현재 권한·버전 확인", "KnowledgeImpactCheck", "한 결과로 valid만 반환하며 접근이 사라진 자료의 존재·이유를 추가 노출하지 않습니다.", false},
		{"/documents/{id}/impact-reviews", "post", "정책에 따라 담당자 영향 검토 요청", "KnowledgeImpactRequest", "관리자가 approval_enabled를 켠 경우만 허용. 현재 source작성·target읽기·직접관계·담당자열람권한 검사. 게시승인과 별개입니다.", false},
		{"/knowledge/impact-reviews/{id}", "put", "현재 원문·대상·검토 CAS로 검토 결과 기록", "KnowledgeImpactDecision", "관리자 검토 비활성시403. 담당자 또는 현재 workspace관리자와 대상 작성 권한 필요. exception_requested는 예외 승인 완료가 아닙니다. 원문/대상 자동 수정·게시 없음.", false},
		{"/documents/{id}/proposals", "post", "원문을 유지하고 변경안을 열람자에게 공유", "KnowledgeProposalCreate", "현재 document:read와document:write, 정확한 base_version·공유동의 필요. 사용자당 열린100개. provenance는 작성자의 표기이며 AI 공급자 인증이 아닙니다.", false},
		{"/documents/{id}/proposal-context", "get", "변경안 작성 중 현재 편집 권한·원문·보호 정책 확인", "", "현재 document:read와document:write 모두 필요. 본문 없이 version과 protection_revision만 반환하며 접근이 사라지면404. 미저장 입력을 보관하는 API가 아닙니다.", false},
		{"/knowledge/proposals", "get", "현재 문서 ACL의 변경안 목록", "", "workspace_id 필수, 최근200개 metadata. 숨겨진 원문/암호문 없음.", false},
		{"/knowledge/proposals/{id}", "get", "기준·제안·현재 원문과 변경안 이력 비교", "", "현재 출처 ACL·세션/키·민감정보 정책 재검사. 기준 변경은 stale:true이며 자동 rebase하지 않습니다.", false},
		{"/knowledge/proposals/{id}/merge", "post", "기존 문서 저장 경로로 변경안 원자 반영", "KnowledgeProposalMerge", "현재 기준 문서 버전과 제안 revision CAS. canonical문서·블록/태그·업무 이벤트·변경안 receipt 같은 TX. 게시승인 정책은 유지. 이미 같은 변경안이 반영되었다면 원문을 재작성하지 않고200 receipt.", false},
		{"/knowledge/proposals/{id}/decision", "post", "내 변경안 철회 또는 정책상 검토 반려", "KnowledgeProposalDecision", "withdrawn은작성자, rejected는관리자승인기능 활성+현재문서작성자. 처리 의견 현재보호정책/암호화이력 적용.", false},
		{"/documents/{id}/validity", "put", "문서별 업무 유효기간 전체 CAS 등록", "KnowledgeValiditySave", validityNotice + " 정확한 현재 문서 version과 등록 revision(최초0) 필수. 최대100개, 겹침과존재하지않는원문버전거절. 이유암호화/최근50이력 조회.", false},
		{"/documents/{id}/validity", "get", "현재 권한의 업무 유효기간·등록 이력 조회", "", validityNotice, false},
		{"/documents/{id}/valid-at", "get", "명시 기준일의 당시 원문 조회", "", validityNotice + " date필수, revision을주면 등록변경시409. 현재모든문서ACL/키범위/민감정보정책검사. 원문 hash는내용동일성비교용이며서명이아닙니다.", false},
		{"/knowledge/time-search", "get", "업무 유효일 기준의 과거 원문 검색", "", validityNotice + " workspace_id와date필수,q최대500바이트,limit1~50,after=직전next_after. 현재조직용어사전/NFKC/공백정규화,고급OR/제외문법400. 문서ID순실시간페이지이며고정DB스냅샷아님. 전체8초읽기예산/SQL7초제한. 비공개모집단수미노출.", false},
		{"/knowledge/time-check", "post", "표시 중 시점검색 자료를 한 요청으로 재검사", "KnowledgeTimeCheck", "최대52개 현재문서version·유효기간revision·보호정책revision·현재권한을 검사하여 valid만 반환.", false},
	} {
		methods, _ := paths[entry.path].(map[string]any)
		op, _ := methods[entry.method].(map[string]any)
		if op == nil {
			continue
		}
		op["summary"], op["description"] = entry.summary, entry.description
		if entry.personal {
			op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		}
		if entry.schema != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
		}
		responses, _ := op["responses"].(map[string]any)
		if responses == nil {
			responses = map[string]any{}
			op["responses"] = responses
		}
		responses["409"] = map[string]any{"description": "자료·정책·현재 권한·CAS 변경: 최신 상태를 다시 확인하고 명시적으로 재시도"}
		responses["422"] = map[string]any{"description": "현재 보호 정책·근거·예산 등 처리 조건을 만족하지 않음"}
		if entry.path == "/ai/evidence" && entry.method == "post" {
			responses["201"] = map[string]any{"description": "새 근거 기록 생성; 같은 티켓의 기존 기록은200"}
		}
		if entry.path == "/documents/{id}/proposals" && entry.method == "post" || entry.path == "/knowledge/packages" && entry.method == "post" || entry.path == "/documents/{id}/impact-reviews" && entry.method == "post" {
			responses["201"] = map[string]any{"description": "생성된 기록"}
			delete(responses, "200")
		}
	}
	query := func(path, name string, schema any, required bool) {
		if p, ok := paths[path].(map[string]any); ok {
			if op, ok := p["get"].(map[string]any); ok {
				ps, _ := op["parameters"].([]any)
				op["parameters"] = append(ps, map[string]any{"name": name, "in": "query", "required": required, "schema": schema})
			}
		}
	}
	for _, path := range []string{"/ai/evidence", "/knowledge/packages", "/knowledge/packages/context", "/knowledge/proposals", "/knowledge/impact-reviews", "/knowledge/time-search"} {
		query(path, "workspace_id", uuid, true)
	}
	query("/documents/{id}/impact", "from", version, true)
	query("/documents/{id}/impact", "depth", map[string]any{"type": "integer", "minimum": 1, "maximum": 5}, false)
	for _, path := range []string{"/documents/{id}/valid-at", "/knowledge/time-search"} {
		query(path, "date", map[string]any{"type": "string", "format": "date"}, true)
	}
	query("/documents/{id}/valid-at", "revision", version, false)
	query("/knowledge/time-search", "q", text, false)
	query("/knowledge/time-search", "after", uuid, false)
	query("/knowledge/time-search", "limit", map[string]any{"type": "integer", "minimum": 1, "maximum": 50}, false)
	if schema, ok := schemas["DocumentRelation"].(map[string]any); ok {
		if properties, ok := schema["properties"].(map[string]any); ok {
			properties["type"] = choice("reference", "related", "policy", "execution", "data")
		}
	}
}
