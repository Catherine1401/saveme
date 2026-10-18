package update

import (
	"strings"
	"sync"
)

type limitedOutput struct {
	mu   sync.Mutex
	text strings.Builder
}

func (b *limitedOutput) Write(data []byte) (int, error) {
	const (
		literal1 = 0
	)

	const limit = 8 * 1024
	b.mu.Lock()
	defer b.mu.Unlock()
	count := len(data)
	remaining := max(limit-b.text.Len(), literal1)
	if remaining < len(data) {
		data = data[:remaining]
	}
	b.text.Write(data)
	return count, nil
}

func (b *limitedOutput) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.text.String()
}
