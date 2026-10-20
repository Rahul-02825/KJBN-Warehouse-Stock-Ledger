package httpapi

import (
	"bytes"
	"sync"
)

const logBufferSize = 500

type LogBuffer struct {
	mu    sync.Mutex
	lines [][]byte
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{}
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	line := bytes.Clone(bytes.TrimSpace(p))
	b.mu.Lock()
	b.lines = append(b.lines, line)
	if len(b.lines) > logBufferSize {
		b.lines = b.lines[len(b.lines)-logBufferSize:]
	}
	b.mu.Unlock()
	return len(p), nil
}

func (b *LogBuffer) Lines() [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([][]byte(nil), b.lines...)
}
