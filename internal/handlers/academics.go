package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/security"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// AcademicsAPIHandler handles Category 2 and Category 3 academic operations.
type AcademicsAPIHandler struct {
	svc *services.Service
}

// NewAcademicsAPIHandler creates a new academics handler.
func NewAcademicsAPIHandler(svc *services.Service) *AcademicsAPIHandler {
	return &AcademicsAPIHandler{svc: svc}
}

// Route 4: GET /api/v1/timeTable
func (h *AcademicsAPIHandler) GetTimeTable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" && claims != nil {
		classID = claims.ClassID
	}
	sectionID := r.URL.Query().Get("section_id")
	if sectionID == "" && claims != nil {
		sectionID = claims.SectionID
	}
	if classID == "" {
		classID = "10"
	}
	if sectionID == "" {
		sectionID = "A"
	}

	data, err := h.svc.GetTimeTable(r.Context(), classID, sectionID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 5: PUT /api/v1/updateTimeTable
func (h *AcademicsAPIHandler) UpdateTimeTable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.UpdateTimeTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.UpdateTimeTable(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"message": "timetable_updated"}))
}

// Route 6: GET /api/v1/labTimeTable
func (h *AcademicsAPIHandler) GetLabTimeTable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" && claims != nil {
		classID = claims.ClassID
	}
	if classID == "" {
		classID = "10"
	}

	data, err := h.svc.GetLabTimeTable(r.Context(), classID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 7: GET /api/v1/syllabus
func (h *AcademicsAPIHandler) GetSyllabus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" && claims != nil {
		classID = claims.ClassID
	}
	if classID == "" {
		classID = "10"
	}

	data, err := h.svc.GetSyllabus(r.Context(), classID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 8: GET /api/v1/dairy
func (h *AcademicsAPIHandler) GetDiary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	classID := r.URL.Query().Get("class_id")
	if classID == "" {
		classID = claims.ClassID
	}
	sectionID := r.URL.Query().Get("section_id")
	if sectionID == "" {
		sectionID = claims.SectionID
	}
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}
	if classID == "" {
		classID = "10"
	}
	if sectionID == "" {
		sectionID = "A"
	}

	if !claims.HasStudentAccess(studentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	data, err := h.svc.GetDiary(r.Context(), classID, sectionID, studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 9: POST /api/v1/dairy
func (h *AcademicsAPIHandler) AddDiary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.PostDiaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	req.Remark = security.SanitizeText(req.Remark)
	req.Conduct = security.SanitizeText(req.Conduct)

	refNo, err := h.svc.AddDiary(r.Context(), &req, "Dr. R. Iyer", claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{"message": "diary_entry_added"}))
}

// Route 10: GET /api/v1/homework
func (h *AcademicsAPIHandler) GetHomework(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" && claims != nil {
		classID = claims.ClassID
	}
	sectionID := r.URL.Query().Get("section_id")
	if sectionID == "" && claims != nil {
		sectionID = claims.SectionID
	}
	if classID == "" {
		classID = "10"
	}
	if sectionID == "" {
		sectionID = "A"
	}

	data, err := h.svc.GetHomework(r.Context(), classID, sectionID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 11: POST /api/v1/homework
func (h *AcademicsAPIHandler) PostHomework(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.PostHomeworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	req.Title = security.SanitizeText(req.Title)
	req.Description = security.SanitizeText(req.Description)

	refNo, err := h.svc.PostHomework(r.Context(), &req, "Dr. K. Rao", claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{"message": "homework_posted"}))
}

// Route 12: POST /api/v1/homework/submit
func (h *AcademicsAPIHandler) SubmitHomework(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var body struct {
		HomeworkID string `json:"homework_id"`
		StudentID  string `json:"student_id"`
		FileURL    string `json:"file_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if body.StudentID == "" {
		body.StudentID = claims.UserID
	}

	if !claims.HasStudentAccess(body.StudentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	if body.FileURL != "" {
		if err := security.ValidateSafeURL(body.FileURL); err != nil {
			audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_file_url: "+err.Error()))
			return
		}
	}

	refNo, err := h.svc.SubmitHomework(r.Context(), body.HomeworkID, body.StudentID, body.FileURL, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "submitted"}))
}

// Route 18: POST /api/v1/setExamDates
func (h *AcademicsAPIHandler) SetExamDates(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.SetExamDatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.SetExamDates(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "exam_dates_configured"}))
}

// Route 19: GET /api/v1/getExamDates
func (h *AcademicsAPIHandler) GetExamDates(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" && claims != nil {
		classID = claims.ClassID
	}
	if classID == "" {
		classID = "10"
	}

	data, err := h.svc.GetExamDates(r.Context(), classID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 20: GET /api/v1/examTimeTable
func (h *AcademicsAPIHandler) GetExamTimeTable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" && claims != nil {
		classID = claims.ClassID
	}
	if classID == "" {
		classID = "10"
	}

	data, err := h.svc.GetExamTimeTable(r.Context(), classID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 21: GET /api/v1/exam (Guidelines)
func (h *AcademicsAPIHandler) GetExamGuidelines(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" && claims != nil {
		studentID = claims.UserID
	}
	if studentID == "" {
		studentID = "STU1001"
	}

	if claims != nil && !claims.HasStudentAccess(studentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	data, err := h.svc.GetExamGuidelines(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 22: GET /api/v1/results
func (h *AcademicsAPIHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	if !claims.HasStudentAccess(studentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	data, err := h.svc.GetResults(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 23: POST /api/v1/results
func (h *AcademicsAPIHandler) BatchSaveResults(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.BatchSaveResultsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.BatchSaveResults(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "results_published"}))
}

// Route 24: GET /api/v1/reportCard
func (h *AcademicsAPIHandler) GetReportCard(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	if !claims.HasStudentAccess(studentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	data, err := h.svc.GetReportCard(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}
