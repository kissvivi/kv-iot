package configstore

import (
	"strings"
	"testing"
	"time"
)

func TestMergeJSONConfig(t *testing.T) {
	got, err := mergeJSONConfig(
		`{"area":"holding_register","address":10,"scale":1}`,
		`{"unitId":2,"address":20}`,
	)
	if err != nil {
		t.Fatalf("mergeJSONConfig() error = %v", err)
	}

	text := string(got)
	for _, expected := range []string{
		`"area":"holding_register"`,
		`"address":20`,
		`"unitId":2`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("merged config %s does not contain %s", text, expected)
		}
	}
}

func TestParseRuntimeTiming(t *testing.T) {
	interval, groups, err := parseRuntimeTiming(`{
		"pollIntervalMs": 3000,
		"pollGroups": {
			"fast": 1000,
			"slow": "10s"
		}
	}`)
	if err != nil {
		t.Fatalf("parseRuntimeTiming() error = %v", err)
	}
	if interval != 3*time.Second {
		t.Fatalf("poll interval = %s, want 3s", interval)
	}
	if groups["fast"] != time.Second || groups["slow"] != 10*time.Second {
		t.Fatalf("poll groups = %#v", groups)
	}
}

func TestParseRuntimeTimingRejectsInvalidGroup(t *testing.T) {
	_, _, err := parseRuntimeTiming(`{"pollGroups":{"fast":0}}`)
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("parseRuntimeTiming() error = %v, want positive interval error", err)
	}
}
