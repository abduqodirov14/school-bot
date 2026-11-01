package service

import (
	"testing"
)

func TestAttendanceStatusTransitions(t *testing.T) {
	tests := []struct {
		current  string
		expected string
	}{
		{StatusUnmarked, StatusPresent},
		{StatusPresent, StatusAbsent},
		{StatusAbsent, StatusLate},
		{StatusLate, StatusExcused},
		{StatusExcused, StatusPresent},
		{"unknown", StatusPresent},
		{"", StatusPresent},
	}

	for _, tc := range tests {
		actual := NextStatus(tc.current)
		if actual != tc.expected {
			t.Errorf("NextStatus(%q) = %q; expected %q", tc.current, actual, tc.expected)
		}
	}
}

func TestStatusHelpers(t *testing.T) {
	if StatusLabel(StatusPresent) != "Keldi" {
		t.Errorf("expected 'Keldi', got %s", StatusLabel(StatusPresent))
	}
	if StatusLabel(StatusAbsent) != "Kelmadi" {
		t.Errorf("expected 'Kelmadi', got %s", StatusLabel(StatusAbsent))
	}
	if StatusCode(StatusPresent) != "+" {
		t.Errorf("expected '+', got %s", StatusCode(StatusPresent))
	}
	if StatusCode(StatusAbsent) != "-" {
		t.Errorf("expected '-', got %s", StatusCode(StatusAbsent))
	}
	if StatusCode(StatusLate) != "K" {
		t.Errorf("expected 'K', got %s", StatusCode(StatusLate))
	}
	if StatusCode(StatusExcused) != "S" {
		t.Errorf("expected 'S', got %s", StatusCode(StatusExcused))
	}

	if StatusEmoji(StatusPresent) != "✅" {
		t.Errorf("expected '✅', got %s", StatusEmoji(StatusPresent))
	}
	if StatusEmoji(StatusAbsent) != "❌" {
		t.Errorf("expected '❌', got %s", StatusEmoji(StatusAbsent))
	}
}

