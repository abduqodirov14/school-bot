package api

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Helper to generate a test Excel buffer
func createTestExcel(t *testing.T, rows [][]string) *bytes.Buffer {
	f := excelize.NewFile()
	sheet := "Sheet1"
	for rIdx, row := range rows {
		for cIdx, cell := range row {
			cellName, err := excelize.CoordinatesToCellName(cIdx+1, rIdx+1)
			if err != nil {
				t.Fatalf("error creating cell: %v", err)
			}
			if err := f.SetCellValue(sheet, cellName, cell); err != nil {
				t.Fatalf("error setting cell value: %v", err)
			}
		}
	}
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		t.Fatalf("failed to write excel to buffer: %v", err)
	}
	return buf
}

func TestExcelParsing_WithHeaders(t *testing.T) {
	testData := [][]string{
		{"№", "Sinf", "F.I.SH"},
		{"1", "5-A", "Rahimov Aziz"},
		{"2", "5-A", "Karimova Madina"},
		{"3", "6-B", "Sobirov Dilshod"},
	}

	buf := createTestExcel(t, testData)

	xlsx, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("failed to open excel reader: %v", err)
	}
	defer xlsx.Close()

	sheetList := xlsx.GetSheetList()
	if len(sheetList) == 0 {
		t.Fatal("no sheets found")
	}

	rows, err := xlsx.GetRows(sheetList[0])
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}

	classCol := 0
	nameCol := 1
	startIndex := 0

	firstRow := rows[0]
	foundHeader := false
	for i, cell := range firstRow {
		cellLower := strings.ToLower(strings.TrimSpace(cell))
		if strings.Contains(cellLower, "sinf") {
			classCol = i
			foundHeader = true
		}
		if strings.Contains(cellLower, "f.i.sh") {
			nameCol = i
			foundHeader = true
		}
	}

	if !foundHeader {
		t.Fatal("expected headers to be recognized")
	}
	if classCol != 1 || nameCol != 2 {
		t.Errorf("expected classCol=1 and nameCol=2, got classCol=%d, nameCol=%d", classCol, nameCol)
	}

	startIndex = 1
	var parsedStudents []string
	for i := startIndex; i < len(rows); i++ {
		row := rows[i]
		if len(row) > nameCol && strings.TrimSpace(row[nameCol]) != "" {
			parsedStudents = append(parsedStudents, strings.TrimSpace(row[nameCol]))
		}
	}

	if len(parsedStudents) != 3 {
		t.Errorf("expected 3 students, got %d", len(parsedStudents))
	}
	if parsedStudents[0] != "Rahimov Aziz" {
		t.Errorf("expected first student 'Rahimov Aziz', got '%s'", parsedStudents[0])
	}
}

func TestExcelParsing_WithoutHeaders(t *testing.T) {
	testData := [][]string{
		{"7-V", "Aliyev Vali"},
		{"7-V", "Salimov Akmal"},
	}

	buf := createTestExcel(t, testData)

	xlsx, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("failed to open reader: %v", err)
	}
	defer xlsx.Close()

	rows, err := xlsx.GetRows(xlsx.GetSheetList()[0])
	if err != nil {
		t.Fatalf("failed to get rows: %v", err)
	}

	classCol := 0
	nameCol := 1

	var classes []string
	var students []string
	for _, row := range rows {
		if len(row) > nameCol {
			classes = append(classes, row[classCol])
			students = append(students, row[nameCol])
		}
	}

	if len(students) != 2 {
		t.Fatalf("expected 2 students, got %d", len(students))
	}
	if classes[0] != "7-V" || students[0] != "Aliyev Vali" {
		t.Errorf("unexpected parsing: %s, %s", classes[0], students[0])
	}
}
