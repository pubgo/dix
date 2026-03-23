package dixinternal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const diagFileEnv = "DIX_DIAG_FILE"

var (
	diagFileMu   sync.Mutex
	diagFilePath string
	diagFile     *os.File
	diagWarnOnce sync.Map
	diagRecordID atomic.Int64
)

type diagFileRecord struct {
	RecordID    int64          `json:"record_id,omitempty"`
	Source      string         `json:"source,omitempty"`
	PID         int            `json:"pid,omitempty"`
	Process     string         `json:"process,omitempty"`
	Hostname    string         `json:"hostname,omitempty"`
	TraceDI     bool           `json:"trace_di,omitempty"`
	LLMDiagMode string         `json:"llm_diag_mode,omitempty"`
	Kind        string         `json:"kind"`
	OccurredAt  int64          `json:"occurred_at_unix_nano"`
	Event       string         `json:"event,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
	Payload     any            `json:"payload,omitempty"`
}

// DiagFileRecord is an exported diagnostic record returned by file-query APIs.
type DiagFileRecord struct {
	RecordID    int64          `json:"record_id,omitempty"`
	Source      string         `json:"source,omitempty"`
	PID         int            `json:"pid,omitempty"`
	Process     string         `json:"process,omitempty"`
	Hostname    string         `json:"hostname,omitempty"`
	TraceDI     bool           `json:"trace_di,omitempty"`
	LLMDiagMode string         `json:"llm_diag_mode,omitempty"`
	Kind        string         `json:"kind"`
	OccurredAt  int64          `json:"occurred_at_unix_nano"`
	Event       string         `json:"event,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
	Payload     any            `json:"payload,omitempty"`
}

// DiagFileQuery controls filtering and pagination for DIX_DIAG_FILE records.
type DiagFileQuery struct {
	Kind      string
	Event     string
	Search    string
	Limit     int
	BeforeID  int64
	SinceUnix int64
	UntilUnix int64
}

// DiagFileReadResult is the API response object for diagnostic file queries.
type DiagFileReadResult struct {
	Enabled    bool             `json:"enabled"`
	Path       string           `json:"path,omitempty"`
	Exists     bool             `json:"exists"`
	Total      int              `json:"total"`
	Returned   int              `json:"returned"`
	NextBefore int64            `json:"next_before_id,omitempty"`
	Records    []DiagFileRecord `json:"records"`
}

func nextDiagRecordID() int64 {
	return diagRecordID.Add(1)
}

func toDiagRecord(r diagFileRecord) DiagFileRecord {
	return DiagFileRecord{
		RecordID:    r.RecordID,
		Source:      r.Source,
		PID:         r.PID,
		Process:     r.Process,
		Hostname:    r.Hostname,
		TraceDI:     r.TraceDI,
		LLMDiagMode: r.LLMDiagMode,
		Kind:        r.Kind,
		OccurredAt:  r.OccurredAt,
		Event:       r.Event,
		Fields:      r.Fields,
		Payload:     r.Payload,
	}
}

func toInternalRecord(r DiagFileRecord) diagFileRecord {
	return diagFileRecord{
		RecordID:    r.RecordID,
		Source:      r.Source,
		PID:         r.PID,
		Process:     r.Process,
		Hostname:    r.Hostname,
		TraceDI:     r.TraceDI,
		LLMDiagMode: r.LLMDiagMode,
		Kind:        r.Kind,
		OccurredAt:  r.OccurredAt,
		Event:       r.Event,
		Fields:      r.Fields,
		Payload:     r.Payload,
	}
}

func buildDiagRecord(kind, event string, fields map[string]any, payload any, occurredAt int64) diagFileRecord {
	if occurredAt <= 0 {
		occurredAt = time.Now().UnixNano()
	}
	hostname, _ := os.Hostname()
	return diagFileRecord{
		RecordID:    nextDiagRecordID(),
		Source:      "dix",
		PID:         os.Getpid(),
		Process:     filepath.Base(os.Args[0]),
		Hostname:    hostname,
		TraceDI:     shouldTraceDependencyFlow(),
		LLMDiagMode: currentLLMDiagMode(),
		Kind:        kind,
		OccurredAt:  occurredAt,
		Event:       event,
		Fields:      fields,
		Payload:     payload,
	}
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
	record := buildDiagRecord("trace", event, kvArgsToMap(args...), nil, time.Now().UnixNano())
	emitDiagFileRecord(record)
}

func emitDiagFileErrorRecord(record recentErrorRecord) {
	emitDiagFileRecord(buildDiagRecord("error", "", nil, record, record.Occurred.UnixNano()))
}

func emitDiagFileLLMRecord(payload any) {
	emitDiagFileRecord(buildDiagRecord("llm", "", nil, payload, time.Now().UnixNano()))
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
	diagRecordID.Store(0)
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

func defaultDiagReadLimit(limit int) int {
	if limit <= 0 {
		return 200
	}
	if limit > 2000 {
		return 2000
	}
	return limit
}

func normalizeDiagRecordID(rec *DiagFileRecord, fallbackLineNo int) {
	if rec.RecordID > 0 {
		return
	}
	if v, ok := rec.Fields["record_id"]; ok {
		switch vv := v.(type) {
		case float64:
			rec.RecordID = int64(vv)
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(vv), 10, 64); err == nil {
				rec.RecordID = parsed
			}
		}
	}
	if rec.RecordID <= 0 {
		rec.RecordID = int64(fallbackLineNo)
	}
}

func diagRecordMatchesQuery(rec DiagFileRecord, q DiagFileQuery) bool {
	if kind := strings.TrimSpace(strings.ToLower(q.Kind)); kind != "" {
		if strings.ToLower(rec.Kind) != kind {
			return false
		}
	}

	if event := strings.TrimSpace(strings.ToLower(q.Event)); event != "" {
		if !strings.Contains(strings.ToLower(rec.Event), event) {
			return false
		}
	}

	if q.BeforeID > 0 && rec.RecordID >= q.BeforeID {
		return false
	}
	if q.SinceUnix > 0 && rec.OccurredAt < q.SinceUnix {
		return false
	}
	if q.UntilUnix > 0 && rec.OccurredAt > q.UntilUnix {
		return false
	}

	search := strings.TrimSpace(strings.ToLower(q.Search))
	if search == "" {
		return true
	}

	if strings.Contains(strings.ToLower(rec.Kind), search) || strings.Contains(strings.ToLower(rec.Event), search) {
		return true
	}

	b, err := json.Marshal(rec)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), search)
}

// ReadDiagFileRecords loads and filters records from DIX_DIAG_FILE.
func ReadDiagFileRecords(query DiagFileQuery) (DiagFileReadResult, error) {
	path := currentDiagFilePath()
	result := DiagFileReadResult{
		Enabled: path != "",
		Path:    path,
		Exists:  false,
		Total:   0,
		Records: []DiagFileRecord{},
	}

	if path == "" {
		return result, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, fmt.Errorf("open diagnostic file: %w", err)
	}
	defer f.Close()

	result.Exists = true
	limit := defaultDiagReadLimit(query.Limit)

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 8*1024*1024)

	lineNo := 0
	all := make([]DiagFileRecord, 0, 256)
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var internal diagFileRecord
		if err := json.Unmarshal([]byte(line), &internal); err != nil {
			continue
		}

		rec := toDiagRecord(internal)
		normalizeDiagRecordID(&rec, lineNo)

		if !diagRecordMatchesQuery(rec, query) {
			continue
		}
		all = append(all, rec)
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scan diagnostic file: %w", err)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].OccurredAt == all[j].OccurredAt {
			return all[i].RecordID > all[j].RecordID
		}
		return all[i].OccurredAt > all[j].OccurredAt
	})

	result.Total = len(all)
	if len(all) == 0 {
		return result, nil
	}

	start := len(all) - limit
	if start < 0 {
		start = 0
	}
	result.Records = append(result.Records, all[start:]...)
	result.Returned = len(result.Records)
	if start > 0 {
		result.NextBefore = result.Records[0].RecordID
	}

	return result, nil
}

// ReadDiagFileRecordsFromLines is a test helper for parsing query behavior without I/O.
func ReadDiagFileRecordsFromLines(lines []string, query DiagFileQuery) DiagFileReadResult {
	result := DiagFileReadResult{Enabled: true, Exists: true, Records: []DiagFileRecord{}}
	limit := defaultDiagReadLimit(query.Limit)
	all := make([]DiagFileRecord, 0, len(lines))

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		var internal diagFileRecord
		if err := json.Unmarshal([]byte(line), &internal); err != nil {
			continue
		}
		rec := toDiagRecord(internal)
		normalizeDiagRecordID(&rec, i+1)
		if diagRecordMatchesQuery(rec, query) {
			all = append(all, rec)
		}
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].OccurredAt == all[j].OccurredAt {
			return all[i].RecordID > all[j].RecordID
		}
		return all[i].OccurredAt > all[j].OccurredAt
	})

	result.Total = len(all)
	start := len(all) - limit
	if start < 0 {
		start = 0
	}
	result.Records = append(result.Records, all[start:]...)
	result.Returned = len(result.Records)
	if start > 0 && len(result.Records) > 0 {
		result.NextBefore = result.Records[0].RecordID
	}
	return result
}
