package importer

import (
	"testing"
)

func TestNormalizeClassName(t *testing.T) {
	tests := []struct {
		input       string
		expectedNorm string
	}{
		{"5A", "5A"},
		{"5-a", "5A"},
		{"5 A", "5A"},
		{"5А", "5A"},   // Cyrillic А
		{" 5 - а ", "5A"}, // Cyrillic а with spaces and dash
		{"9-B", "9B"},
		{"9В", "9B"},   // Cyrillic В
		{"10-v", "10V"},
		{"11-Г", "11G"},
		{"1-C", "1C"},  // Cyrillic С
		{"8-H", "8H"},  // Cyrillic Н
		{"7K", "7K"},   // Cyrillic К
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			norm, _ := NormalizeClassName(tt.input)
			if norm != tt.expectedNorm {
				t.Errorf("NormalizeClassName(%q) = %q, expected %q", tt.input, norm, tt.expectedNorm)
			}
		})
	}
}

func TestNormalizeStudentName(t *testing.T) {
	tests := []struct {
		input       string
		expectedDisp string
		expectedKey  string
	}{
		{
			input:       "  aliyev   vali   g'anisher   oʻgʻli  ",
			expectedDisp: "Aliyev Vali G'anisher o'g'li",
			expectedKey:  "aliyev vali g'anisher o'g'li",
		},
		{
			input:       "KARIMOVA  DILNOZA  RUSTAM  QIZI",
			expectedDisp: "Karimova Dilnoza Rustam qizi",
			expectedKey:  "karimova dilnoza rustam qizi",
		},
		{
			input:       "to'xtayev   sardor",
			expectedDisp: "To'xtayev Sardor",
			expectedKey:  "to'xtayev sardor",
		},
		{
			input:       "g‘aniyev  jasur",
			expectedDisp: "G'aniyev Jasur",
			expectedKey:  "g'aniyev jasur",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			disp, key := NormalizeStudentName(tt.input)
			if disp != tt.expectedDisp {
				t.Errorf("NormalizeStudentName(%q) disp = %q, expected %q", tt.input, disp, tt.expectedDisp)
			}
			if key != tt.expectedKey {
				t.Errorf("NormalizeStudentName(%q) key = %q, expected %q", tt.input, key, tt.expectedKey)
			}
		})
	}
}
