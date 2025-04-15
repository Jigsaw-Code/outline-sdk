package smart

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// CancellableLogWriter is a log writer that can be cancelled.
type CancellableLogWriter struct {
	Writer io.Writer
	ctx    context.Context
	logMu  sync.Mutex
}

func NewCancellableLogWriter(ctx context.Context, writer io.Writer) *CancellableLogWriter {
	return &CancellableLogWriter{Writer: writer, ctx: ctx}
}

// Only log if context is not done
func (f *CancellableLogWriter) logCtx(ctx context.Context, format string, a ...any) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	f.log(format, a...)
}

func (f *CancellableLogWriter) log(format string, a ...any) {
	if f.Writer != nil {
		f.logMu.Lock()
		defer f.logMu.Unlock()
		fmt.Fprintf(f.Writer, format, a...)
	}
}

// Flush ensures that all logs have been written to the underlying writer.
func (f *CancellableLogWriter) Flush() error {
	if f.Writer == nil {
		return nil
	}
	if flusher, ok := f.Writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}
