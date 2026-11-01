package importer

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// GenerateTemplate creates an Excel template for importing students.
// It includes an "O'quvchilar" sheet with sample data and a "Yo'riqnoma" sheet with instructions.
func GenerateTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetStudents := "O'quvchilar"
	sheetInstructions := "Yo'riqnoma"

	// Rename default sheet to O'quvchilar
	f.SetSheetName("Sheet1", sheetStudents)

	// Create Header Style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#059669"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	// 1. Setup O'quvchilar sheet
	_ = f.SetCellValue(sheetStudents, "A1", "Sinf")
	_ = f.SetCellValue(sheetStudents, "B1", "F.I.Sh")
	_ = f.SetCellStyle(sheetStudents, "A1", "B1", headerStyle)

	// Sample Rows
	sampleRows := [][]string{
		{"5A", "Aliyev Vali G'anisher o'g'li"},
		{"5A", "Karimova Madina Rustam qizi"},
		{"6B", "Sobirov Jasur Otabek o'g'li"},
		{"7V", "Nazarova Zarina Shavkat qizi"},
	}

	for i, row := range sampleRows {
		rowNum := i + 2
		_ = f.SetCellValue(sheetStudents, fmt.Sprintf("A%d", rowNum), row[0])
		_ = f.SetCellValue(sheetStudents, fmt.Sprintf("B%d", rowNum), row[1])
	}

	_ = f.SetColWidth(sheetStudents, "A", "A", 15)
	_ = f.SetColWidth(sheetStudents, "B", "B", 40)
	_ = f.SetRowHeight(sheetStudents, 1, 26)

	// 2. Setup Yo'riqnoma sheet
	_, _ = f.NewSheet(sheetInstructions)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#065F46", Size: 14},
	})
	_ = f.SetCellValue(sheetInstructions, "A1", "Maktab Davomati — O'quvchilarni Import Qilish Yo'riqnomasi")
	_ = f.SetCellStyle(sheetInstructions, "A1", "A1", titleStyle)

	instructions := []string{
		"1. Fayl formati faqat .xlsx (Excel) bo'lishi kerak. Maksimal hajm: 5 MB, maksimal qatorlar soni: 10 000 ta.",
		"2. 1-ustunda Sinf nomi (masalan: 5A, 5-A, 9-B, 11-V) ko'rsatiladi.",
		"3. 2-ustunda O'quvchining to'liq F.I.Sh (Familiya Ism Sharif) kiritiladi.",
		"4. Agar har bir sinf alohida varaqda (sheet) bo'lsa, varaq nomi sinf nomi bo'lishi kifoya (masalan: '5A', '6B').",
		"5. Familiya va Ism alohida ustunlarda bo'lsa ham ('Familiya' va 'Ism'), tizim ularni avtomatik birlashtiradi.",
		"6. Harflar kirill yoki lotinda bo'lishidan qat'i nazar (5-а, 5 А, 5A), tizim bir xil '5A' standartiga o'tkazadi.",
		"7. Bir xil sinfda bir xil F.I.Sh takror kelsa, faqat bittasi qo'shiladi va dublikat deb belgilanadi.",
	}

	for i, inst := range instructions {
		rowNum := i + 3
		_ = f.SetCellValue(sheetInstructions, fmt.Sprintf("A%d", rowNum), inst)
	}

	_ = f.SetColWidth(sheetInstructions, "A", "A", 85)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write excel file to buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateErrorReport creates an Excel file containing all row-level import errors.
func GenerateErrorReport(errors []RowError) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Xatolar ro'yxati"
	f.SetSheetName("Sheet1", sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#DC2626"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	_ = f.SetCellValue(sheetName, "A1", "Varaq (Sheet)")
	_ = f.SetCellValue(sheetName, "B1", "Qator raqami")
	_ = f.SetCellValue(sheetName, "C1", "Kiritilgan sinf")
	_ = f.SetCellValue(sheetName, "D1", "Kiritilgan F.I.Sh")
	_ = f.SetCellValue(sheetName, "E1", "Xatolik sababi")
	_ = f.SetCellStyle(sheetName, "A1", "E1", headerStyle)

	for i, errItem := range errors {
		rowNum := i + 2
		_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), errItem.Sheet)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), errItem.RowNumber)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), errItem.RawClass)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), errItem.RawName)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), errItem.Reason)
	}

	_ = f.SetColWidth(sheetName, "A", "A", 20)
	_ = f.SetColWidth(sheetName, "B", "B", 15)
	_ = f.SetColWidth(sheetName, "C", "C", 20)
	_ = f.SetColWidth(sheetName, "D", "D", 35)
	_ = f.SetColWidth(sheetName, "E", "E", 45)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write error report: %w", err)
	}
	return buf.Bytes(), nil
}
