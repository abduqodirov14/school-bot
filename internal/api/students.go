package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"

	"school-attendance/internal/db"
)

type StudentHandler struct {
	db *db.DB
}

func NewStudentHandler(database *db.DB) *StudentHandler {
	return &StudentHandler{db: database}
}

type CreateStudentRequest struct {
	ClassID  int64  `json:"class_id"`
	FullName string `json:"full_name"`
}

type UpdateStudentRequest struct {
	ClassID  int64  `json:"class_id"`
	FullName string `json:"full_name"`
	Active   bool   `json:"active"`
}

func (h *StudentHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab topilmadi", http.StatusBadRequest)
		return
	}

	classIDStr := r.URL.Query().Get("class_id")
	if classIDStr != "" {
		classID, err := strconv.ParseInt(classIDStr, 10, 64)
		if err != nil {
			JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
			return
		}

		// Teacher validation: can only view students of assigned class
		if user.Role == "teacher" {
			class, err := h.db.Queries.GetClassByID(r.Context(), classID)
			if err != nil || !class.TeacherID.Valid || class.TeacherID.Int64 != user.ID {
				JSONError(w, "Siz faqat o'zingizga biriktirilgan sinf o'quvchilarini ko'ra olasiz", http.StatusForbidden)
				return
			}
		}

		students, err := h.db.Queries.ListStudentsByClass(r.Context(), classID)
		if err != nil {
			JSONError(w, "O'quvchilarni yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
			return
		}
		JSONResponse(w, students, http.StatusOK)
		return
	}

	if user.Role != "director" {
		JSONError(w, "Sinf tanlanishi shart", http.StatusBadRequest)
		return
	}

	// Director: list all students in school
	students, err := h.db.Queries.ListStudentsBySchool(r.Context(), user.SchoolID.Int64)
	if err != nil {
		JSONError(w, "O'quvchilarni yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}
	JSONResponse(w, students, http.StatusOK)
}

func (h *StudentHandler) GetStudent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz o'quvchi identifikatori", http.StatusBadRequest)
		return
	}

	student, err := h.db.Queries.GetStudentByID(r.Context(), id)
	if err != nil {
		JSONError(w, "O'quvchi topilmadi", http.StatusNotFound)
		return
	}

	JSONResponse(w, student, http.StatusOK)
}

func (h *StudentHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	var req CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar", http.StatusBadRequest)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		JSONError(w, "O'quvchi F.I.SH kiritilishi shart", http.StatusBadRequest)
		return
	}
	if req.ClassID <= 0 {
		JSONError(w, "Sinf tanlanishi shart", http.StatusBadRequest)
		return
	}

	created, err := h.db.Queries.CreateStudent(r.Context(), db.CreateStudentParams{
		ClassID:  req.ClassID,
		FullName: req.FullName,
	})
	if err != nil {
		JSONError(w, "O'quvchi qo'shishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, created, http.StatusCreated)
}

func (h *StudentHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz o'quvchi identifikatori", http.StatusBadRequest)
		return
	}

	var req UpdateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar", http.StatusBadRequest)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		JSONError(w, "O'quvchi F.I.SH kiritilishi shart", http.StatusBadRequest)
		return
	}
	if req.ClassID <= 0 {
		JSONError(w, "Sinf tanlanishi shart", http.StatusBadRequest)
		return
	}

	updated, err := h.db.Queries.UpdateStudent(r.Context(), db.UpdateStudentParams{
		ID:       id,
		FullName: req.FullName,
		ClassID:  req.ClassID,
		Active:   req.Active,
	})
	if err != nil {
		JSONError(w, "O'quvchi ma'lumotlarini yangilashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, updated, http.StatusOK)
}

func (h *StudentHandler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz o'quvchi identifikatori", http.StatusBadRequest)
		return
	}

	// Soft delete (deactivate)
	if err := h.db.Queries.DeactivateStudent(r.Context(), id); err != nil {
		JSONError(w, "O'quvchini o'chirishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]string{"message": "O'quvchi muvaffaqiyatli o'chirildi"}, http.StatusOK)
}

type ImportResult struct {
	ImportedStudents int      `json:"imported_students"`
	CreatedClasses   int      `json:"created_classes"`
	Errors           []string `json:"errors,omitempty"`
}

func (h *StudentHandler) ImportStudentsExcel(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab topilmadi", http.StatusBadRequest)
		return
	}

	// 10 MB max upload size
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		JSONError(w, "Fayl hajmi 10 MB dan oshmasligi kerak", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		JSONError(w, "Excel fayli topilmadi ('file' maydoni talab qilinadi)", http.StatusBadRequest)
		return
	}
	defer file.Close()

	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		JSONError(w, "Excel faylini o'qishda xatolik: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer xlsx.Close()

	sheetList := xlsx.GetSheetList()
	if len(sheetList) == 0 {
		JSONError(w, "Excel faylida sahifalar topilmadi", http.StatusBadRequest)
		return
	}

	classCache := make(map[string]int64)
	existingClasses, err := h.db.Queries.ListClassesBySchool(r.Context(), user.SchoolID.Int64)
	if err == nil {
		for _, c := range existingClasses {
			classCache[strings.ToUpper(strings.TrimSpace(c.Name))] = c.ID
		}
	}

	result := ImportResult{
		Errors: make([]string, 0),
	}

	for _, sheetName := range sheetList {
		rows, err := xlsx.GetRows(sheetName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s sahifasini o'qib bo'lmadi", sheetName))
			continue
		}

		if len(rows) == 0 {
			continue
		}

		classCol := 0
		nameCol := 1
		startIndex := 0

		// Check if first row contains headers
		firstRow := rows[0]
		foundHeader := false
		for i, cell := range firstRow {
			cellLower := strings.ToLower(strings.TrimSpace(cell))
			if strings.Contains(cellLower, "sinf") || strings.Contains(cellLower, "class") {
				classCol = i
				foundHeader = true
			}
			if strings.Contains(cellLower, "f.i.sh") || strings.Contains(cellLower, "ism") ||
				strings.Contains(cellLower, "familiya") || strings.Contains(cellLower, "student") ||
				strings.Contains(cellLower, "o'quvchi") || strings.Contains(cellLower, "fio") {
				nameCol = i
				foundHeader = true
			}
		}

		if foundHeader {
			startIndex = 1
		} else if len(firstRow) >= 3 {
			// Check if col 0 is index (e.g. 1, 2, 3), col 1 is class, col 2 is name
			if _, numErr := strconv.Atoi(strings.TrimSpace(firstRow[0])); numErr == nil {
				classCol = 1
				nameCol = 2
			}
		}

		for rowIndex := startIndex; rowIndex < len(rows); rowIndex++ {
			row := rows[rowIndex]
			if len(row) <= classCol || len(row) <= nameCol {
				continue
			}

			className := strings.TrimSpace(row[classCol])
			studentName := strings.TrimSpace(row[nameCol])

			if className == "" || studentName == "" {
				continue
			}

			classKey := strings.ToUpper(className)
			classID, exists := classCache[classKey]
			if !exists {
				// Automatically create class if it doesn't exist
				newClass, createErr := h.db.Queries.CreateClass(r.Context(), db.CreateClassParams{
					SchoolID:  user.SchoolID.Int64,
					TeacherID: pgtype.Int8{Valid: false},
					Name:      className,
				})
				if createErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Qator %d: '%s' sinfini yaratib bo'lmadi: %v", rowIndex+1, className, createErr))
					continue
				}
				classID = newClass.ID
				classCache[classKey] = classID
				result.CreatedClasses++
			}

			// Insert student
			_, createStudentErr := h.db.Queries.CreateStudent(r.Context(), db.CreateStudentParams{
				ClassID:  classID,
				FullName: studentName,
			})
			if createStudentErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Qator %d: '%s' o'quvchisini saqlashda xatolik: %v", rowIndex+1, studentName, createStudentErr))
				continue
			}

			result.ImportedStudents++
		}
	}

	JSONResponse(w, result, http.StatusOK)
}

