package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"school-attendance/internal/config"
	"school-attendance/internal/db"
	"school-attendance/internal/export"
)

type ReportHandler struct {
	db       *db.DB
	cfg      *config.Config
	exporter *export.ExcelExporter
}

func NewReportHandler(database *db.DB, cfg *config.Config) *ReportHandler {
	return &ReportHandler{
		db:       database,
		cfg:      cfg,
		exporter: export.NewExcelExporter(database),
	}
}

func (h *ReportHandler) ExportMonthlyExcel(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	now := time.Now().In(h.cfg.Location)
	year := now.Year()
	month := now.Month()

	monthStr := r.URL.Query().Get("month") // YYYY-MM
	if monthStr != "" {
		parts := strings.Split(monthStr, "-")
		if len(parts) == 2 {
			y, yErr := strconv.Atoi(parts[0])
			m, mErr := strconv.Atoi(parts[1])
			if yErr == nil && mErr == nil && m >= 1 && m <= 12 {
				year = y
				month = time.Month(m)
			}
		}
	}

	buf, err := h.exporter.GenerateMonthlyReport(r.Context(), user.SchoolID.Int64, year, month)
	if err != nil {
		JSONError(w, "Hisobotni yaratishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Davomat_Hisobot_%04d-%02d.xlsx", year, month)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	_, _ = w.Write(buf.Bytes())
}

