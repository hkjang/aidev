# SMTP LOGIN 보안 재현

검토 범위: 로컬 main(c9527e7)...HEAD(79f1889). 해당 코드는 152c6ba에서 추가됐고 고정 base 12ada2f에도 존재한다. UUID 커밋 자체의 회귀가 아니다.

원본 server/internal/mail/mail.go와 message.go를 /tmp/invenqor-review-smtp-7venhcug/에 그대로 복사하고, assets/review-smtp_test.go를 같은 임시 디렉터리의 review_test.go로 두었다. 저장소 파일 변경 없음. 실제 net/smtp와 loopback TCP 연결을 사용하고 smtp.NewClient의 원격 호스트 식별자는 relay.corp.example로 설정했다. EHLO 응답은 STARTTLS 없이 AUTH LOGIN만 광고한다. 실제 사용자 정보 없이 합성 자격증명의 전송 여부만 비교했다.

server 디렉터리에서 실행:
```
go test /tmp/invenqor-review-smtp-7venhcug/mail.go /tmp/invenqor-review-smtp-7venhcug/message.go /tmp/invenqor-review-smtp-7venhcug/review_test.go -run TestReviewAutoMustNotSendLOGINCredentialsWithoutTLS -count=1 -v
```
출력 (exit 1):
```
=== RUN   TestReviewAutoMustNotSendLOGINCredentialsWithoutTLS
    review_test.go:40: auto mode transmitted LOGIN username and password over non-TLS SMTP (session error: <nil>)
--- FAIL: TestReviewAutoMustNotSendLOGINCredentialsWithoutTLS (0.00s)
FAIL
FAIL command-line-arguments 0.006s
FAIL
```
사용한 Go: /home/hkjang/sdk/go1.26.7. 로컬 표준 라이브러리 net/smtp/smtp.go:71,202에서 ServerInfo.Name이 호출자가 제공한 host임을 확인했다. 따라서 mail.go:427의 호스트 비교는 신뢰 검증이 아니며, TLS가 없는 원격 LOGIN을 막지 못한다. 수정은 LOGIN의 TLS 사전조건을 강제하는 것. 인증 없는 사내 릴레이의 메일 발송과 자격증명 전송 조건은 별개다.
