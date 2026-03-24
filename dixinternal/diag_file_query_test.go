package dixinternal

import "testing"

func TestReadDiagFileRecordsFromLines_FilterAndLimit(t *testing.T) {
	lines := []string{
		`{"record_id":1,"kind":"trace","event":"inject.start","occurred_at_unix_nano":100,"fields":{"component":"*main.App"}}`,
		`{"record_id":2,"kind":"error","occurred_at_unix_nano":200,"payload":{"message":"missing dependency"}}`,
		`{"record_id":3,"kind":"llm","occurred_at_unix_nano":300,"payload":{"error_type":"inject_dependency_missing"}}`,
		`{"record_id":4,"kind":"trace","event":"provider.call.start","occurred_at_unix_nano":400,"fields":{"provider":"main.NewDB"}}`,
	}

	result := ReadDiagFileRecordsFromLines(lines, DiagFileQuery{
		Kind:  "trace",
		Event: "provider",
		Limit: 10,
	})

	if result.Total != 1 {
		t.Fatalf("expected total 1 after filter, got %d", result.Total)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected one record, got %d", len(result.Records))
	}
	if result.Records[0].RecordID != 4 {
		t.Fatalf("expected record id 4, got %d", result.Records[0].RecordID)
	}
}

func TestReadDiagFileRecordsFromLines_BeforeID(t *testing.T) {
	lines := []string{
		`{"record_id":11,"kind":"trace","event":"a","occurred_at_unix_nano":100}`,
		`{"record_id":12,"kind":"trace","event":"b","occurred_at_unix_nano":200}`,
		`{"record_id":13,"kind":"trace","event":"c","occurred_at_unix_nano":300}`,
	}

	result := ReadDiagFileRecordsFromLines(lines, DiagFileQuery{
		BeforeID: 13,
		Limit:    10,
	})

	if result.Total != 2 {
		t.Fatalf("expected total 2 with before_id filter, got %d", result.Total)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 returned records, got %d", len(result.Records))
	}
	if result.Records[0].RecordID != 12 || result.Records[1].RecordID != 11 {
		t.Fatalf("unexpected returned ids: %+v", result.Records)
	}
}

func TestReadDiagFileRecordsFromLines_TimeDescending(t *testing.T) {
	lines := []string{
		`{"record_id":21,"kind":"error","occurred_at_unix_nano":100}`,
		`{"record_id":22,"kind":"error","occurred_at_unix_nano":300}`,
		`{"record_id":23,"kind":"error","occurred_at_unix_nano":200}`,
	}

	result := ReadDiagFileRecordsFromLines(lines, DiagFileQuery{Kind: "error", Limit: 10})

	if len(result.Records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(result.Records))
	}
	if result.Records[0].RecordID != 22 || result.Records[1].RecordID != 23 || result.Records[2].RecordID != 21 {
		t.Fatalf("expected latest-first order [22,23,21], got %+v", result.Records)
	}
}

func TestReadDiagFileRecordsFromLines_LimitReturnsLatestAndCursor(t *testing.T) {
	lines := []string{
		`{"record_id":31,"kind":"trace","occurred_at_unix_nano":100}`,
		`{"record_id":32,"kind":"trace","occurred_at_unix_nano":200}`,
		`{"record_id":33,"kind":"trace","occurred_at_unix_nano":300}`,
	}

	first := ReadDiagFileRecordsFromLines(lines, DiagFileQuery{Kind: "trace", Limit: 2})
	if first.Total != 3 || first.Returned != 2 {
		t.Fatalf("expected total=3 returned=2, got total=%d returned=%d", first.Total, first.Returned)
	}
	if len(first.Records) != 2 || first.Records[0].RecordID != 33 || first.Records[1].RecordID != 32 {
		t.Fatalf("expected latest-first [33,32], got %+v", first.Records)
	}
	if first.NextBefore != 32 {
		t.Fatalf("expected next_before_id=32, got %d", first.NextBefore)
	}

	second := ReadDiagFileRecordsFromLines(lines, DiagFileQuery{Kind: "trace", Limit: 2, BeforeID: first.NextBefore})
	if second.Total != 1 || second.Returned != 1 {
		t.Fatalf("expected second page total=1 returned=1, got total=%d returned=%d", second.Total, second.Returned)
	}
	if len(second.Records) != 1 || second.Records[0].RecordID != 31 {
		t.Fatalf("expected second page [31], got %+v", second.Records)
	}
}
