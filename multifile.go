package log

import (
	"fmt"
	"io"
)

// MultiWriter is an Writer that log to different writers by different levels
type MultiFileWriter struct {
	Writes map[string]Writer
}

// Close implements io.Closer, and closes the underlying LeveledWriter.
func (w *MultiFileWriter) Close() (err error) {
	if w.Writes == nil {
		return nil
	}
	for _, writer := range w.Writes {
		if writer == nil {
			continue
		}
		if closer, ok := writer.(io.Closer); ok {
			if err1 := closer.Close(); err1 != nil {
				err = err1
			}
		}
	}
	return
}

// WriteEntry implements entryWriter.
func (w *MultiFileWriter) WriteEntry(e *Entry) (n int, err error) {
	var err1 error
	loggerFiles := e.loggerFiles
	if loggerFiles == nil || len(loggerFiles) < 1 {
		return 0, nil
	}
	if w.Writes == nil || len(w.Writes) < 1 {
		return
	}
	find := false
	for _, loggerFileName := range loggerFiles {
		if writer, ok := w.Writes[loggerFileName]; ok {
			find = true
			n, err1 = writer.WriteEntry(e)
			if err1 != nil && err == nil {
				err = err1
			}
		}
	}
	if !find {
		if writer, ok := w.Writes["default"]; ok {
			find = true
			n, err1 = writer.WriteEntry(e)
			if err1 != nil && err == nil {
				err = err1
			}
		}
	}
	return
}

// wlprintf is a helper function for tests
func loggerPrintf(w Writer, loggerName string, level Level, format string, args ...interface{}) (int, error) {
	entry := &Entry{
		Level: level,
		buf:   []byte(fmt.Sprintf(format, args...)),
	}
	entry.LoggerFile(loggerName)
	return w.WriteEntry(entry)
}

var _ Writer = (*MultiWriter)(nil)
