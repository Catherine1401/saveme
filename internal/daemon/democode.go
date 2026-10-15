package daemon

import (
	"strings"

	"saveme/internal/core"
)

// Dùng mã xác nhận cố định cho chế độ demo; mã sai định dạng thì bỏ qua và trả false.
func (e *Engine) UseFixedCode(code string) bool {
	const length, none = 4, ""
	if len(code) != length || strings.Trim(code, core.ConfirmationAlphabet()) != none {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fixedCode = code
	return true
}

// Mã kế tiếp: mã cố định nếu đang ở demo, không thì mã ngẫu nhiên khác mã trước.
func (e *Engine) nextCode(previous string) (string, error) {
	const none = ""
	if e.fixedCode != none {
		return e.fixedCode, nil
	}
	return replacementCode(previous)
}
