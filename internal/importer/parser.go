package importer

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	MaxFileSize = 5 * 1024 * 1024 // 5 MB
	MaxRows     = 10000
)

type ParsedStudentRow struct {
	SheetName      string `json:"sheet_name"`
	RowNumber      int    `json:"row_number"`
	RawClassName   string `json:"raw_class_name"`
	ClassName      string `json:"class_name"` // Normalized, e.g. "5A"
	RawFullName    string `json:"raw_full_name"`
	FullName       string `json:"full_name"` // Display formatted, e.g. "Aliyev Vali"
	NameNormalized string `json:"name_normalized"`
	Error          string `json:"error,omitempty"`
}

type RowError struct {
	Sheet     string `json:"sheet"`
	RowNumber int    `json:"row_number"`
	Reason    string `json:"reason"`
	RawClass  string `json:"raw_class"`
	RawName   string `json:"raw_name"`
}

type ParseResult struct {
	TotalRows        int                           `json:"total_rows"`
	ValidRows        []ParsedStudentRow            `json:"valid_rows"`
	Errors           []RowError                    `json:"errors"`
	First50Errors    []RowError                    `json:"first_50_errors"`
	ClassesFound     map[string]int                `json:"classes_found"` // className -> count
	DuplicateCount   int                           `json:"duplicate_count"`
}

// ParseExcelFile parses an .xlsx file reader using streaming excelize.Rows().
// It detects single-sheet with class column or multi-sheet layouts, maps headers,
// normalizes names, and validates rows.
func ParseExcelFile(r io.Reader, size int64) (*ParseResult, error) {
	if size > MaxFileSize {
		return nil, fmt.Errorf("Fayl hajmi 5 MB dan oshmasligi kerak (joriy hajm: %.2f MB)", float64(size)/(1024*1024))
	}

	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("Faqat .xlsx (Excel) formatidagi fayllar qabul qilinadi: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel faylida biron bir sahifa (varaq) topilmadi")
	}

	result := &ParseResult{
		ValidRows:    make([]ParsedStudentRow, 0),
		Errors:       make([]RowError, 0),
		ClassesFound: make(map[string]int),
	}

	seenInFile := make(map[string]bool) // "CLASS_NORM#NAME_NORM" -> true
	totalRowsCount := 0

	for _, sheetName := range sheets {
		// Ignore instruction sheets
		normSheetLower := strings.ToLower(strings.TrimSpace(sheetName))
		if normSheetLower == "yo'riqnoma" || normSheetLower == "yoriqnoma" || normSheetLower == "instruction" || normSheetLower == "instructions" {
			continue
		}

		rows, err := f.Rows(sheetName)
		if err != nil {
			continue
		}

		// Detect header in first 10 rows
		rowIdx := 0
		headerFound := false
		classColIdx := -1
		nameColIdx := -1
		familiyaColIdx := -1
		ismColIdx := -1

		// Check if sheet name itself represents a class (e.g. "5A", "5-B")
		sheetClassNorm, _ := NormalizeClassName(sheetName)
		sheetIsClass := isLikelyClassName(sheetClassNorm)

		for rows.Next() {
			rowIdx++
			totalRowsCount++
			if totalRowsCount > MaxRows {
				return nil, fmt.Errorf("Faylda qatorlar soni 10 000 tadan oshib ketdi. Iltimos, kichikroq fayl yuklang.")
			}

			cols, err := rows.Columns()
			if err != nil {
				continue
			}

			// Clean cols
			var cleanCols []string
			for _, c := range cols {
				cleanCols = append(cleanCols, strings.TrimSpace(c))
			}

			// Empty row check
			isEmpty := true
			for _, c := range cleanCols {
				if c != "" {
					isEmpty = false
					break
				}
			}
			if isEmpty {
				continue
			}

			if !headerFound && rowIdx <= 10 {
				cIdx, nIdx, fIdx, iIdx := detectHeaderColumns(cleanCols)
				if nIdx != -1 || (fIdx != -1 && iIdx != -1) {
					headerFound = true
					classColIdx = cIdx
					nameColIdx = nIdx
					familiyaColIdx = fIdx
					ismColIdx = iIdx
					continue // Header row itself is not a data row
				}

				// If sheet name is a class, we might have standard data directly without header
				// or row 1 was not a recognized header. If rowIdx == 10 and still not found, we proceed
				if rowIdx == 10 && sheetIsClass {
					// Fallback: assume column 0 is student name if it looks like a name
					nameColIdx = 0
					headerFound = true
				}
			}

			// If header not found yet and still in first 10 rows, keep searching
			if !headerFound {
				continue
			}

			// Extract Class and Name
			var rawClass, rawName string

			// 1. Determine Class
			if classColIdx != -1 && classColIdx < len(cleanCols) {
				rawClass = cleanCols[classColIdx]
			}
			if rawClass == "" && sheetIsClass {
				rawClass = sheetClassNorm
			}

			// 2. Determine Student Full Name
			if nameColIdx != -1 && nameColIdx < len(cleanCols) {
				rawName = cleanCols[nameColIdx]
			} else if familiyaColIdx != -1 && ismColIdx != -1 {
				fam := ""
				ism := ""
				if familiyaColIdx < len(cleanCols) {
					fam = cleanCols[familiyaColIdx]
				}
				if ismColIdx < len(cleanCols) {
					ism = cleanCols[ismColIdx]
				}
				rawName = strings.TrimSpace(fam + " " + ism)
			}

			// Row validation
			normClass, _ := NormalizeClassName(rawClass)
			displayFullName, normFullName := NormalizeStudentName(rawName)

			if normClass == "" {
				errItem := RowError{
					Sheet:     sheetName,
					RowNumber: rowIdx,
					Reason:    "Sinf nomi ko'rsatilmagan",
					RawClass:  rawClass,
					RawName:   rawName,
				}
				result.Errors = append(result.Errors, errItem)
				continue
			}

			if displayFullName == "" {
				errItem := RowError{
					Sheet:     sheetName,
					RowNumber: rowIdx,
					Reason:    "O'quvchi F.I.Sh bo'sh qoldirilgan",
					RawClass:  rawClass,
					RawName:   rawName,
				}
				result.Errors = append(result.Errors, errItem)
				continue
			}

			// Name length check (at least 3 characters)
			if len([]rune(normFullName)) < 3 {
				errItem := RowError{
					Sheet:     sheetName,
					RowNumber: rowIdx,
					Reason:    "O'quvchi F.I.Sh juda qisqa (kamida 3 ta harf)",
					RawClass:  rawClass,
					RawName:   rawName,
				}
				result.Errors = append(result.Errors, errItem)
				continue
			}

			// Check for duplicates inside the file
			fileKey := normClass + "#" + normFullName
			if seenInFile[fileKey] {
				result.DuplicateCount++
				errItem := RowError{
					Sheet:     sheetName,
					RowNumber: rowIdx,
					Reason:    fmt.Sprintf("Fayl ichida takrorlangan o'quvchi (%s sinfida '%s')", normClass, displayFullName),
					RawClass:  rawClass,
					RawName:   rawName,
				}
				result.Errors = append(result.Errors, errItem)
				continue
			}
			seenInFile[fileKey] = true

			// Valid row!
			result.ValidRows = append(result.ValidRows, ParsedStudentRow{
				SheetName:      sheetName,
				RowNumber:      rowIdx,
				RawClassName:   rawClass,
				ClassName:      normClass,
				RawFullName:    rawName,
				FullName:       displayFullName,
				NameNormalized: normFullName,
			})
			result.ClassesFound[normClass]++
		}
		_ = rows.Close()
	}

	result.TotalRows = len(result.ValidRows) + len(result.Errors)

	// First 50 errors for quick preview
	if len(result.Errors) > 50 {
		result.First50Errors = result.Errors[:50]
	} else {
		result.First50Errors = result.Errors
	}

	return result, nil
}

func detectHeaderColumns(cols []string) (classCol, nameCol, familiyaCol, ismCol int) {
	classCol = -1
	nameCol = -1
	familiyaCol = -1
	ismCol = -1

	classSynonyms := []string{"sinf", "class", "sinfi", "класс"}
	nameSynonyms := []string{
		"f.i.sh", "fish", "f.i.o", "fio", "ism familiya", "ism-familiya",
		"full name", "name", "o'quvchi", "o'quvchi f.i.sh", "talaba",
		"familiya ism sharif", "ф.и.ш", "фио", "ф.и.о",
	}
	familiyaSynonyms := []string{"familiya", "last name", "surname", "фамилия"}
	ismSynonyms := []string{"ism", "first name", "имя"}

	for i, col := range cols {
		lower := strings.ToLower(strings.TrimSpace(col))
		if lower == "" {
			continue
		}

		for _, syn := range classSynonyms {
			if lower == syn || strings.Contains(lower, syn) {
				classCol = i
				break
			}
		}

		for _, syn := range nameSynonyms {
			if lower == syn || strings.Contains(lower, syn) {
				nameCol = i
				break
			}
		}

		for _, syn := range familiyaSynonyms {
			if lower == syn {
				familiyaCol = i
				break
			}
		}

		for _, syn := range ismSynonyms {
			if lower == syn {
				ismCol = i
				break
			}
		}
	}

	return classCol, nameCol, familiyaCol, ismCol
}

func isLikelyClassName(s string) bool {
	if len(s) < 2 || len(s) > 6 {
		return false
	}
	hasDigit := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			hasDigit = true
			break
		}
	}
	return hasDigit
}
