package probe

import (
	"strconv"
	"testing"
)

// parseCronField 의 step 분기는 strconv.Atoi(stepRaw) 를 직접 쓴다. 그 Atoi 가 부호를
// 어떻게 다루는지만 여기서 실행으로 확인한다(internal 패키지는 외부 모듈에서 import 할 수 없다).
func TestAtoiAcceptsSignedStep(t *testing.T) {
	for _, raw := range []string{"+2", "+3", "-2", "2", "+0"} {
		value, err := strconv.Atoi(raw)
		t.Logf("Atoi(%q) = %d, err=%v (step 조건: err==nil && 1<=value<=max-min+1)", raw, value, err)
	}
}
