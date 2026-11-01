package export

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"

	"school-attendance/internal/db"
	"school-attendance/internal/service"
)

type ExcelExporter struct {
	db *db.DB
}

func NewExcelExporter(database *db.DB) *ExcelExporter {
	return &ExcelExporter{db: database}
}

func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func MonthNameUzbek(m time.Month) string {
	switch m {
	case time.January:
		return "Yanvar"
	case time.February:
		return "Fevral"
	case time.March:
		return "Mart"
	case time.April:
		return "Aprel"
	case time.May:
		return "May"
	case time.June:
		return "Iyun"
	case time.July:
		return "Iyul"
	case time.August:
		return "Avgust"
	case time.September:
		return "Sentyabr"
	case time.October:
		return "Oktyabr"
	case time.November:
		return "Noyabr"
	case time.December:
		return "Dekabr"
	default:
		return ""
	}
}

type ClassStats struct {
	ClassID      int64
	ClassName    string
	TeacherName  string
	TotalStudents int
	Present      int
	Absent       int
	Late         int
	Excused      int
	Percentage   float64
}

// GenerateMonthlyReport creates an .xlsx workbook with "Umumiy" summary and individual sheets per class.
func (e *ExcelExporter) GenerateMonthlyReport(ctx context.Context, schoolID int64, year int, month time.Month) (*bytes.Buffer, error) {
	school, err := e.db.Queries.GetSchoolByID(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("maktab topilmadi: %w", err)
	}

	settings, err := e.db.Queries.GetSettingsBySchoolID(ctx, schoolID)
	var threshold int32 = 3
	if err == nil && settings.AbsenceThreshold > 0 {
		threshold = settings.AbsenceThreshold
	}

	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	daysInMonth := DaysInMonth(year, month)
	endDate := time.Date(year, month, daysInMonth, 0, 0, 0, 0, time.UTC)

	// Fetch all classes
	classes, err := e.db.Queries.ListClassesBySchool(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("sinflarni yuklashda xatolik: %w", err)
	}

	// Fetch all students and monthly attendance
	records, err := e.db.Queries.GetMonthlyAttendanceForSchool(ctx, db.GetMonthlyAttendanceForSchoolParams{
		SchoolID: schoolID,
		Date:     pgtype.Date{Time: startDate, Valid: true},
		Date_2:   pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("davomat ma'lumotlarini yuklashda xatolik: %w", err)
	}

	// Map data: studentID -> day -> status
	type studentInfo struct {
		ID       int64
		FullName string
		ClassID  int64
	}
	classStudentsMap := make(map[int64][]studentInfo)
	studentAttendanceMap := make(map[int64]map[int]string)
	seenStudents := make(map[int64]bool)

	for _, r := range records {
		if !seenStudents[r.StudentID] {
			seenStudents[r.StudentID] = true
			classStudentsMap[r.ClassID] = append(classStudentsMap[r.ClassID], studentInfo{
				ID:       r.StudentID,
				FullName: r.StudentName,
				ClassID:  r.ClassID,
			})
		}

		if r.Date.Valid && r.Status.Valid && r.Status.String != "" {
			day := r.Date.Time.Day()
			if _, exists := studentAttendanceMap[r.StudentID]; !exists {
				studentAttendanceMap[r.StudentID] = make(map[int]string)
			}
			studentAttendanceMap[r.StudentID][day] = r.Status.String
		}
	}

	f := excelize.NewFile()
	defer f.Close()

	// Setup Styles
	thinBorder := []excelize.Border{
		{Type: "left", Color: "CCCCCC", Style: 1},
		{Type: "top", Color: "CCCCCC", Style: 1},
		{Type: "bottom", Color: "CCCCCC", Style: 1},
		{Type: "right", Color: "CCCCCC", Style: 1},
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1E293B"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorder,
	})

	subHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1E293B", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F1F5F9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	cellCenterStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})

	cellLeftStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    thinBorder,
	})

	// Status styles (+: green, -: red, K: yellow, S: blue)
	presentStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "0F5132", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D1E7DD"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})

	absentStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "842029", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F8D7DA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})

	lateStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "664D03", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFF3CD"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})

	excusedStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "055160", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"CFF4FC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})

	// Threshold highlight style (bold red with red border)
	thresholdExceededStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "DC2626", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FEE2E2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "DC2626", Style: 2},
			{Type: "top", Color: "DC2626", Style: 2},
			{Type: "bottom", Color: "DC2626", Style: 2},
			{Type: "right", Color: "DC2626", Style: 2},
		},
	})

	// 1. Create individual class sheets
	var classStatsList []ClassStats
	for _, c := range classes {
		sheetName := c.Name
		// Sanitize sheet name (replace invalid chars)
		sheetName = strings.ReplaceAll(sheetName, "/", "-")
		sheetName = strings.ReplaceAll(sheetName, "\\", "-")
		sheetName = strings.ReplaceAll(sheetName, ":", "-")

		f.NewSheet(sheetName)

		// Title banner
		titleCell := "A1"
		titleText := fmt.Sprintf("%s — %s sinfi davomat jurnali (%s %d)", school.Name, c.Name, MonthNameUzbek(month), year)
		f.SetCellValue(sheetName, titleCell, titleText)
		f.SetCellStyle(sheetName, "A1", "A1", subHeaderStyle)

		// Header row
		headers := []string{"№", "F.I.SH (O'quvchi)"}
		for d := 1; d <= daysInMonth; d++ {
			headers = append(headers, fmt.Sprintf("%d", d))
		}
		headers = append(headers, "Keldi (+)", "Kelmadi (-)", "Kechikdi (K)", "Sababli (S)", "Davomat (%)")

		headerRowNum := 3
		for colIdx, h := range headers {
			cellName, _ := excelize.CoordinatesToCellName(colIdx+1, headerRowNum)
			f.SetCellValue(sheetName, cellName, h)
			f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
		}

		studentsInClass := classStudentsMap[c.ID]
		classStat := ClassStats{
			ClassID:       c.ID,
			ClassName:     c.Name,
			TeacherName:   c.TeacherName.String,
			TotalStudents: len(studentsInClass),
		}

		// Fill student rows
		currentRow := 4
		for idx, st := range studentsInClass {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), idx+1)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), cellCenterStyle)

			nameCell := fmt.Sprintf("B%d", currentRow)
			f.SetCellValue(sheetName, nameCell, st.FullName)

			studentAttendance := studentAttendanceMap[st.ID]
			presCount := 0
			absCount := 0
			lateCount := 0
			excCount := 0

			for d := 1; d <= daysInMonth; d++ {
				colIdx := 2 + d
				cellCoord, _ := excelize.CoordinatesToCellName(colIdx, currentRow)

				stCode := ""
				if studentAttendance != nil {
					status := studentAttendance[d]
					switch status {
					case service.StatusPresent:
						stCode = "+"
						presCount++
						f.SetCellStyle(sheetName, cellCoord, cellCoord, presentStyle)
					case service.StatusAbsent:
						stCode = "-"
						absCount++
						f.SetCellStyle(sheetName, cellCoord, cellCoord, absentStyle)
					case service.StatusLate:
						stCode = "K"
						lateCount++
						f.SetCellStyle(sheetName, cellCoord, cellCoord, lateStyle)
					case service.StatusExcused:
						stCode = "S"
						excCount++
						f.SetCellStyle(sheetName, cellCoord, cellCoord, excusedStyle)
					default:
						f.SetCellStyle(sheetName, cellCoord, cellCoord, cellCenterStyle)
					}
				} else {
					f.SetCellStyle(sheetName, cellCoord, cellCoord, cellCenterStyle)
				}
				f.SetCellValue(sheetName, cellCoord, stCode)
			}

			// Highlights students over absence threshold in red!
			if int32(absCount) >= threshold {
				f.SetCellStyle(sheetName, nameCell, nameCell, thresholdExceededStyle)
				f.SetCellValue(sheetName, nameCell, fmt.Sprintf("⚠️ %s", st.FullName))
			} else {
				f.SetCellStyle(sheetName, nameCell, nameCell, cellLeftStyle)
			}

			// Summary columns
			baseCol := 2 + daysInMonth
			cPres, _ := excelize.CoordinatesToCellName(baseCol+1, currentRow)
			cAbs, _ := excelize.CoordinatesToCellName(baseCol+2, currentRow)
			cLate, _ := excelize.CoordinatesToCellName(baseCol+3, currentRow)
			cExc, _ := excelize.CoordinatesToCellName(baseCol+4, currentRow)
			cPct, _ := excelize.CoordinatesToCellName(baseCol+5, currentRow)

			f.SetCellValue(sheetName, cPres, presCount)
			f.SetCellValue(sheetName, cAbs, absCount)
			f.SetCellValue(sheetName, cLate, lateCount)
			f.SetCellValue(sheetName, cExc, excCount)

			f.SetCellStyle(sheetName, cPres, cPres, cellCenterStyle)
			f.SetCellStyle(sheetName, cAbs, cAbs, cellCenterStyle)
			f.SetCellStyle(sheetName, cLate, cLate, cellCenterStyle)
			f.SetCellStyle(sheetName, cExc, cExc, cellCenterStyle)

			totalMarked := presCount + absCount + lateCount + excCount
			pct := 0.0
			if totalMarked > 0 {
				pct = math.Round((float64(presCount)/float64(totalMarked))*1000) / 10
			}
			f.SetCellValue(sheetName, cPct, fmt.Sprintf("%.1f%%", pct))
			f.SetCellStyle(sheetName, cPct, cPct, cellCenterStyle)

			classStat.Present += presCount
			classStat.Absent += absCount
			classStat.Late += lateCount
			classStat.Excused += excCount

			currentRow++
		}

		totalClassMarked := classStat.Present + classStat.Absent + classStat.Late + classStat.Excused
		if totalClassMarked > 0 {
			classStat.Percentage = math.Round((float64(classStat.Present)/float64(totalClassMarked))*1000) / 10
		}
		classStatsList = append(classStatsList, classStat)

		// Set Frozen Panes: Freeze first 2 columns and header row 3
		topLeftCell, _ := excelize.CoordinatesToCellName(3, 4)
		_ = f.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			Split:       false,
			XSplit:      2,
			YSplit:      3,
			TopLeftCell: topLeftCell,
			ActivePane:  "bottomRight",
		})

		// Column widths
		_ = f.SetColWidth(sheetName, "A", "A", 5)
		_ = f.SetColWidth(sheetName, "B", "B", 28)
		for d := 1; d <= daysInMonth; d++ {
			colName, _ := excelize.ColumnNumberToName(2 + d)
			_ = f.SetColWidth(sheetName, colName, colName, 3.8)
		}
		lastColStart, _ := excelize.ColumnNumberToName(3 + daysInMonth)
		lastColEnd, _ := excelize.ColumnNumberToName(7 + daysInMonth)
		_ = f.SetColWidth(sheetName, lastColStart, lastColEnd, 11)
	}

	// 2. Setup "Umumiy" summary sheet (rename default Sheet1 or create)
	summarySheet := "Umumiy"
	f.SetSheetName("Sheet1", summarySheet)

	f.SetCellValue(summarySheet, "A1", fmt.Sprintf("📊 %s — Umumiy Davomat Hisoboti", school.Name))
	f.SetCellValue(summarySheet, "A2", fmt.Sprintf("Davr: %s %d | Me'yoriy chegara: %d kun sababsiz", MonthNameUzbek(month), year, threshold))
	f.SetCellStyle(summarySheet, "A1", "A1", subHeaderStyle)

	sumHeaders := []string{
		"№", "Sinf nomi", "Sinf rahbari (O'qituvchi)", "O'quvchilar soni",
		"Keldi (+)", "Kelmadi (-)", "Kechikdi (K)", "Sababli (S)", "Davomat foizi (%)",
	}

	for i, h := range sumHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		f.SetCellValue(summarySheet, cell, h)
		f.SetCellStyle(summarySheet, cell, cell, headerStyle)
	}

	rowIdx := 5
	schoolTotalStudents := 0
	schoolPresent := 0
	schoolAbsent := 0
	schoolLate := 0
	schoolExcused := 0

	for i, cs := range classStatsList {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", rowIdx), i+1)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", rowIdx), cs.ClassName)
		f.SetCellValue(summarySheet, fmt.Sprintf("C%d", rowIdx), cs.TeacherName)
		f.SetCellValue(summarySheet, fmt.Sprintf("D%d", rowIdx), cs.TotalStudents)
		f.SetCellValue(summarySheet, fmt.Sprintf("E%d", rowIdx), cs.Present)
		f.SetCellValue(summarySheet, fmt.Sprintf("F%d", rowIdx), cs.Absent)
		f.SetCellValue(summarySheet, fmt.Sprintf("G%d", rowIdx), cs.Late)
		f.SetCellValue(summarySheet, fmt.Sprintf("H%d", rowIdx), cs.Excused)
		f.SetCellValue(summarySheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("%.1f%%", cs.Percentage))

		f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), cellLeftStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), cellCenterStyle)
		f.SetCellStyle(summarySheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), cellCenterStyle)

		schoolTotalStudents += cs.TotalStudents
		schoolPresent += cs.Present
		schoolAbsent += cs.Absent
		schoolLate += cs.Late
		schoolExcused += cs.Excused
		rowIdx++
	}

	// Total Row
	schoolTotalMarked := schoolPresent + schoolAbsent + schoolLate + schoolExcused
	schoolOverallPct := 0.0
	if schoolTotalMarked > 0 {
		schoolOverallPct = math.Round((float64(schoolPresent)/float64(schoolTotalMarked))*1000) / 10
	}

	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", rowIdx), "")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", rowIdx), "JAMI (Maktab bo'yicha):")
	f.SetCellValue(summarySheet, fmt.Sprintf("C%d", rowIdx), "")
	f.SetCellValue(summarySheet, fmt.Sprintf("D%d", rowIdx), schoolTotalStudents)
	f.SetCellValue(summarySheet, fmt.Sprintf("E%d", rowIdx), schoolPresent)
	f.SetCellValue(summarySheet, fmt.Sprintf("F%d", rowIdx), schoolAbsent)
	f.SetCellValue(summarySheet, fmt.Sprintf("G%d", rowIdx), schoolLate)
	f.SetCellValue(summarySheet, fmt.Sprintf("H%d", rowIdx), schoolExcused)
	f.SetCellValue(summarySheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("%.1f%%", schoolOverallPct))

	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "0F172A", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})
	f.SetCellStyle(summarySheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("I%d", rowIdx), totalStyle)

	// Column widths for "Umumiy"
	_ = f.SetColWidth(summarySheet, "A", "A", 5)
	_ = f.SetColWidth(summarySheet, "B", "B", 14)
	_ = f.SetColWidth(summarySheet, "C", "C", 26)
	_ = f.SetColWidth(summarySheet, "D", "D", 16)
	_ = f.SetColWidth(summarySheet, "E", "E", 12)
	_ = f.SetColWidth(summarySheet, "F", "F", 12)
	_ = f.SetColWidth(summarySheet, "G", "G", 12)
	_ = f.SetColWidth(summarySheet, "H", "H", 12)
	_ = f.SetColWidth(summarySheet, "I", "I", 18)

	// Make "Umumiy" the active sheet
	index, _ := f.GetSheetIndex(summarySheet)
	f.SetActiveSheet(index)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("excel faylini yozishda xatolik: %w", err)
	}

	return buf, nil
}

