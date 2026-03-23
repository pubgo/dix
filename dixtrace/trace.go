package dixtrace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	traceFileEnv = "DIX_TRACE_FILE"
)

// Event 是统一 trace 事件结构。
type Event struct {
	ID               int64          `json:"id"`
	TraceID          string         `json:"trace_id,omitempty"`
	SpanID           string         `json:"span_id,omitempty"`
	ParentSpanID     string         `json:"parent_span_id,omitempty"`
	Operation        string         `json:"operation,omitempty"`
	Phase            string         `json:"phase,omitempty"`
	Event            string         `json:"event,omitempty"`
	Status           string         `json:"status,omitempty"`
	Component        string         `json:"component,omitempty"`
	ProviderFunction string         `json:"provider_function,omitempty"`
	OutputType       string         `json:"output_type,omitempty"`
	InputType        string         `json:"input_type,omitempty"`
	InputTypes       []string       `json:"input_types,omitempty"`
	Message          string         `json:"message,omitempty"`
	Error            string         `json:"error,omitempty"`
	TimedOut         bool           `json:"timed_out,omitempty"`
	DurationNs       int64          `json:"duration_ns,omitempty"`
	OccurredAt       int64          `json:"occurred_at_unix_nano"`
	Attrs            map[string]any `json:"attrs,omitempty"`
}

// Query 控制 trace 查询过滤。
type Query struct {
	TraceID     string
	Operation   string
	Status      string
	Event       string
	Component   string
	Provider    string
	OutputType  string
	Search      string
	Limit       int
	BeforeID    int64
	SinceUnixNs int64
	UntilUnixNs int64
}

// ReadResult 是 trace 查询返回结构。
type ReadResult struct {
	Enabled    bool    `json:"enabled"`
	Total      int     `json:"total"`
	Returned   int     `json:"returned"`
	NextBefore int64   `json:"next_before_id,omitempty"`
	Records    []Event `json:"records"`
}

type Sink interface {
	Write(Event)
}

type spanFrame struct {
	TraceID   string
	SpanID    string
	Operation string
	Component string
}

type spanContextKey struct{}

type Span struct {
	traceID      string
	spanID       string
	parentSpanID string
	operation    string
	component    string
	startedAt    int64
	ended        atomic.Bool
}

func spanFrameFromContext(ctx context.Context) (spanFrame, bool) {
	if ctx == nil {
		return spanFrame{}, false
	}
	v := ctx.Value(spanContextKey{})
	if v == nil {
		return spanFrame{}, false
	}
	f, ok := v.(spanFrame)
	if !ok || f.TraceID == "" || f.SpanID == "" {
		return spanFrame{}, false
	}
	return f, true
}

func resolveCurrentParentFrame(ctx context.Context) (spanFrame, bool) {
	if f, ok := spanFrameFromContext(ctx); ok {
		return f, true
	}
	return spanFrame{}, false
}

func (s *Span) IDs() (traceID, spanID, parentSpanID string) {
	if s == nil {
		return "", "", ""
	}
	return s.traceID, s.spanID, s.parentSpanID
}

func (s *Span) End(err error, args ...any) {
	if s == nil {
		return
	}
	if !s.ended.CompareAndSwap(false, true) {
		return
	}

	attrs := TraceToAttrs(args...)
	duration := time.Now().UnixNano() - s.startedAt
	status := "ok"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	Emit(Event{
		TraceID:      s.traceID,
		SpanID:       s.spanID,
		ParentSpanID: s.parentSpanID,
		Operation:    s.operation,
		Phase:        "end",
		Event:        "span.end",
		Status:       status,
		Component:    s.component,
		Error:        errMsg,
		DurationNs:   duration,
		OccurredAt:   time.Now().UnixNano(),
		Attrs:        attrs,
	})

}

func nextTraceID() string {
	return "t-" + strconv.FormatInt(traceSeq.Add(1), 10)
}

func nextSpanID() string {
	return "s-" + strconv.FormatInt(spanSeq.Add(1), 10)
}

// BeginSpan 开始一个 span。
// 无 context 传递时，该 span 不会自动继承父 span。
func BeginSpan(operation, component string, args ...any) *Span {
	_, span := BeginSpanCtx(context.Background(), operation, component, args...)
	return span
}

// BeginSpanCtx starts a span and returns a context carrying this span as current parent.
// Parent resolution is context-only.
func BeginSpanCtx(ctx context.Context, operation, component string, args ...any) (context.Context, *Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	parent, hasParent := resolveCurrentParentFrame(ctx)

	traceID := ""
	parentSpanID := ""
	if hasParent {
		traceID = parent.TraceID
		parentSpanID = parent.SpanID
	} else {
		traceID = nextTraceID()
	}

	span := &Span{
		traceID:      traceID,
		spanID:       nextSpanID(),
		parentSpanID: parentSpanID,
		operation:    strings.TrimSpace(operation),
		component:    strings.TrimSpace(component),
		startedAt:    time.Now().UnixNano(),
	}

	frame := spanFrame{
		TraceID:   span.traceID,
		SpanID:    span.spanID,
		Operation: span.operation,
		Component: span.component,
	}

	Emit(Event{
		TraceID:      span.traceID,
		SpanID:       span.spanID,
		ParentSpanID: span.parentSpanID,
		Operation:    span.operation,
		Phase:        "start",
		Event:        "span.start",
		Status:       "start",
		Component:    span.component,
		OccurredAt:   span.startedAt,
		Attrs:        TraceToAttrs(args...),
	})

	return context.WithValue(ctx, spanContextKey{}, frame), span
}

type Tracer struct {
	sinks []Sink
	seq   atomic.Int64
}

func NewTracer(sinks ...Sink) *Tracer {
	return &Tracer{sinks: sinks}
}

func (t *Tracer) Emit(e Event) {
	if t == nil {
		return
	}
	if e.OccurredAt <= 0 {
		e.OccurredAt = time.Now().UnixNano()
	}
	if e.ID <= 0 {
		e.ID = t.seq.Add(1)
	}
	for _, s := range t.sinks {
		if s == nil {
			continue
		}
		s.Write(e)
	}
}

// MemorySink 用于 API 查询。
type MemorySink struct {
	mu     sync.RWMutex
	max    int
	events []Event
}

func NewMemorySink(max int) *MemorySink {
	if max <= 0 {
		max = 5000
	}
	return &MemorySink{max: max, events: make([]Event, 0, max)}
}

func (m *MemorySink) Write(e Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	if len(m.events) > m.max {
		m.events = m.events[len(m.events)-m.max:]
	}
}

func (m *MemorySink) Query(q Query) ReadResult {
	m.mu.RLock()
	all := make([]Event, len(m.events))
	copy(all, m.events)
	m.mu.RUnlock()

	matched := make([]Event, 0, len(all))
	for _, rec := range all {
		if !matches(rec, q) {
			continue
		}
		matched = append(matched, rec)
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].OccurredAt == matched[j].OccurredAt {
			return matched[i].ID > matched[j].ID
		}
		return matched[i].OccurredAt > matched[j].OccurredAt
	})

	limit := normalizeLimit(q.Limit)
	result := ReadResult{
		Enabled: true,
		Total:   len(matched),
		Records: []Event{},
	}
	if len(matched) == 0 {
		return result
	}

	start := len(matched) - limit
	if start < 0 {
		start = 0
	}
	result.Records = append(result.Records, matched[start:]...)
	result.Returned = len(result.Records)
	if start > 0 {
		result.NextBefore = result.Records[0].ID
	}
	return result
}

type FileSink struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

func NewFileSink(path string) *FileSink {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	return &FileSink{path: path}
}

func (f *FileSink) ensureFile() *os.File {
	if f == nil || f.path == "" {
		return nil
	}
	if f.f != nil {
		return f.f
	}
	dir := filepath.Dir(f.path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	fd, err := os.OpenFile(f.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil
	}
	f.f = fd
	return f.f
}

func (f *FileSink) Write(e Event) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fd := f.ensureFile()
	if fd == nil {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = fd.Write(append(b, '\n'))
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 200
	}
	if limit > 2000 {
		return 2000
	}
	return limit
}

func parseString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func parseInt64(v any) int64 {
	switch vv := v.(type) {
	case int:
		return int64(vv)
	case int64:
		return vv
	case int32:
		return int64(vv)
	case float64:
		return int64(vv)
	}
	s := parseString(v)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func matches(rec Event, q Query) bool {
	if q.BeforeID > 0 && rec.ID >= q.BeforeID {
		return false
	}
	if q.SinceUnixNs > 0 && rec.OccurredAt < q.SinceUnixNs {
		return false
	}
	if q.UntilUnixNs > 0 && rec.OccurredAt > q.UntilUnixNs {
		return false
	}

	contains := func(src, target string) bool {
		target = strings.TrimSpace(strings.ToLower(target))
		if target == "" {
			return true
		}
		return strings.Contains(strings.ToLower(src), target)
	}

	if !contains(rec.TraceID, q.TraceID) {
		return false
	}
	if !contains(rec.Operation, q.Operation) {
		return false
	}
	if !contains(rec.Status, q.Status) {
		return false
	}
	if !contains(rec.Event, q.Event) {
		return false
	}
	if !contains(rec.Component, q.Component) {
		return false
	}
	if !contains(rec.ProviderFunction, q.Provider) {
		return false
	}
	if !contains(rec.OutputType, q.OutputType) {
		return false
	}

	search := strings.TrimSpace(strings.ToLower(q.Search))
	if search == "" {
		return true
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), search)
}

var (
	defaultMemorySink = NewMemorySink(5000)
	defaultTracer     = NewTracer(defaultMemorySink)
	traceSeq          atomic.Int64
	spanSeq           atomic.Int64
)

func init() {
	if fs := NewFileSink(os.Getenv(traceFileEnv)); fs != nil {
		defaultTracer.sinks = append(defaultTracer.sinks, fs)
	}
}

// Emit 写入 trace 事件。
func Emit(e Event) {
	defaultTracer.Emit(e)
}

// QueryEvents 查询内存中的 trace 事件。
func QueryEvents(q Query) ReadResult {
	return defaultMemorySink.Query(q)
}

// ParseQueryFromMap 从 query 参数 map 解析过滤条件。
func ParseQueryFromMap(values map[string]any) Query {
	return Query{
		TraceID:     parseString(values["trace_id"]),
		Operation:   parseString(values["operation"]),
		Status:      parseString(values["status"]),
		Event:       parseString(values["event"]),
		Component:   parseString(values["component"]),
		Provider:    parseString(values["provider"]),
		OutputType:  parseString(values["output_type"]),
		Search:      parseString(values["q"]),
		Limit:       int(parseInt64(values["limit"])),
		BeforeID:    parseInt64(values["before_id"]),
		SinceUnixNs: parseInt64(values["since_unix_nano"]),
		UntilUnixNs: parseInt64(values["until_unix_nano"]),
	}
}

// TraceEnabled 表示 trace 查询是否可用。
func TraceEnabled() bool {
	return defaultTracer != nil && defaultMemorySink != nil
}

// TraceToAttrs 将 kv 参数转换为 attrs。
func TraceToAttrs(args ...any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	attrs := make(map[string]any, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			attrs["_arg"] = args[i]
			break
		}
		key, ok := args[i].(string)
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		attrs[key] = args[i+1]
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func resetDefaultForTest() {
	defaultMemorySink.mu.Lock()
	defaultMemorySink.events = defaultMemorySink.events[:0]
	defaultMemorySink.mu.Unlock()
	defaultTracer.seq.Store(0)
	traceSeq.Store(0)
	spanSeq.Store(0)
}

// ResetForTest clears in-memory trace state. Intended for tests only.
func ResetForTest() {
	resetDefaultForTest()
}
