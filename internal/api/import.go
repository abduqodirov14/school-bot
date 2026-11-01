package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/importer"
)

type ImportHandler struct {
	svc *importer.Service
}

func NewImportHandler(svc *importer.Service) *ImportHandler {
	return &ImportHandler{svc: svc}
}

// GetTemplate handles GET /api/import/template
func (h *ImportHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	data, err := importer.GenerateTemplate()
	if err != nil {
		JSONError(w, "Shablonni yaratishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"Maktab_Oquvchilar_Import_Shablon.xlsx\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// Preview handles POST /api/import/preview (multipart/form-data)
func (h *ImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori import qila oladi", http.StatusForbidden)
		return
	}

	// Limit request size to 10 MB (file itself max 5 MB)
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		JSONError(w, "Faylni yuklashda xatolik: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		JSONError(w, "Excel fayli ('file') tanlanishi shart", http.StatusBadRequest)
		return
	}
	defer file.Close()

	res, err := h.svc.Preview(r.Context(), user.SchoolID.Int64, file, header.Size)
	if err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	JSONResponse(w, res, http.StatusOK)
}

type CommitImportRequest struct {
	CreateMissingClasses bool `json:"create_missing_classes"`
}

// Commit handles POST /api/import/{job_id}/commit
func (h *ImportHandler) Commit(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori importni tasdiqlashi mumkin", http.StatusForbidden)
		return
	}

	jobIDStr := chi.URLParam(r, "job_id")
	var jobUUID pgtype.UUID
	if err := jobUUID.Scan(jobIDStr); err != nil {
		JSONError(w, "Yaroqsiz job identifikatori", http.StatusBadRequest)
		return
	}

	var req CommitImportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.svc.Commit(user.SchoolID.Int64, jobUUID, req.CreateMissingClasses); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Import jarayoni orqa fonda boshlandi",
		"status":  "running",
	}, http.StatusAccepted)
}

// GetStatus handles GET /api/import/{job_id}
func (h *ImportHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Ruxsat etilmagan", http.StatusUnauthorized)
		return
	}

	jobIDStr := chi.URLParam(r, "job_id")
	var jobUUID pgtype.UUID
	if err := jobUUID.Scan(jobIDStr); err != nil {
		JSONError(w, "Yaroqsiz job identifikatori", http.StatusBadRequest)
		return
	}

	job, err := h.svc.GetJob(r.Context(), user.SchoolID.Int64, jobUUID)
	if err != nil {
		JSONError(w, "Import jobi topilmadi: "+err.Error(), http.StatusNotFound)
		return
	}

	JSONResponse(w, job, http.StatusOK)
}

// DownloadErrors handles GET /api/import/{job_id}/errors/download
func (h *ImportHandler) DownloadErrors(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Ruxsat etilmagan", http.StatusUnauthorized)
		return
	}

	jobIDStr := chi.URLParam(r, "job_id")
	var jobUUID pgtype.UUID
	if err := jobUUID.Scan(jobIDStr); err != nil {
		JSONError(w, "Yaroqsiz job identifikatori", http.StatusBadRequest)
		return
	}

	excelData, err := h.svc.GenerateErrorReport(r.Context(), user.SchoolID.Int64, jobUUID)
	if err != nil {
		JSONError(w, "Xatoliklar hisobotini yaratib bo'lmadi: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Import_Xatoliklar_%s.xlsx\"", jobIDStr[:8]))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(excelData)
}

type DeactivateMissingRequest struct {
	StudentIDs []int64 `json:"student_ids"`
}

// DeactivateMissing handles POST /api/import/deactivate-missing
func (h *ImportHandler) DeactivateMissing(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori o'quvchilarni faolsizlantirishi mumkin", http.StatusForbidden)
		return
	}

	var req DeactivateMissingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz so'rov formati", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeactivateMissingStudents(r.Context(), user.SchoolID.Int64, req.StudentIDs); err != nil {
		JSONError(w, "Faolsizlantirishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]string{
		"message": fmt.Sprintf("%d nafar o'quvchi muvaffaqiyatli faolsizlantirildi", len(req.StudentIDs)),
		"count":   strconv.Itoa(len(req.StudentIDs)),
	}, http.StatusOK)
}
