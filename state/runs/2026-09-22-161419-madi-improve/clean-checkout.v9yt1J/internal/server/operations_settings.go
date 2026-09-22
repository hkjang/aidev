package server

import (
	"crypto/x509"
	"errors"
	"math"
	"net/url"
	"strings"
)

func defaultOperationsSettings() map[string]any {
	return map[string]any{
		"feature_flags": map[string]any{}, "otel_enabled": false, "otel_endpoint": "", "otel_allow_http": false,
		"otel_ca_pem": "", "otel_auth_token": "", "otel_sample_rate": float64(0.1), "otel_timeout_seconds": 5,
		"operations_errors_enabled": true, "operations_retention_days": 7,
	}
}
func operationNumber(value any) (float64, bool) {
	switch n := value.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, !math.IsNaN(n) && !math.IsInf(n, 0)
	}
	return 0, false
}
func validateOperationsSettings(cfg map[string]any) error {
	if err := validateFeatureFlags(cfg["feature_flags"]); err != nil {
		return err
	}
	for _, key := range []string{"otel_enabled", "otel_allow_http", "operations_errors_enabled"} {
		if _, ok := cfg[key].(bool); !ok {
			return errors.New("운영 기능 활성 여부는 true 또는 false로 설정하세요")
		}
	}
	for _, key := range []string{"otel_endpoint", "otel_ca_pem", "otel_auth_token"} {
		if _, ok := cfg[key].(string); !ok {
			return errors.New("운영 연결 설정은 문자열이어야 합니다")
		}
	}
	rate, ok := operationNumber(cfg["otel_sample_rate"])
	if !ok || rate < 0 || rate > 1 {
		return errors.New("Trace 수집 비율은 0~1 범위의 숫자여야 합니다")
	}
	timeout, ok := operationNumber(cfg["otel_timeout_seconds"])
	if !ok || timeout != math.Trunc(timeout) || timeout < 1 || timeout > 30 {
		return errors.New("OTLP 요청 제한 시간은 1~30초 정수여야 합니다")
	}
	days, ok := operationNumber(cfg["operations_retention_days"])
	if !ok || days != math.Trunc(days) || days < 1 || days > 90 {
		return errors.New("운영 오류 보존 기간은 1~90일 정수여야 합니다")
	}
	endpoint := str(cfg, "otel_endpoint")
	if len(endpoint) > 2048 {
		return errors.New("OTLP 수신 주소가 너무 깁니다")
	}
	if endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !oneOf(u.Scheme, "http", "https") {
			return errors.New("OTLP 수신 주소는 인증정보·쿼리·fragment 없는 HTTP(S) 주소여야 합니다")
		}
		if u.Scheme == "http" && !boolean(cfg, "otel_allow_http") {
			return errors.New("HTTP OTLP 전송에는 평문 전송 허용을 명시적으로 켜야 합니다")
		}
	}
	if boolean(cfg, "otel_enabled") && endpoint == "" {
		return errors.New("OpenTelemetry를 켜려면 OTLP 수신 주소를 설정하세요")
	}
	ca := str(cfg, "otel_ca_pem")
	if len(ca) > 65536 {
		return errors.New("OTLP CA 인증서는 64KB까지 지원합니다")
	}
	if ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return errors.New("올바른 PEM CA 인증서를 입력하세요")
		}
	}
	secret := str(cfg, "otel_auth_token")
	if len(secret) > 8192 || strings.ContainsAny(secret, "\r\n\x00") {
		return errors.New("OTLP 인증 토큰 형식과 길이를 확인하세요")
	}
	return nil
}
