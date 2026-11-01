package scheduler

import (
	"testing"
)

func TestAddMinutesToTimeStr(t *testing.T) {
	tests := []struct {
		input    string
		minutes  int
		expected string
	}{
		{"09:00", 30, "09:30"},
		{"08:45", 30, "09:15"},
		{"00:00", 30, "00:30"},
		{"23:45", 30, "00:15"},
		{"12:15", 45, "13:00"},
		{"invalid", 30, "09:30"},
	}

	for _, tc := range tests {
		actual := AddMinutesToTimeStr(tc.input, tc.minutes)
		if actual != tc.expected {
			t.Errorf("AddMinutesToTimeStr(%q, %d) = %q; expected %q", tc.input, tc.minutes, actual, tc.expected)
		}
	}
}

func TestThresholdLogic(t *testing.T) {
	tests := []struct {
		name         string
		absentCount  int64
		threshold    int32
		shouldNotify bool
	}{
		{"Below threshold (0 absences)", 0, 3, false},
		{"Below threshold (2 absences)", 2, 3, false},
		{"Exactly at threshold (3 absences)", 3, 3, true},
		{"Above threshold (4 absences)", 4, 3, true},
		{"Custom threshold (5 absences, threshold 5)", 5, 5, true},
		{"Custom threshold (4 absences, threshold 5)", 4, 5, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isExceeded := tc.absentCount >= int64(tc.threshold)
			if isExceeded != tc.shouldNotify {
				t.Errorf("expected shouldNotify=%v, got %v", tc.shouldNotify, isExceeded)
			}
		})
	}
}

func TestOncePerMonthAlertSimulation(t *testing.T) {
	notifiedMap := make(map[string]bool)

	checkAndNotify := func(studentID int64, yearMonth string) bool {
		key := string(rune(studentID)) + ":" + yearMonth
		if notifiedMap[key] {
			return false // Already notified this month
		}
		notifiedMap[key] = true
		return true // Notify
	}

	studentID := int64(42)
	monthNovember := "2026-11"
	monthDecember := "2026-12"

	// 1st time in November: should trigger notification
	if !checkAndNotify(studentID, monthNovember) {
		t.Error("expected 1st alert in November to trigger")
	}

	// 2nd time in November: should NOT trigger notification (only once per month)
	if checkAndNotify(studentID, monthNovember) {
		t.Error("expected 2nd alert in November to be blocked")
	}

	// 3rd time in November: still blocked
	if checkAndNotify(studentID, monthNovember) {
		t.Error("expected 3rd alert in November to be blocked")
	}

	// Next month (December): should trigger once again for new month
	if !checkAndNotify(studentID, monthDecember) {
		t.Error("expected alert in new month December to trigger")
	}
}

