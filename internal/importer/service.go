package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"school-attendance/internal/db"
)

type Service struct {
	database    *db.DB
	schoolLocks sync.Map // schoolID -> *sync.Mutex
}

func NewService(database *db.DB) *Service {
	return &Service{
		database: database,
	}
}

func (s *Service) getSchoolLock(schoolID int64) *sync.Mutex {
	val, _ := s.schoolLocks.LoadOrStore(schoolID, &sync.Mutex{})
	return val.(*sync.Mutex)
}

type MissingClassItem struct {
	ClassName    string `json:"class_name"`
	StudentCount int    `json:"student_count"`
}

type NotInFileStudentItem struct {
	ID        int64  `json:"id"`
	ClassID   int64  `json:"class_id"`
	ClassName string `json:"class_name"`
	FullName  string `json:"full_name"`
}

type PreviewSummary struct {
	TotalRows             int                    `json:"total_rows"`
	ValidRowsCount        int                    `json:"valid_rows_count"`
	ClassesCount          int                    `json:"classes_count"`
	NewStudentsCount      int                    `json:"new_students_count"`
	ExistingStudentsCount int                    `json:"existing_students_count"`
	DuplicatesCount       int                    `json:"duplicates_count"`
	ErrorsCount           int                    `json:"errors_count"`
	MissingClasses        []MissingClassItem     `json:"missing_classes"`
	NotInFile             []NotInFileStudentItem `json:"not_in_file"`
	NotInFileCount        int                    `json:"not_in_file_count"`
}

type PreviewResponse struct {
	JobID         string         `json:"job_id"`
	Summary       PreviewSummary `json:"summary"`
	First50Errors []RowError     `json:"first_50_errors"`
}

// Preview parses and validates the Excel file without modifying student data.
func (s *Service) Preview(ctx context.Context, schoolID int64, file io.Reader, size int64) (*PreviewResponse, error) {
	parseResult, err := ParseExcelFile(file, size)
	if err != nil {
		return nil, err
	}

	// 1. Fetch existing classes in school
	existingClasses, err := s.database.Queries.ListClassesBySchool(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("mavjud sinflarni yuklashda xatolik: %w", err)
	}

	classMap := make(map[string]int64) // normName -> classID
	for _, c := range existingClasses {
		norm, _ := NormalizeClassName(c.Name)
		classMap[norm] = c.ID
	}

	// 2. Identify missing classes
	missingClassesMap := make(map[string]int)
	for className, count := range parseResult.ClassesFound {
		if _, exists := classMap[className]; !exists {
			missingClassesMap[className] = count
		}
	}

	var missingClassesList []MissingClassItem
	for cls, count := range missingClassesMap {
		missingClassesList = append(missingClassesList, MissingClassItem{
			ClassName:    cls,
			StudentCount: count,
		})
	}

	// 3. Fetch existing students in school to check new vs existing
	existingStudents, err := s.database.Queries.ListStudentsBySchool(ctx, schoolID)
	if err != nil {
		existingStudents = nil
	}

	existingStudentKeyMap := make(map[string]int64) // "classID#normName" -> studentID
	existingActiveByClass := make(map[int64][]NotInFileStudentItem)
	for _, st := range existingStudents {
		normName := st.NameNormalized
		if normName == "" {
			_, normName = NormalizeStudentName(st.FullName)
		}
		key := fmt.Sprintf("%d#%s", st.ClassID, normName)
		existingStudentKeyMap[key] = st.ID

		existingActiveByClass[st.ClassID] = append(existingActiveByClass[st.ClassID], NotInFileStudentItem{
			ID:        st.ID,
			ClassID:   st.ClassID,
			ClassName: st.ClassName,
			FullName:  st.FullName,
		})
	}

	newStudentsCount := 0
	existingStudentsCount := 0
	fileSeenKeys := make(map[string]bool)

	for _, row := range parseResult.ValidRows {
		classID, classExists := classMap[row.ClassName]
		if !classExists {
			// Belongs to a missing class -> will be new students once class is created
			newStudentsCount++
			continue
		}

		key := fmt.Sprintf("%d#%s", classID, row.NameNormalized)
		fileSeenKeys[key] = true
		if _, exists := existingStudentKeyMap[key]; exists {
			existingStudentsCount++
		} else {
			newStudentsCount++
		}
	}

	// Find active students in these classes who are not in the uploaded file
	var notInFileList []NotInFileStudentItem
	for clsName := range parseResult.ClassesFound {
		classID, classExists := classMap[clsName]
		if !classExists {
			continue
		}
		for _, dbSt := range existingActiveByClass[classID] {
			norm := strings.ToLower(dbSt.FullName)
			key := fmt.Sprintf("%d#%s", classID, norm)
			if !fileSeenKeys[key] {
				notInFileList = append(notInFileList, dbSt)
			}
		}
	}

	summary := PreviewSummary{
		TotalRows:             parseResult.TotalRows,
		ValidRowsCount:        len(parseResult.ValidRows),
		ClassesCount:          len(parseResult.ClassesFound),
		NewStudentsCount:      newStudentsCount,
		ExistingStudentsCount: existingStudentsCount,
		DuplicatesCount:       parseResult.DuplicateCount,
		ErrorsCount:           len(parseResult.Errors),
		MissingClasses:        missingClassesList,
		NotInFile:             notInFileList,
		NotInFileCount:        len(notInFileList),
	}

	// Store preview job in import_jobs
	payloadBytes, _ := json.Marshal(parseResult.ValidRows)
	summaryBytes, _ := json.Marshal(summary)
	errorsBytes, _ := json.Marshal(parseResult.Errors)

	job, err := s.database.Queries.CreateImportJob(ctx, db.CreateImportJobParams{
		SchoolID:      schoolID,
		Status:        "pending",
		TotalRows:     int32(len(parseResult.ValidRows)),
		ProcessedRows: 0,
		Payload:       payloadBytes,
		Summary:       summaryBytes,
		Errors:        errorsBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("import jobini saqlashda xatolik: %w", err)
	}

	var jobIDStr string
	if job.ID.Valid {
		jobIDStr = fmt.Sprintf("%x-%x-%x-%x-%x", job.ID.Bytes[0:4], job.ID.Bytes[4:6], job.ID.Bytes[6:8], job.ID.Bytes[8:10], job.ID.Bytes[10:16])
	}

	return &PreviewResponse{
		JobID:         jobIDStr,
		Summary:       summary,
		First50Errors: parseResult.First50Errors,
	}, nil
}

// Commit starts asynchronous chunk processing of the previewed job.
func (s *Service) Commit(schoolID int64, jobID pgtype.UUID, createMissingClasses bool) error {
	ctx := context.Background()
	job, err := s.database.Queries.GetImportJobByID(ctx, db.GetImportJobByIDParams{
		ID:       jobID,
		SchoolID: schoolID,
	})
	if err != nil {
		return fmt.Errorf("import jobi topilmadi: %w", err)
	}

	if job.Status != "pending" {
		return fmt.Errorf("ushbu job allaqachon boshlangan yoki yakunlangan (holati: %s)", job.Status)
	}

	// Launch worker in goroutine
	go s.runImportWorker(schoolID, jobID, job.Payload, createMissingClasses)

	return nil
}

func (s *Service) runImportWorker(schoolID int64, jobID pgtype.UUID, rawPayload []byte, createMissingClasses bool) {
	// School-level worker lock ensures concurrency safety per school
	lock := s.getSchoolLock(schoolID)
	lock.Lock()
	defer lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[IMPORT PANIC] school=%d job=%v: %v", schoolID, jobID, r)
			errMsg, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("Kutilmagan xatolik: %v", r)})
			_ = s.database.Queries.UpdateImportJobStatus(context.Background(), db.UpdateImportJobStatusParams{
				ID:       jobID,
				SchoolID: schoolID,
				Status:   "failed",
				Summary:  errMsg,
			})
		}
	}()

	// Mark status as running
	_ = s.database.Queries.UpdateImportJobStatus(ctx, db.UpdateImportJobStatusParams{
		ID:       jobID,
		SchoolID: schoolID,
		Status:   "running",
	})

	var rows []ParsedStudentRow
	if err := json.Unmarshal(rawPayload, &rows); err != nil {
		s.failJob(jobID, schoolID, "Payload ma'lumotlarini o'qishda xatolik: "+err.Error())
		return
	}

	// 1. Create missing classes if requested
	if createMissingClasses {
		uniqueClasses := make(map[string]bool)
		for _, r := range rows {
			uniqueClasses[r.ClassName] = true
		}
		for cls := range uniqueClasses {
			_, _ = s.database.Queries.CreateClassIfNotExists(ctx, db.CreateClassIfNotExistsParams{
				SchoolID: schoolID,
				Name:     cls,
			})
		}
	}

	// 2. Fetch all classes in school for class ID lookup
	classList, err := s.database.Queries.ListClassesBySchool(ctx, schoolID)
	if err != nil {
		s.failJob(jobID, schoolID, "Sinflarni olishda xatolik: "+err.Error())
		return
	}
	classMap := make(map[string]int64)
	for _, c := range classList {
		norm, _ := NormalizeClassName(c.Name)
		classMap[norm] = c.ID
	}

	// 3. Process in chunks of 500 rows using pgx CopyFrom into temp staging table
	chunkSize := 500
	total := len(rows)
	processed := 0
	createdCount := 0
	updatedCount := 0
	failedCount := 0

	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		chunk := rows[i:end]

		// Prepare staging data
		type StagingRow struct {
			SchoolID       int64
			ClassID        int64
			FullName       string
			NameNormalized string
		}

		var validStagingRows []StagingRow
		for _, r := range chunk {
			cid, ok := classMap[r.ClassName]
			if !ok {
				failedCount++
				continue
			}
			validStagingRows = append(validStagingRows, StagingRow{
				SchoolID:       schoolID,
				ClassID:        cid,
				FullName:       r.FullName,
				NameNormalized: r.NameNormalized,
			})
		}

		if len(validStagingRows) > 0 {
			// Begin transaction for chunk
			tx, err := s.database.Pool.Begin(ctx)
			if err != nil {
				s.failJob(jobID, schoolID, "Tranzaksiyani boshlashda xatolik: "+err.Error())
				return
			}

			// Create temporary staging table
			_, err = tx.Exec(ctx, `
				CREATE TEMP TABLE staging_students (
					school_id BIGINT,
					class_id BIGINT,
					full_name VARCHAR(255),
					name_normalized VARCHAR(255)
				) ON COMMIT DROP;
			`)
			if err != nil {
				_ = tx.Rollback(ctx)
				s.failJob(jobID, schoolID, "Staging jadvalini yaratishda xatolik: "+err.Error())
				return
			}

			// CopyFrom staging
			copySource := pgx.CopyFromSlice(len(validStagingRows), func(idx int) ([]any, error) {
				row := validStagingRows[idx]
				return []any{row.SchoolID, row.ClassID, row.FullName, row.NameNormalized}, nil
			})

			_, err = tx.CopyFrom(
				ctx,
				pgx.Identifier{"staging_students"},
				[]string{"school_id", "class_id", "full_name", "name_normalized"},
				copySource,
			)
			if err != nil {
				_ = tx.Rollback(ctx)
				s.failJob(jobID, schoolID, "Ma'lumotlarni nusxalashda xatolik: "+err.Error())
				return
			}

			// Insert / Update in single batch statement
			tag, err := tx.Exec(ctx, `
				INSERT INTO students (school_id, class_id, full_name, name_normalized, active)
				SELECT school_id, class_id, full_name, name_normalized, TRUE
				FROM staging_students
				ON CONFLICT (class_id, name_normalized)
				DO UPDATE SET full_name = EXCLUDED.full_name, active = TRUE, updated_at = NOW();
			`)
			if err != nil {
				_ = tx.Rollback(ctx)
				s.failJob(jobID, schoolID, "O'quvchilarni saqlashda xatolik: "+err.Error())
				return
			}

			if err := tx.Commit(ctx); err != nil {
				s.failJob(jobID, schoolID, "Tranzaksiyani saqlashda xatolik: "+err.Error())
				return
			}

			createdCount += int(tag.RowsAffected())
		}

		processed += len(chunk)

		// Update progress in DB
		_ = s.database.Queries.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
			ID:            jobID,
			SchoolID:      schoolID,
			ProcessedRows: int32(processed),
			CreatedCount:  int32(createdCount),
			UpdatedCount:  int32(updatedCount),
			SkippedCount:  0,
			FailedCount:   int32(failedCount),
		})
	}

	// Completed successfully!
	summaryData, _ := json.Marshal(map[string]any{
		"total_rows":    total,
		"created_count": createdCount,
		"updated_count": updatedCount,
		"failed_count":  failedCount,
		"completed_at":  time.Now().Format(time.RFC3339),
	})

	_ = s.database.Queries.UpdateImportJobStatus(ctx, db.UpdateImportJobStatusParams{
		ID:       jobID,
		SchoolID: schoolID,
		Status:   "done",
		Summary:  summaryData,
	})
}

func (s *Service) failJob(jobID pgtype.UUID, schoolID int64, reason string) {
	log.Printf("[IMPORT ERROR] school=%d job=%v: %s", schoolID, jobID, reason)
	errBytes, _ := json.Marshal(map[string]string{"error": reason})
	_ = s.database.Queries.UpdateImportJobStatus(context.Background(), db.UpdateImportJobStatusParams{
		ID:       jobID,
		SchoolID: schoolID,
		Status:   "failed",
		Summary:  errBytes,
	})
}

// GetJob retrieves an import job by ID for the school.
func (s *Service) GetJob(ctx context.Context, schoolID int64, jobID pgtype.UUID) (*db.ImportJob, error) {
	job, err := s.database.Queries.GetImportJobByID(ctx, db.GetImportJobByIDParams{
		ID:       jobID,
		SchoolID: schoolID,
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GenerateErrorReport generates an .xlsx spreadsheet of all errors recorded in the import job.
func (s *Service) GenerateErrorReport(ctx context.Context, schoolID int64, jobID pgtype.UUID) ([]byte, error) {
	job, err := s.GetJob(ctx, schoolID, jobID)
	if err != nil {
		return nil, err
	}

	if len(job.Errors) == 0 {
		return nil, fmt.Errorf("ushbu importda xatoliklar qayd etilmagan")
	}

	var errorsList []RowError
	if err := json.Unmarshal(job.Errors, &errorsList); err != nil {
		return nil, fmt.Errorf("xatoliklarni o'qishda xatolik: %w", err)
	}

	return GenerateErrorReport(errorsList)
}

// DeactivateMissingStudents sets active = FALSE for the given student IDs belonging to the school.
func (s *Service) DeactivateMissingStudents(ctx context.Context, schoolID int64, studentIDs []int64) error {
	if len(studentIDs) == 0 {
		return nil
	}
	return s.database.Queries.DeactivateMissingStudents(ctx, db.DeactivateMissingStudentsParams{
		SchoolID: pgtype.Int8{Int64: schoolID, Valid: true},
		Column2:  studentIDs,
	})
}

// MarkStaleJobsFailed marks any pending/running jobs as failed on server restart.
func (s *Service) MarkStaleJobsFailed(ctx context.Context) error {
	return s.database.Queries.FailStaleRunningJobs(ctx)
}
