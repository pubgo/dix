package dixinternal

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const diagFileEnv = "DIX_DIAG_FILE"

var (
	diagFileMu   sync.Mutex
	diagFilePath string
	diagFile     *os.File
	diagWarnOnce sync.Map
)

type diagFileRecord struct {
	Kind       string         `json:"kind"`
	OccurredAt int64          `json:"occurred_at_unix_nano"`
	Event      string         `json:"event,omitempty"`
	Fields     map[string]any `json:"fields,omitempty"`
	Payload    any            `json:"payload,omitempty"`
}

func currentDiagFilePath() string {
	return strings.TrimSpace(os.Getenv(diagFileEnv))
}

func ensureDiagFile() *os.File {
	path := currentDiagFilePath()
	if path == "" {
		return nil
	}

	if diagFile != nil && diagFilePath == path {
		return diagFile
	}

	if diagFile != nil {
		if err := diagFile.Close(); err != nil {
			warnDiagOnce("close_previous", err, "diag_file", diagFilePath)
		}
		diagFile = nil
		diagFilePath = ""
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			warnDiagOnce("mkdir_all", err, "diag_file", path, "dir", dir)
			return nil
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		warnDiagOnce("open_file", err, "diag_file", path)
		return nil
	}

	diagFile = f
	diagFilePath = path
	return diagFile
}

func emitDiagFileRecord(record diagFileRecord) {
	if currentDiagFilePath() == "" {
		return
	}

	diagFileMu.Lock()
	defer diagFileMu.Unlock()

	f := ensureDiagFile()
	if f == nil {
		return
	}

	data, err := json.Marshal(record)
	if err != nil {
		warnDiagOnce("marshal", err, "diag_file", currentDiagFilePath(), "kind", record.Kind, "event", record.Event)
		return
	}

	line := append(data, '\n')
	n, err := f.Write(line)
	if err != nil {
		warnDiagOnce("write", err, "diag_file", currentDiagFilePath(), "kind", record.Kind, "event", record.Event)
		return
	}
	if n != len(line) {
		warnDiagOnce("short_write", io.ErrShortWrite, "diag_file", currentDiagFilePath(), "kind", record.Kind, "event", record.Event, "written", n, "expected", len(line))
	}
}

func emitDiagFileTraceEvent(event string, args ...any) {
	record := diagFileRecord{
		Kind:       "trace",
		OccurredAt: time.Now().UnixNano(),
		Event:      event,
		Fields:     kvArgsToMap(args...),
	}
	emitDiagFileRecord(record)
}

func emitDiagFileErrorRecord(record recentErrorRecord) {
	emitDiagFileRecord(diagFileRecord{
		Kind:       "error",
		OccurredAt: record.Occurred.UnixNano(),
		Payload:    record,
	})
}

func emitDiagFileLLMRecord(payload any) {
	emitDiagFileRecord(diagFileRecord{
		Kind:       "llm",
		OccurredAt: time.Now().UnixNano(),
		Payload:    payload,
	})
}

func kvArgsToMap(args ...any) map[string]any {
	if len(args) == 0 {
		return nil
	}

	fields := make(map[string]any, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			fields["_arg"] = args[i]
			break
		}
		key, ok := args[i].(string)
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		fields[key] = args[i+1]
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func resetDiagFileWriterForTest() {
	diagFileMu.Lock()
	defer diagFileMu.Unlock()
	if diagFile != nil {
		if err := diagFile.Close(); err != nil {
			warnDiagOnce("close_reset", err, "diag_file", diagFilePath)
		}
	}
	diagFile = nil
	diagFilePath = ""
	diagWarnOnce = sync.Map{}
}

func warnDiagOnce(kind string, err error, args ...any) {
	if err == nil || logger == nil {
		return
	}

	key := kind + ":" + currentDiagFilePath()
	if _, loaded := diagWarnOnce.LoadOrStore(key, struct{}{}); loaded {
		return
	}

	fields := []any{"kind", kind, "error", err}
	fields = append(fields, args...)
	logger.Warn("diagnostic file write issue", fields...)
}
