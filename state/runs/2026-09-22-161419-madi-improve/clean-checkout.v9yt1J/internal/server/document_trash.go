package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Older REST/MCP clients may omit the version. The browser always supplies it,
// so an Undo cannot restore a different deletion or overwrite a later revision.
func documentTrashExpectedVersion(r *http.Request) (*int, error) {
	var input map[string]json.RawMessage
	if e := decode(r, &input); e != nil && e != io.EOF {
		return nil, errors.New("expected_version은 양의 정수여야 합니다")
	}
	raw, supplied := input["expected_version"]
	if !supplied {
		return nil, nil
	}
	var expected int
	if json.Unmarshal(raw, &expected) != nil || expected < 1 || expected > 2147483647 {
		return nil, errors.New("expected_version은 양의 정수여야 합니다")
	}
	return &expected, nil
}
