package importer

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseExcelFile_SingleSheetWithClassColumn(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "Sinf")
	_ = f.SetCellValue(sheet, "B1", "F.I.Sh")

	_ = f.SetCellValue(sheet, "A2", "5-a") // Cyrillic or mixed
	_ = f.SetCellValue(sheet, "B2", "aliyev vali g'anisher o'g'li")

	_ = f.SetCellValue(sheet, "A3", "5A")
	_ = f.SetCellValue(sheet, "B3", "Karimova Madina")

	_ = f.SetCellValue(sheet, "A4", "5A")
	_ = f.SetCellValue(sheet, "B4", "aliyev vali g'anisher o'g'li") // Duplicate in file!

	_ = f.SetCellValue(sheet, "A5", "6-b")
	_ = f.SetCellValue(sheet, "B5", "S") // Too short!

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("failed to write excel file: %v", err)
	}

	reader := bytes.NewReader(buf.Bytes())
	res, err := ParseExcelFile(reader, int64(buf.Len()))
	if err != nil {
		t.Fatalf("ParseExcelFile failed: %v", err)
	}

	if len(res.ValidRows) != 2 {
		t.Errorf("expected 2 valid rows, got %d", len(res.ValidRows))
	}

	if res.DuplicateCount != 1 {
		t.Errorf("expected 1 duplicate, got %d", res.DuplicateCount)
	}

	if len(res.Errors) != 2 { // 1 duplicate + 1 too short
		t.Errorf("expected 2 errors, got %d", len(res.Errors))
	}

	if res.ValidRows[0].ClassName != "5A" {
		t.Errorf("expected ClassName 5A, got %q", res.ValidRows[0].ClassName)
	}
	if res.ValidRows[0].FullName != "Aliyev Vali G'anisher o'g'li" {
		t.Errorf("expected normalized name 'Aliyev Vali G'anisher o'g'li', got %q", res.ValidRows[0].FullName)
	}
}

func TestParseExcelFile_MultiSheetByClass(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: 5A
	f.SetSheetName("Sheet1", "5A")
	_ = f.SetCellValue("5A", "A1", "№")
	_ = f.SetCellValue("5A", "B1", "O'quvchi F.I.Sh")
	_ = f.SetCellValue("5A", "A2", "1")
	_ = f.SetCellValue("5A", "B2", "Eshmatov Toshmat")

	// Sheet 2: 9-B
	_, _ = f.NewSheet("9-B")
	_ = f.SetCellValue("9-B", "A1", "Ism Familiya")
	_ = f.SetCellValue("9-B", "A2", "Xoliqov Jasur")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("failed to write excel: %v", err)
	}

	reader := bytes.NewReader(buf.Bytes())
	res, err := ParseExcelFile(reader, int64(buf.Len()))
	if err != nil {
		t.Fatalf("ParseExcelFile failed: %v", err)
	}

	if len(res.ValidRows) != 2 {
		t.Fatalf("expected 2 valid rows, got %d", len(res.ValidRows))
	}

	if res.ValidRows[0].ClassName != "5A" || res.ValidRows[1].ClassName != "9B" {
		t.Errorf("expected classes 5A and 9B, got %s and %s", res.ValidRows[0].ClassName, res.ValidRows[1].ClassName)
	}
}

func TestParseExcelFile_SeparateFamiliyaIsm(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "Sinf")
	_ = f.SetCellValue(sheet, "B1", "Familiya")
	_ = f.SetCellValue(sheet, "C1", "Ism")

	_ = f.SetCellValue(sheet, "A2", "7V")
	_ = f.SetCellValue(sheet, "B2", "Rustamov")
	_ = f.SetCellValue(sheet, "C2", "Sarvar")

	var buf bytes.Buffer
	_ = f.Write(&buf)

	reader := bytes.NewReader(buf.Bytes())
	res, err := ParseExcelFile(reader, int64(buf.Len()))
	if err != nil {
		t.Fatalf("ParseExcelFile failed: %v", err)
	}

	if len(res.ValidRows) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(res.ValidRows))
	}

	if res.ValidRows[0].FullName != "Rustamov Sarvar" {
		t.Errorf("expected 'Rustamov Sarvar', got %q", res.ValidRows[0].FullName)
	}
}
