//go:build !linux || (!amd64 && !arm64)

package extract

import "errors"

func enterSandbox(directory, program string) error {
	return errors.New("이 호스트는 첨부 추출 격리를 지원하지 않습니다")
}
func replaceProcess(program string, args, env []string) error {
	return errors.New("이 호스트는 첨부 추출 격리를 지원하지 않습니다")
}
