package export

import (
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		year     int
		month    time.Month
		expected int
	}{
		{2026, time.January, 31},
		{2026, time.February, 28},
		{2024, time.February, 29}, // Leap year
		{2026, time.April, 30},
		{2026, time.November, 30},
		{2026, time.December, 31},
	}

	for _, tc := range tests {
		actual := DaysInMonth(tc.year, tc.month)
		if actual != tc.expected {
			t.Errorf("DaysInMonth(%d, %v) = %d; expected %d", tc.year, tc.month, actual, tc.expected)
		}
	}
}

func TestMonthNameUzbek(t *testing.T) {
	if MonthNameUzbek(time.November) != "Noyabr" {
		t.Errorf("expected 'Noyabr', got %s", MonthNameUzbek(time.November))
	}
	if MonthNameUzbek(time.May) != "May" {
		t.Errorf("expected 'May', got %s", MonthNameUzbek(time.May))
	}
}

func TestExcelWorkbookPanesAndSheets(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	summarySheet := "Umumiy"
	f.SetSheetName("Sheet1", summarySheet)

	classSheet := "7-A"
	f.NewSheet(classSheet)

	// Set Frozen Panes on classSheet (freeze top 3 rows, left 2 columns)
	err := f.SetPanes(classSheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      2,
		YSplit:      3,
		TopLeftCell: "C4",
		ActivePane:  "bottomRight",
	})
	if err != nil {
		t.Fatalf("failed to set panes: %v", err)
	}

	sheets := f.GetSheetList()
	if len(sheets) != 2 {
		t.Errorf("expected 2 sheets, got %d", len(sheets))
	}
	if sheets[0] != "Umumiy" || sheets[1] != "7-A" {
		t.Errorf("unexpected sheet names: %v", sheets)
	}

	// Verify buffer write
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to write buffer: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty buffer")
	}
}

