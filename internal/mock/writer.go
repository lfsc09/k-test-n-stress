package mock

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

const defaultBufferSizeBytes uint = 1024 * 1024 // 1 MB

type BufferWriter struct {
	atomicFileRef *AtomicFile
	writer        *bufio.Writer
	BufferSize    uint
}

// NewBufferWriter creates a BufferWriter with a bufio.Writer that writes to both stdout and the atomic file (if set).
// The buffer size is fixed to `defaultBufferSizeBytes`.
// Note: MultiWriter writes sequentially to each writer.
func NewBufferWriter(stdoutWriter io.Writer, atomicFile *AtomicFile) *BufferWriter {
	writers := []io.Writer{}
	if stdoutWriter != nil {
		writers = append(writers, stdoutWriter)
	}
	if atomicFile != nil {
		writers = append(writers, atomicFile.File)
	}

	if len(writers) == 0 {
		return nil
	}

	multiWriter := io.MultiWriter(writers...)
	bufferWriter := bufio.NewWriterSize(multiWriter, int(defaultBufferSizeBytes))

	return &BufferWriter{
		BufferSize:    defaultBufferSizeBytes,
		atomicFileRef: atomicFile,
		writer:        bufferWriter,
	}
}

// Done flushes the buffer and commits the atomic file if set. If flushing fails, it aborts the atomic file and returns the error.
func (mw *BufferWriter) Done() error {
	if err := mw.writer.Flush(); err != nil {
		mw.Abort()
		return err
	}
	if mw.atomicFileRef != nil {
		return mw.atomicFileRef.Commit()
	}
	return nil
}

// Abort aborts the atomic file if set in case of error.
func (mw *BufferWriter) Abort() {
	if mw.atomicFileRef != nil {
		mw.atomicFileRef.Abort()
	}
}

// WriteJSONEncode marshals the given value into JSON and writes it to the buffer.
// This is used for writing JSON values, ensuring proper escaping and formatting.
func (mw *BufferWriter) WriteJSONEncode(value any, stats *DebugStats) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		mw.Abort()
		return err
	}

	byteWritten, err := mw.writer.Write(jsonBytes)
	if err != nil {
		mw.Abort()
		return err
	}

	if stats != nil {
		stats.BytesWritten.Add(uint32(byteWritten))
	}

	return nil
}

// WriteJSONRaw writes the given string to the buffer as is, without JSON encoding.
// This is used for writing JSON structural characters like {, }, [, ], :, and ,.
func (mw *BufferWriter) WriteJSONRaw(jsonStr string, stats *DebugStats) error {
	byteWritten, err := mw.writer.WriteString(jsonStr)
	if err != nil {
		mw.Abort()
		return err
	}

	if stats != nil {
		stats.BytesWritten.Add(uint32(byteWritten))
	}

	return nil
}

// WriteCSV writes the given string to the buffer as a CSV field, applying necessary escaping and quoting (RFC 4180).
func (mw *BufferWriter) WriteCSV(csvStr string, stats *DebugStats) error {
	var byteWritten int
	var err error
	if strings.ContainsAny(csvStr, `,"`+"\r\n") {
		// needs quoting: escape inner quotes and wrap
		quoted := `"` + strings.ReplaceAll(csvStr, `"`, `""`) + `"`
		byteWritten, err = mw.writer.WriteString(quoted)
	} else {
		byteWritten, err = mw.writer.WriteString(csvStr)
	}

	if err != nil {
		mw.Abort()
		return err
	}

	if stats != nil {
		stats.BytesWritten.Add(uint32(byteWritten))
	}

	return nil
}

// WriteCSVRaw writes the given string to the buffer as is, without any encoding.
// This is used for writing CSV content directly, assuming the template provides properly formatted CSV values.
func (mw *BufferWriter) WriteCSVRaw(csvStr string, stats *DebugStats) error {
	byteWritten, err := mw.writer.WriteString(csvStr)
	if err != nil {
		mw.Abort()
		return err
	}

	if stats != nil {
		stats.BytesWritten.Add(uint32(byteWritten))
	}

	return nil
}
