package server

func distributionOpenAPI(paths, schemas map[string]any) {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	text := map[string]any{"type": "string"}
	yes := map[string]any{"type": "boolean", "const": true}
	revision := map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483646}
	object := func(fields map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": fields, "required": required}
	}
	schemas["DistributionPolicy"] = object(map[string]any{"revision": revision, "enabled": map[string]any{"type": "boolean"}, "max_valid_days": map[string]any{"type": "integer", "minimum": 1, "maximum": 365}, "consent": yes}, "revision", "enabled", "max_valid_days", "consent")
	schemas["DistributionKey"] = object(map[string]any{"kind": map[string]any{"enum": []string{"signing", "trusted"}}, "label": text, "source_instance": uuid, "source_key_id": uuid, "public_key": text, "fingerprint": text, "consent": yes}, "kind", "label", "consent")
	schemas["DistributionKeyRevoke"] = object(map[string]any{"revision": revision, "confirmation": map[string]any{"const": "REVOKE"}}, "revision", "confirmation")
	schemas["DistributionTrust"] = object(map[string]any{"revision": revision, "fingerprint": text, "confirmation": map[string]any{"const": "TRUST"}}, "revision", "fingerprint", "confirmation")
	schemas["DistributionImport"] = object(map[string]any{"manifest_base64": text, "signature": text, "consent": yes}, "manifest_base64", "signature", "consent")
	schemas["DistributionExport"] = object(map[string]any{"export_id": uuid, "signing_key_id": uuid, "receiver_instance": uuid, "valid_days": map[string]any{"type": "integer", "minimum": 1, "maximum": 365}, "consent": yes, "documents": map[string]any{"type": "array", "minItems": 1, "maxItems": 1000, "items": object(map[string]any{"id": uuid, "version": revision}, "id", "version")}}, "export_id", "signing_key_id", "receiver_instance", "valid_days", "consent", "documents")
	schemas["DistributionRevoke"] = object(map[string]any{"confirmation": map[string]any{"const": "REVOKE"}}, "confirmation")
	schemas["DistributionApprovalSubmit"] = object(map[string]any{"manifest_sha256": map[string]any{"type": "string", "pattern": "^[a-f0-9]{64}$"}, "consent": yes}, "manifest_sha256", "consent")
	schemas["DistributionApprovedSign"] = object(map[string]any{"request_id": uuid, "request_version": map[string]any{"type": "integer", "minimum": 1}, "consent": yes}, "request_id", "request_version", "consent")
	for _, v := range []struct{ path, method, schema, description string }{
		{"/admin/knowledge-distribution", "get", "", "서비스 관리자 전용 정책·공개키·최근100정책이력. 개인키/암호문은 반환하지 않습니다."},
		{"/admin/knowledge-distribution", "put", "DistributionPolicy", "현재revision CAS+명시동의. 기본off,최대1~365일. 변경하면 기존준비반출을새로생성해야합니다. 망instance는이API로변경하지않습니다."},
		{"/admin/knowledge-distribution/keys", "post", "DistributionKey", "서명키는서버Ed25519생성·개인키암호화. trusted는source_instance/source_key_id/32byte공개키base64/별도확인SHA256지문필수. 기존키자동철회/자동신뢰/덮어쓰기없음. 이력포함100키한도."},
		{"/admin/knowledge-distribution/keys/{id}/revoke", "post", "DistributionKeyRevoke", "키철회CAS. 이미전달·반입된사본을회수하지않습니다."},
		{"/admin/knowledge-distribution/keys/{id}/trust", "post", "DistributionTrust", "철회된trusted공개키만별도지문재확인·CAS로재신뢰. signing키재활성불가. 복원후이동작없이신뢰부활하지않습니다."},
		{"/knowledge/distribution/context", "get", "", "workspace_id현재조회권한. 현재망instance/활성서명키label·지문/정책. 공개키신뢰관리는별도관리자페이지입니다."},
		{"/knowledge/distribution/exports", "get", "", "workspace_id필수. 본인최근100건상태/작업ID/수신망/만료. 서명원문·암호문·선택문서참조배열은반환하지않습니다."},
		{"/knowledge/distribution/exports", "post", "DistributionExport", "현재같은로그인에서완료한markdown export +명시확인한문서ID/version으로비동기서명준비(202). 최대파일1000·원본합계50MiB. 현재게시원문·현재승인정책의정확한원문승인·첨부해시·정보보호·서명키·세션을최종TX에서재검사. 수신망별도지정필수."},
		{"/knowledge/distribution/exports/{id}/download", "get", "", "원본내보내기유효기간(최대24시간)과본인원래세션·현재모든원문ACL·원문버전·보호정책·서명키를검사하고서명ZIP반환. 전송중약1초마다다시검사하며이미전송한바이트는회수불가."},
		{"/knowledge/distribution/exports/{id}/revoke", "post", "DistributionRevoke", "본인서버임시사본폐기. 외부사본삭제는수행하지않습니다."},
		{"/knowledge/distribution/exports/{id}/review", "get", "", "현재 모든 원문/첨부 ACL·PII·원신청자 세션·유효기간을 검사하여 검토용 매니페스트와 승인 상태를 표시합니다. 파일 본문은 복제하지 않습니다."},
		{"/knowledge/distribution/exports/{id}/approval", "post", "DistributionApprovalSubmit", "개인 원신청자만 exact manifest SHA256+consent로 전체문서/첨부·수신망·키·유효기간의 명시 정책 승인을 제출합니다. 기본문서승인으로 대체하지 않습니다. 승인설정off면절차제외.201요청ID/version반환."},
		{"/knowledge/distribution/exports/{id}/sign", "post", "DistributionApprovedSign", "원신청자 원래 로그인에서 현재 승인 요청/version+consent를 재검사하고 비동기서명202를 요청합니다. 승인만으로 서명/전송하지 않습니다. 현재승인/원본/정책변경시409. runbook실행권한부여없음."},
		{"/migrations/sessions/{id}/signed-source", "post", "DistributionImport", "빈uploading세션에정확한서명매니페스트(base64,원문≤2MiB)를먼저결합합니다. source_key=signed:<source_instance>:<source_workspace>,format=markdown필수. 현재등록trusted공개키·서명·수신망·기간확인,파일검증완료주장은하지않습니다. 동일재전송멱등/다른원본409. 개별파일1MiB청크업로드및준비/검토/원자확정은기존이관API사용. 확정TX에서모든선언파일·원본메타해시·신뢰·기간재검사+버전매핑같이저장."},
		{"/migrations/sessions/{id}/signed-source", "get", "", "본인이준비한서명반입기록과현재접근가능한대상버전매핑. imported_at은당시반입사실이며현재키신뢰인증이아닙니다. 원본서명은변환된수신망정본의서명이아닙니다."},
	} {
		methods, _ := paths[v.path].(map[string]any)
		op, _ := methods[v.method].(map[string]any)
		if op == nil {
			continue
		}
		op["description"] = v.description
		op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		if v.schema != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + v.schema}}}}
		}
		if v.path == "/knowledge/distribution/exports" && v.method == "post" {
			op["responses"] = map[string]any{"202": map[string]any{"description": "서명작업queued: id/job_id/status"}, "409": map[string]any{"description": "현재원문·정책·범위를다시확인"}}
		}
		if v.path == "/knowledge/distribution/exports/{id}/approval" {
			op["responses"] = map[string]any{"201": map[string]any{"description": "검토 요청 생성"}, "409": map[string]any{"description": "정책/원본/확인 해시 변경"}}
		}
		if v.path == "/knowledge/distribution/exports/{id}/sign" {
			op["responses"] = map[string]any{"202": map[string]any{"description": "승인된 정확한 배포 서명 작업 대기"}, "409": map[string]any{"description": "승인/정책/원본/요청 버전 변경"}}
		}
		if v.path == "/admin/knowledge-distribution/keys" {
			op["responses"] = map[string]any{"201": map[string]any{"description": "공개키정보생성;개인키미노출"}, "409": map[string]any{"description": "기존키덮어쓰기또는한도초과거절"}}
		}
		if v.path == "/knowledge/distribution/exports/{id}/download" {
			op["responses"] = map[string]any{"200": map[string]any{"description": "서명ZIP", "content": map[string]any{"application/zip": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}}}}, "409": map[string]any{"description": "정책·권한·키·원문변경"}, "410": map[string]any{"description": "임시결과만료"}}
		}
		if v.method == "get" && (v.path == "/knowledge/distribution/context" || v.path == "/knowledge/distribution/exports") {
			op["parameters"] = []any{map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": uuid}}
		}
	}
}
