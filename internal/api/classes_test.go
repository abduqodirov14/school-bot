package api

import (
	"strings"
	"testing"

	"school-attendance/internal/importer"
)

func TestBulkClassNamesGeneration(t *testing.T) {
	startGrade := 5
	endGrade := 11
	parallels := []string{"A", "B", "V", "G"}

	var generated []string
	for g := startGrade; g <= endGrade; g++ {
		for _, p := range parallels {
			p = strings.ToUpper(strings.TrimSpace(p))
			if p == "" {
				continue
			}
			rawName := string(rune('0'+g))
			if g >= 10 {
				rawName = "10"
				if g == 11 {
					rawName = "11"
				}
			}
			rawName = rawName + p
			norm, disp := importer.NormalizeClassName(rawName)
			if norm == "" {
				norm = rawName
				disp = rawName
			}
			generated = append(generated, disp)
		}
	}

	expectedCount := 7 * 4 // 5,6,7,8,9,10,11 = 7 grades * 4 parallels = 28 classes
	if len(generated) != expectedCount {
		t.Fatalf("expected %d classes generated, got %d", expectedCount, len(generated))
	}

	if generated[0] != "5-A" {
		t.Errorf("expected 5-A, got %s", generated[0])
	}
	if generated[len(generated)-1] != "11-G" {
		t.Errorf("expected 11-G, got %s", generated[len(generated)-1])
	}
}

func TestClassNormalizationEquivalence(t *testing.T) {
	// Requirements: "5-а", "5 A", "5А" all become "5A"
	variants := []string{"5-а", "5 A", "5А", "5a", "5-A"}
	for _, v := range variants {
		norm, _ := importer.NormalizeClassName(v)
		if norm != "5A" {
			t.Errorf("variant %q normalized to %q, expected '5A'", v, norm)
		}
	}
}
