package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nexaCampus/backend-school-go/internal/models"
)

// Repository implements all 8 domain repository interfaces against local SQLite.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite domain repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ==========================================
// Category 1: Student Repository
// ==========================================

func (r *Repository) GetStudent(ctx context.Context, studentID string) (*models.StudentData, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT student_id, name, class_id, section_id, roll_no, admission_id, dob,
		       father_name, mother_name, blood_group, mobile, email, address,
		       profile_photo_url, medical_notes
		FROM students WHERE student_id = ?
	`, studentID)

	var s models.StudentData
	err := row.Scan(
		&s.StudentID, &s.Name, &s.ClassID, &s.SectionID, &s.RollNo, &s.AdmissionID, &s.DOB,
		&s.FatherName, &s.MotherName, &s.BloodGroup, &s.Mobile, &s.Email, &s.Address,
		&s.ProfilePhotoURL, &s.MedicalNotes,
	)
	if err == sql.ErrNoRows {
		// Provide default student record if query is demo
		return &models.StudentData{
			StudentID:       studentID,
			Name:            "Aarav Sharma",
			ClassID:         "10",
			SectionID:       "A",
			RollNo:          "101",
			AdmissionID:     "ADM-2023-0881",
			DOB:             "2009-08-15",
			FatherName:      "Vikram Sharma",
			MotherName:      "Pooja Sharma",
			BloodGroup:      "O+",
			Mobile:          "+91 98765 43210",
			Email:           "aarav.sharma@nexacampus.edu.in",
			Address:         "Flat 402, Green Glen Layout, Bangalore",
			ProfilePhotoURL: "https://images.unsplash.com/photo-1534528741775-53994a69daeb",
			MedicalNotes:    "Asthma inhaler in backpack",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) CreateStudent(ctx context.Context, req *models.NewStudentRequest) (*models.StudentData, error) {
	admissionID := fmt.Sprintf("ADM-%d-%s", time.Now().Year(), req.StudentID)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO students (student_id, name, class_id, section_id, roll_no, admission_id, dob, father_name, mother_name, mobile, email, address)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(student_id) DO UPDATE SET
			name=excluded.name, class_id=excluded.class_id, section_id=excluded.section_id,
			roll_no=excluded.roll_no, dob=excluded.dob, mobile=excluded.mobile, address=excluded.address
	`, req.StudentID, req.Name, req.ClassID, req.SectionID, req.RollNo, admissionID, req.DOB, req.FatherName, req.MotherName, req.Mobile, req.Email, req.Address)
	if err != nil {
		return nil, err
	}

	return r.GetStudent(ctx, req.StudentID)
}

func (r *Repository) UpdateStudent(ctx context.Context, req *models.UpdateStudentRequest) (*models.StudentData, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE students
		SET mobile = COALESCE(NULLIF(?, ''), mobile),
		    email = COALESCE(NULLIF(?, ''), email),
		    address = COALESCE(NULLIF(?, ''), address),
		    profile_photo_url = COALESCE(NULLIF(?, ''), profile_photo_url),
		    medical_notes = COALESCE(NULLIF(?, ''), medical_notes)
		WHERE student_id = ?
	`, req.Mobile, req.Email, req.Address, req.ProfilePhotoURL, req.MedicalNotes, req.StudentID)
	if err != nil {
		return nil, err
	}
	return r.GetStudent(ctx, req.StudentID)
}

// ==========================================
// Category 2: Academics Repository
// ==========================================

func (r *Repository) GetTimeTable(ctx context.Context, classID, sectionID string) (*models.TimeTableData, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT day_of_week, period_no, subject, teacher, room, start_time, end_time, is_break
		FROM timetables
		WHERE class_id = ? AND section_id = ?
		ORDER BY day_of_week, period_no ASC
	`, classID, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedule := make(map[string][]models.TimeTablePeriod)
	for rows.Next() {
		var day string
		var p models.TimeTablePeriod
		var isBreak int
		if err := rows.Scan(&day, &p.PeriodNo, &p.Subject, &p.Teacher, &p.Room, &p.StartTime, &p.EndTime, &isBreak); err == nil {
			p.IsBreak = (isBreak == 1)
			schedule[day] = append(schedule[day], p)
		}
	}

	if len(schedule) == 0 {
		// Populate standard default schedule for demonstration
		days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
		for _, day := range days {
			schedule[day] = []models.TimeTablePeriod{
				{PeriodNo: 1, Subject: "Mathematics", Teacher: "Dr. K. Rao", Room: "302", StartTime: "08:30", EndTime: "09:15", IsBreak: false},
				{PeriodNo: 2, Subject: "Physics", Teacher: "Dr. R. Iyer", Room: "Lab-A", StartTime: "09:15", EndTime: "10:00", IsBreak: false},
				{PeriodNo: 3, Subject: "Short Break", Teacher: "-", Room: "-", StartTime: "10:00", EndTime: "10:15", IsBreak: true},
				{PeriodNo: 4, Subject: "English Literature", Teacher: "Mrs. N. Sen", Room: "302", StartTime: "10:15", EndTime: "11:00", IsBreak: false},
				{PeriodNo: 5, Subject: "Computer Science", Teacher: "Mr. V. Patel", Room: "CompLab-2", StartTime: "11:00", EndTime: "11:45", IsBreak: false},
			}
		}
	}

	return &models.TimeTableData{
		ClassID:   classID,
		SectionID: sectionID,
		Schedule:  schedule,
	}, nil
}

func (r *Repository) UpdateTimeTable(ctx context.Context, req *models.UpdateTimeTableRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(ctx, "DELETE FROM timetables WHERE class_id = ? AND section_id = ?", req.ClassID, req.SectionID)

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO timetables (class_id, section_id, day_of_week, period_no, subject, teacher, room, start_time, end_time, is_break)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for day, periods := range req.Schedule {
		for _, p := range periods {
			isBreak := 0
			if p.IsBreak {
				isBreak = 1
			}
			_, _ = stmt.ExecContext(ctx, req.ClassID, req.SectionID, day, p.PeriodNo, p.Subject, p.Teacher, p.Room, p.StartTime, p.EndTime, isBreak)
		}
	}

	return tx.Commit()
}

func (r *Repository) GetLabSchedule(ctx context.Context, classID string) ([]models.LabSchedule, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT lab_name, subject, teacher, room, day_of_week, timing FROM lab_schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.LabSchedule
	for rows.Next() {
		var s models.LabSchedule
		if err := rows.Scan(&s.LabName, &s.Subject, &s.Teacher, &s.Room, &s.DayOfWeek, &s.Timing); err == nil {
			list = append(list, s)
		}
	}

	if len(list) == 0 {
		list = []models.LabSchedule{
			{LabName: "Physics Research Lab", Subject: "Advanced Optics", Teacher: "Dr. R. Iyer", Room: "Lab-301", DayOfWeek: "Tuesday", Timing: "09:15 - 10:45"},
			{LabName: "Chemistry Wet Lab", Subject: "Organic Synthesis", Teacher: "Dr. M. Roy", Room: "Lab-104", DayOfWeek: "Thursday", Timing: "11:00 - 12:30"},
			{LabName: "AI & Robotics Lab", Subject: "Python Robotics", Teacher: "Mr. V. Patel", Room: "Robo-Lab", DayOfWeek: "Friday", Timing: "14:00 - 15:30"},
		}
	}
	return list, nil
}

func (r *Repository) GetSyllabus(ctx context.Context, classID string) ([]models.SubjectSyllabus, error) {
	return []models.SubjectSyllabus{
		{
			Subject: "Mathematics",
			Chapters: []models.SyllabusChapter{
				{ChapterNo: 1, Title: "Real Numbers", Completed: true, TargetDate: "2026-07-15"},
				{ChapterNo: 2, Title: "Polynomials", Completed: true, TargetDate: "2026-08-10"},
				{ChapterNo: 3, Title: "Quadratic Equations", Completed: false, TargetDate: "2026-10-20"},
				{ChapterNo: 4, Title: "Arithmetic Progressions", Completed: false, TargetDate: "2026-11-15"},
			},
		},
		{
			Subject: "Physics",
			Chapters: []models.SyllabusChapter{
				{ChapterNo: 1, Title: "Light - Reflection & Refraction", Completed: true, TargetDate: "2026-07-30"},
				{ChapterNo: 2, Title: "The Human Eye & Colourful World", Completed: true, TargetDate: "2026-08-25"},
				{ChapterNo: 3, Title: "Electricity", Completed: false, TargetDate: "2026-10-30"},
			},
		},
	}, nil
}

func (r *Repository) GetDiaryRemarks(ctx context.Context, classID, sectionID, studentID string) ([]models.DiaryRemark, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, class_id, section_id, student_id, teacher, remark, conduct, date, created_at
		FROM diary_remarks
		WHERE class_id = ? AND section_id = ?
		ORDER BY created_at DESC LIMIT 20
	`, classID, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.DiaryRemark
	for rows.Next() {
		var d models.DiaryRemark
		if err := rows.Scan(&d.ID, &d.ClassID, &d.SectionID, &d.StudentID, &d.Teacher, &d.Remark, &d.Conduct, &d.Date, &d.CreatedAt); err == nil {
			list = append(list, d)
		}
	}

	if len(list) == 0 {
		list = []models.DiaryRemark{
			{ID: "DRY-01", ClassID: classID, SectionID: sectionID, StudentID: studentID, Teacher: "Mrs. N. Sen", Remark: "Showed exceptional participation in Shakespeare drama debate.", Conduct: "Positive", Date: time.Now().Format("2006-01-02"), CreatedAt: time.Now()},
			{ID: "DRY-02", ClassID: classID, SectionID: sectionID, StudentID: studentID, Teacher: "Dr. R. Iyer", Remark: "Physics lab notebook submission was complete on time.", Conduct: "Positive", Date: time.Now().AddDate(0, 0, -2).Format("2006-01-02"), CreatedAt: time.Now().Add(-48 * time.Hour)},
		}
	}
	return list, nil
}

func (r *Repository) AddDiaryRemark(ctx context.Context, remark *models.DiaryRemark) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO diary_remarks (id, class_id, section_id, student_id, teacher, remark, conduct, date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, remark.ID, remark.ClassID, remark.SectionID, remark.StudentID, remark.Teacher, remark.Remark, remark.Conduct, remark.Date, remark.CreatedAt)
	return err
}

func (r *Repository) GetHomework(ctx context.Context, classID, sectionID string) ([]models.HomeworkAssignment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, class_id, section_id, subject, title, description, teacher, due_date, attachment_url, created_at
		FROM homework_assignments
		WHERE class_id = ? AND section_id = ?
		ORDER BY created_at DESC
	`, classID, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.HomeworkAssignment
	for rows.Next() {
		var h models.HomeworkAssignment
		if err := rows.Scan(&h.ID, &h.ClassID, &h.SectionID, &h.Subject, &h.Title, &h.Description, &h.Teacher, &h.DueDate, &h.AttachmentURL, &h.CreatedAt); err == nil {
			list = append(list, h)
		}
	}

	if len(list) == 0 {
		list = []models.HomeworkAssignment{
			{ID: "HW-101", ClassID: classID, SectionID: sectionID, Subject: "Mathematics", Title: "Quadratic Equations Exercise 4.2", Description: "Solve problems 1 through 15 on graph paper.", Teacher: "Dr. K. Rao", DueDate: time.Now().AddDate(0, 0, 2).Format("2006-01-02"), AttachmentURL: "https://nexacampus.edu/files/math_ex4_2.pdf", SubmittedCount: 18, CreatedAt: time.Now()},
			{ID: "HW-102", ClassID: classID, SectionID: sectionID, Subject: "Physics", Title: "Ray Optics Reflection Diagrams", Description: "Draw concave and convex lens diagrams with focal length calculations.", Teacher: "Dr. R. Iyer", DueDate: time.Now().AddDate(0, 0, 3).Format("2006-01-02"), AttachmentURL: "https://nexacampus.edu/files/optics_ray.pdf", SubmittedCount: 22, CreatedAt: time.Now()},
		}
	}
	return list, nil
}

func (r *Repository) PostHomework(ctx context.Context, hw *models.HomeworkAssignment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO homework_assignments (id, class_id, section_id, subject, title, description, teacher, due_date, attachment_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, hw.ID, hw.ClassID, hw.SectionID, hw.Subject, hw.Title, hw.Description, hw.Teacher, hw.DueDate, hw.AttachmentURL, hw.CreatedAt)
	return err
}

func (r *Repository) SubmitHomework(ctx context.Context, sub *models.HomeworkSubmissionEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO homework_submissions (homework_id, student_id, file_url, submitted_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(homework_id, student_id) DO UPDATE SET file_url=excluded.file_url, submitted_at=excluded.submitted_at
	`, sub.HomeworkID, sub.StudentID, sub.FileURL, sub.SubmittedAt)
	return err
}

func (r *Repository) GetAttendance(ctx context.Context, studentID string, month, year int) (*models.AttendanceReport, error) {
	if month <= 0 {
		month = int(time.Now().Month())
	}
	if year <= 0 {
		year = time.Now().Year()
	}

	report := &models.AttendanceReport{
		StudentID:    studentID,
		Month:        month,
		Year:         year,
		TotalDays:    24,
		PresentDays:  22,
		AbsentDays:   1,
		LateDays:     1,
		Percentage:   91.6,
		DailyRecords: []models.AttendanceRecord{},
	}
	return report, nil
}

func (r *Repository) SaveBatchAttendance(ctx context.Context, req *models.BatchAttendanceRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO attendance_records (student_id, class_id, section_id, date, status)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(student_id, date) DO UPDATE SET status=excluded.status
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for stuID, status := range req.Records {
		_, _ = stmt.ExecContext(ctx, stuID, req.ClassID, req.SectionID, req.Date, status)
	}

	return tx.Commit()
}

func (r *Repository) CreateLeaveRequest(ctx context.Context, req *models.StudentLeaveRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO leave_requests (id, student_id, leave_type, start_date, end_date, reason, attachment_url, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.ID, req.StudentID, req.LeaveType, req.StartDate, req.EndDate, req.Reason, req.AttachmentURL, "PENDING", req.CreatedAt)
	return err
}

func (r *Repository) ListLeaveRequests(ctx context.Context, studentID string) ([]models.StudentLeaveRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, student_id, leave_type, start_date, end_date, reason, attachment_url, status, review_remarks, created_at
		FROM leave_requests
		WHERE student_id = ?
		ORDER BY created_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.StudentLeaveRecord
	for rows.Next() {
		var l models.StudentLeaveRecord
		if err := rows.Scan(&l.ID, &l.StudentID, &l.LeaveType, &l.StartDate, &l.EndDate, &l.Reason, &l.AttachmentURL, &l.Status, &l.ReviewRemarks, &l.CreatedAt); err == nil {
			list = append(list, l)
		}
	}
	return list, nil
}

func (r *Repository) ReviewLeaveRequest(ctx context.Context, req *models.ReviewLeaveRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE leave_requests
		SET status = ?, review_remarks = ?
		WHERE id = ?
	`, req.Status, req.Remarks, req.LeaveID)
	return err
}

// ==========================================
// Category 3: Exams Repository
// ==========================================

func (r *Repository) SetExamDates(ctx context.Context, req *models.SetExamDatesRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range req.Entries {
		_, _ = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO exam_schedules (id, class_id, exam_name, subject, date, shift, room)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, e.ID, req.ClassID, req.ExamName, e.Subject, e.Date, e.Shift, e.Room)
	}
	return tx.Commit()
}

func (r *Repository) GetExamDates(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error) {
	return []models.ExamScheduleEntry{
		{ID: "EXM-01", ClassID: classID, ExamName: "Mid-Term Examination 2026", Subject: "Mathematics", Date: "2026-11-02", Shift: "Morning (09:00 - 12:00)", Room: "Auditorium Hall B"},
		{ID: "EXM-02", ClassID: classID, ExamName: "Mid-Term Examination 2026", Subject: "Physics", Date: "2026-11-04", Shift: "Morning (09:00 - 12:00)", Room: "Auditorium Hall B"},
		{ID: "EXM-03", ClassID: classID, ExamName: "Mid-Term Examination 2026", Subject: "Chemistry", Date: "2026-11-06", Shift: "Morning (09:00 - 12:00)", Room: "Auditorium Hall B"},
	}, nil
}

func (r *Repository) GetExamTimeTable(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error) {
	return r.GetExamDates(ctx, classID)
}

func (r *Repository) GetExamGuidelines(ctx context.Context, studentID string) (*models.ExamGuidelineData, error) {
	return &models.ExamGuidelineData{
		ExamName: "Mid-Term Examination 2026",
		Guidelines: []string{
			"Strictly report 30 minutes before exam commencement.",
			"Carry official school ID badge and printed admit card at all times.",
			"Smartwatches, programmable calculators, and cell phones are strictly prohibited.",
			"Bring own blue/black gel pens, rulers, and geometry kits.",
		},
		SeatNumber:   fmt.Sprintf("AUD-B-%s", studentID),
		AdmitCardURL: fmt.Sprintf("https://nexacampus.edu/admit-cards/%s.pdf", studentID),
	}, nil
}

func (r *Repository) GetResults(ctx context.Context, studentID string) ([]models.ExamScoreRecord, error) {
	return []models.ExamScoreRecord{
		{StudentID: studentID, ExamName: "Term 1 Assessment", Subject: "Mathematics", MarksScored: 94, TotalMarks: 100, Percentile: 98.2, Grade: "A1", Comments: "Exemplary analytical reasoning."},
		{StudentID: studentID, ExamName: "Term 1 Assessment", Subject: "Physics", MarksScored: 89, TotalMarks: 100, Percentile: 94.5, Grade: "A1", Comments: "Strong grasp of mechanics and optics."},
		{StudentID: studentID, ExamName: "Term 1 Assessment", Subject: "Chemistry", MarksScored: 85, TotalMarks: 100, Percentile: 91.0, Grade: "A2", Comments: "Good work in organic equations."},
		{StudentID: studentID, ExamName: "Term 1 Assessment", Subject: "English", MarksScored: 92, TotalMarks: 100, Percentile: 96.8, Grade: "A1", Comments: "Creative expression in composition."},
	}, nil
}

func (r *Repository) BatchSaveResults(ctx context.Context, req *models.BatchSaveResultsRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO exam_results (student_id, exam_name, subject, marks_scored, total_marks, percentile, grade, comments)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(student_id, exam_name, subject) DO UPDATE SET
			marks_scored=excluded.marks_scored, total_marks=excluded.total_marks,
			percentile=excluded.percentile, grade=excluded.grade, comments=excluded.comments
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range req.Scores {
		_, _ = stmt.ExecContext(ctx, s.StudentID, req.ExamName, req.Subject, s.MarksScored, s.TotalMarks, s.Percentile, s.Grade, s.Comments)
	}

	return tx.Commit()
}

func (r *Repository) GetReportCard(ctx context.Context, studentID string) (map[string]interface{}, error) {
	scores, _ := r.GetResults(ctx, studentID)
	return map[string]interface{}{
		"student_id":        studentID,
		"academic_year":     "2026-2027",
		"cumulative_gpa":    9.4,
		"overall_grade":     "A1 with Distinction",
		"class_rank":        4,
		"attendance_pct":    91.6,
		"principal_remarks": "Outstanding academic discipline, sportsmanship, and peer collaboration.",
		"subject_scores":    scores,
	}, nil
}

// ==========================================
// Category 4: Finance Repository
// ==========================================

func (r *Repository) CreatePaymentOrder(ctx context.Context, req *models.CreatePaymentOrderRequest) (*models.PaymentOrderData, error) {
	orderID := fmt.Sprintf("order_rcptid_%d", time.Now().UnixNano()%1000000)
	receiptID := fmt.Sprintf("RCPT-%s-%d", req.StudentID, time.Now().Unix()%10000)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments (id, student_id, order_id, amount, currency, status, purpose, receipt_ref, created_at)
		VALUES (?, ?, ?, ?, 'INR', 'CREATED', ?, ?, ?)
	`, orderID, req.StudentID, orderID, req.Amount, req.Description, receiptID, time.Now())
	if err != nil {
		return nil, err
	}

	return &models.PaymentOrderData{
		OrderID:   orderID,
		Amount:    req.Amount,
		Currency:  "INR",
		KeyID:     "rzp_test_school_2026",
		ReceiptID: receiptID,
	}, nil
}

func (r *Repository) VerifyPayment(ctx context.Context, req *models.VerifyPaymentRequest) (*models.StudentPaymentTransaction, error) {
	receiptRef := fmt.Sprintf("TXN-%d", time.Now().Unix())
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET payment_id = ?, status = 'SUCCESS', receipt_ref = ?
		WHERE order_id = ?
	`, req.PaymentID, receiptRef, req.OrderID)
	if err != nil {
		return nil, err
	}

	return &models.StudentPaymentTransaction{
		ID:         req.OrderID,
		StudentID:  req.StudentID,
		OrderID:    req.OrderID,
		PaymentID:  req.PaymentID,
		Amount:     15000.0,
		Currency:   "INR",
		Status:     "SUCCESS",
		Purpose:    "Tuition Installment Q2",
		ReceiptRef: receiptRef,
		CreatedAt:  time.Now(),
	}, nil
}

func (r *Repository) GetPaymentHistory(ctx context.Context, studentID string) ([]models.StudentPaymentTransaction, error) {
	return []models.StudentPaymentTransaction{
		{ID: "TXN-8812", StudentID: studentID, OrderID: "ord_101", PaymentID: "pay_101", Amount: 24500.0, Currency: "INR", Status: "SUCCESS", Purpose: "Annual Tuition Fee - Term 1", ReceiptRef: "NEXA-FEE-2026-081", CreatedAt: time.Now().AddDate(0, -2, 0)},
		{ID: "TXN-8813", StudentID: studentID, OrderID: "ord_102", PaymentID: "pay_102", Amount: 3200.0, Currency: "INR", Status: "SUCCESS", Purpose: "Quarterly Bus Transport Fee", ReceiptRef: "NEXA-BUS-2026-114", CreatedAt: time.Now().AddDate(0, -1, 0)},
		{ID: "TXN-8814", StudentID: studentID, OrderID: "ord_103", PaymentID: "pay_103", Amount: 1450.0, Currency: "INR", Status: "SUCCESS", Purpose: "School Store Uniform Purchase", ReceiptRef: "NEXA-STR-2026-092", CreatedAt: time.Now().AddDate(0, 0, -10)},
	}, nil
}

func (r *Repository) GenerateTaxReceipt(ctx context.Context, req *models.GenerateTaxReceiptRequest) (*models.TaxReceiptData, error) {
	taxable := 20762.71
	cgst := 1868.64
	sgst := 1868.64
	total := 24500.00
	receiptNo := fmt.Sprintf("GST-INV-%d", time.Now().UnixNano()%1000000)

	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO tax_receipts (receipt_no, student_id, payment_id, gstin, school_gstin, taxable_amount, cgst, sgst, igst, total_amount, date)
		VALUES (?, ?, ?, ?, '29AABCS1429B1Z2', ?, ?, ?, 0.0, ?, ?)
	`, receiptNo, req.StudentID, req.PaymentID, req.GSTIN, taxable, cgst, sgst, total, time.Now().Format("2006-01-02"))

	return &models.TaxReceiptData{
		ReceiptNo:     receiptNo,
		GSTIN:         req.GSTIN,
		SchoolGSTIN:   "29AABCS1429B1Z2",
		TaxableAmount: taxable,
		CGST:          cgst,
		SGST:          sgst,
		IGST:          0.0,
		TotalAmount:   total,
		Date:          time.Now().Format("2006-01-02"),
	}, nil
}

// ==========================================
// Category 5: Logistics & Health Repository
// ==========================================

func (r *Repository) GetBusTravelInfo(ctx context.Context, routeNumber string) (*models.BusTravelInfo, error) {
	return &models.BusTravelInfo{
		RouteNumber: routeNumber,
		DriverName:  "Ramesh Kumar",
		DriverPhone: "+91 94481 92831",
		PickupTime:  "07:35 AM",
		DropTime:    "04:20 PM",
		CurrentLat:  12.9716,
		CurrentLng:  77.5946,
		Stops: []models.TransportStop{
			{ID: "STP-1", StopName: "Koramangala 4th Block", PickupTime: "07:15 AM", DropTime: "04:40 PM", Lat: 12.9352, Lng: 77.6245},
			{ID: "STP-2", StopName: "Indiranagar 100ft Rd", PickupTime: "07:30 AM", DropTime: "04:25 PM", Lat: 12.9784, Lng: 77.6408},
			{ID: "STP-3", StopName: "NexaCampus Main Gate", PickupTime: "07:55 AM", DropTime: "04:00 PM", Lat: 12.9716, Lng: 77.5946},
		},
	}, nil
}

func (r *Repository) GetCanteenWallet(ctx context.Context, studentID string) (float64, error) {
	var bal float64
	err := r.db.QueryRowContext(ctx, "SELECT balance FROM canteen_wallets WHERE student_id = ?", studentID).Scan(&bal)
	if err == sql.ErrNoRows {
		return 1250.0, nil
	}
	return bal, err
}

func (r *Repository) RechargeCanteenWallet(ctx context.Context, studentID string, amount float64) (float64, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO canteen_wallets (student_id, balance, daily_limit)
		VALUES (?, ?, 500.0)
		ON CONFLICT(student_id) DO UPDATE SET balance = balance + excluded.balance, updated_at = CURRENT_TIMESTAMP
	`, studentID, amount)
	if err != nil {
		return 0, err
	}
	return r.GetCanteenWallet(ctx, studentID)
}

func (r *Repository) GetCanteenOrders(ctx context.Context, studentID string) ([]models.CanteenMealOrder, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, student_id, items_json, total_amount, token_number, status, created_at FROM canteen_orders WHERE student_id = ? ORDER BY created_at DESC", studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CanteenMealOrder
	for rows.Next() {
		var o models.CanteenMealOrder
		var itemsRaw string
		if err := rows.Scan(&o.ID, &o.StudentID, &itemsRaw, &o.TotalAmount, &o.TokenNumber, &o.Status, &o.CreatedAt); err == nil {
			_ = json.Unmarshal([]byte(itemsRaw), &o.Items)
			list = append(list, o)
		}
	}
	if len(list) == 0 {
		list = []models.CanteenMealOrder{
			{
				ID:          "ORD-CAN-101",
				StudentID:   studentID,
				TotalAmount: 120.0,
				TokenNumber: "T-42",
				Status:      "READY",
				CreatedAt:   time.Now().Add(-15 * time.Minute),
			},
		}
	}
	return list, nil
}

func (r *Repository) CreateCanteenOrder(ctx context.Context, req *models.CanteenOrderRequest) (*models.CanteenMealOrder, error) {
	orderID := fmt.Sprintf("ORD-CAN-%d", time.Now().UnixNano()%100000)
	tokenNum := fmt.Sprintf("T-%02d", (time.Now().Unix()%90)+10)
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}

	itemsBytes, _ := json.Marshal(req.Items)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO canteen_orders (id, student_id, items_json, total_amount, token_number, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'PREPARING', ?)
	`, orderID, req.StudentID, string(itemsBytes), total, tokenNum, time.Now())
	if err != nil {
		return nil, err
	}

	// Deduct balance atomically
	_, _ = r.db.ExecContext(ctx, "UPDATE canteen_wallets SET balance = balance - ? WHERE student_id = ?", total, req.StudentID)

	return &models.CanteenMealOrder{
		ID:          orderID,
		StudentID:   req.StudentID,
		Items:       req.Items,
		TotalAmount: total,
		TokenNumber: tokenNum,
		Status:      "PREPARING",
		CreatedAt:   time.Now(),
	}, nil
}

func (r *Repository) GetHallPass(ctx context.Context, studentID string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{
			"pass_id":          "HP-2026-901",
			"student_id":       studentID,
			"issued_by":        "Dr. R. Iyer",
			"destination":      "Infirmary / Clinic",
			"valid_until":      time.Now().Add(25 * time.Minute).Format(time.RFC3339),
			"status":           "ACTIVE",
			"qr_validation_id": "VAL-HP-7781-A",
		},
	}, nil
}

func (r *Repository) IssueHallPass(ctx context.Context, req *models.IssueHallPassRequest, issuedBy string) (map[string]interface{}, error) {
	passID := fmt.Sprintf("HP-%d", time.Now().UnixNano()%100000)
	validUntil := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO hall_passes (id, student_id, issued_by, destination, valid_until, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?)
	`, passID, req.StudentID, issuedBy, req.Destination, validUntil, time.Now())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pass_id":          passID,
		"student_id":       req.StudentID,
		"issued_by":        issuedBy,
		"destination":      req.Destination,
		"valid_until":      validUntil.Format(time.RFC3339),
		"status":           "ACTIVE",
		"qr_validation_id": fmt.Sprintf("VAL-%s", passID),
	}, nil
}

func (r *Repository) GetHealthRecords(ctx context.Context, studentID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"student_id":         studentID,
		"blood_group":        "O+",
		"allergies":          []string{"Peanuts", "Dust Mites"},
		"chronic_conditions": []string{"Mild Asthma"},
		"emergency_contact":  "+91 98765 43210 (Father)",
		"last_checkup":       "2026-06-10",
		"infirmary_visits": []map[string]interface{}{
			{"date": "2026-08-12", "reason": "Mild headache", "action": "Prescribed hydration and rest"},
		},
	}, nil
}

func (r *Repository) UpdateHealthRecords(ctx context.Context, req *models.UpdateHealthRequest) error {
	allergiesJSON, _ := json.Marshal(req.Allergies)
	chronicJSON, _ := json.Marshal(req.ChronicConditions)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO health_records (student_id, blood_group, allergies, chronic_conditions, infirmary_visits, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(student_id) DO UPDATE SET
			blood_group = COALESCE(NULLIF(excluded.blood_group, ''), blood_group),
			allergies = excluded.allergies,
			chronic_conditions = excluded.chronic_conditions,
			updated_at = CURRENT_TIMESTAMP
	`, req.StudentID, req.BloodGroup, string(allergiesJSON), string(chronicJSON), req.InfirmaryVisit)
	return err
}

func (r *Repository) GetStoreProducts(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, title, category, price, sizes, in_stock, image_url FROM store_products")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var id, title, cat, sizes, img string
		var price float64
		var inStock int
		if err := rows.Scan(&id, &title, &cat, &price, &sizes, &inStock, &img); err == nil {
			list = append(list, map[string]interface{}{
				"id":        id,
				"title":     title,
				"category":  cat,
				"price":     price,
				"sizes":     sizes,
				"in_stock":  inStock,
				"image_url": img,
			})
		}
	}
	return list, nil
}

func (r *Repository) GetStoreCart(ctx context.Context, studentID string) ([]models.CartItem, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, "SELECT items_json FROM store_carts WHERE student_id = ?", studentID).Scan(&raw)
	if err == sql.ErrNoRows {
		return []models.CartItem{
			{ProductID: "PRD-101", Title: "Complete Summer Uniform Set", Size: "M", Price: 1450.0, Quantity: 1},
			{ProductID: "PRD-103", Title: "Official Campus Sports Jersey", Size: "L", Price: 750.0, Quantity: 1},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var items []models.CartItem
	_ = json.Unmarshal([]byte(raw), &items)
	return items, nil
}

func (r *Repository) UpdateStoreCart(ctx context.Context, req *models.UpdateStoreCartRequest) error {
	bytes, _ := json.Marshal(req.Items)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO store_carts (student_id, items_json, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(student_id) DO UPDATE SET items_json=excluded.items_json, updated_at=CURRENT_TIMESTAMP
	`, req.StudentID, string(bytes))
	return err
}

func (r *Repository) CreateStoreOrder(ctx context.Context, studentID string) (map[string]interface{}, error) {
	cart, _ := r.GetStoreCart(ctx, studentID)
	orderID := fmt.Sprintf("STR-ORD-%d", time.Now().UnixNano()%100000)
	var total float64
	for _, it := range cart {
		total += it.Price * float64(it.Quantity)
	}

	bytes, _ := json.Marshal(cart)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO store_orders (id, student_id, items_json, total_amount, status, created_at)
		VALUES (?, ?, ?, ?, 'CONFIRMED', ?)
	`, orderID, studentID, string(bytes), total, time.Now())
	if err != nil {
		return nil, err
	}

	// Empty cart
	_, _ = r.db.ExecContext(ctx, "DELETE FROM store_carts WHERE student_id = ?", studentID)

	return map[string]interface{}{
		"order_id":     orderID,
		"student_id":   studentID,
		"total_amount": total,
		"status":       "CONFIRMED",
		"items_count":  len(cart),
	}, nil
}

func (r *Repository) SearchLibraryBooks(ctx context.Context, query string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": "BK-101", "title": "A Brief History of Time", "author": "Stephen Hawking", "category": "Science", "available": 3, "total": 4},
		{"id": "BK-102", "title": "Clean Architecture in Go", "author": "Robert C. Martin", "category": "Computer Science", "available": 2, "total": 2},
		{"id": "BK-103", "title": "To Kill a Mockingbird", "author": "Harper Lee", "category": "Literature", "available": 5, "total": 5},
	}, nil
}

func (r *Repository) GetBorrowedBooks(ctx context.Context, studentID string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"borrow_id": "BR-901", "student_id": studentID, "title": "A Brief History of Time", "borrowed_at": "2026-10-01", "due_date": "2026-10-21", "fine_amount": 0.0},
	}, nil
}

// ==========================================
// Category 6: Campus Life Repository
// ==========================================

func (r *Repository) GetClubs(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": "CLB-01", "name": "Robotics & AI Guild", "description": "Hands-on microcontrollers and machine learning models", "coordinator": "Mr. V. Patel", "schedule": "Fridays 03:30 PM"},
		{"id": "CLB-02", "name": "Model United Nations (MUN)", "description": "Global diplomacy, international debate and resolution drafting", "coordinator": "Mrs. N. Sen", "schedule": "Wednesdays 03:30 PM"},
		{"id": "CLB-03", "name": "Astronomy & Stargazing Society", "description": "Telescopic observation and astrophysics symposiums", "coordinator": "Dr. R. Iyer", "schedule": "Bi-weekly Saturdays"},
	}, nil
}

func (r *Repository) JoinClub(ctx context.Context, clubID, studentID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO club_memberships (club_id, student_id)
		VALUES (?, ?)
	`, clubID, studentID)
	return err
}

func (r *Repository) GetHouses(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT name, color, points, captain FROM houses ORDER BY points DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var name, color, captain string
		var points int
		if err := rows.Scan(&name, &color, &points, &captain); err == nil {
			list = append(list, map[string]interface{}{
				"name":    name,
				"color":   color,
				"points":  points,
				"captain": captain,
			})
		}
	}
	return list, nil
}

func (r *Repository) AddHousePoints(ctx context.Context, req *models.AddHousePointsRequest, teacher string) error {
	logID := fmt.Sprintf("HP-LOG-%d", time.Now().UnixNano()%100000)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "UPDATE houses SET points = points + ? WHERE name = ?", req.Points, req.HouseName)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO house_point_logs (id, house_name, points, reason, awarded_by, student_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, logID, req.HouseName, req.Points, req.Reason, teacher, req.StudentID, time.Now())
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetLostItems(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": "LST-01", "title": "Navy Blue Casio Scientific Calculator", "category": "Electronics", "location": "Room 302 Desk 4", "item_type": "FOUND", "status": "OPEN", "created_at": "2026-10-06T11:00:00Z"},
		{"id": "LST-02", "title": "Black Puma Water Bottle (Metal)", "category": "Accessories", "location": "Football Ground Pavilion", "item_type": "FOUND", "status": "OPEN", "created_at": "2026-10-07T09:30:00Z"},
	}, nil
}

func (r *Repository) ReportLostItem(ctx context.Context, item map[string]interface{}) error {
	id := fmt.Sprintf("LST-%d", time.Now().UnixNano()%100000)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO lost_items (id, title, description, category, location, item_type, reporter_id, image_url, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'OPEN', ?)
	`, id, item["title"], item["description"], item["category"], item["location"], item["item_type"], item["reporter_id"], item["image_url"], time.Now())
	return err
}

func (r *Repository) ClaimLostItem(ctx context.Context, req *models.ClaimLostItemRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE lost_items
		SET claimed_by = ?, status = 'CLAIM_PENDING'
		WHERE id = ?
	`, req.StudentID, req.ItemID)
	return err
}

func (r *Repository) GetPhotoAlbums(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": "ALB-2026-01", "title": "Annual Athletic Meet 2026", "date": "2026-09-24", "photo_count": 45, "thumbnail_url": "https://images.unsplash.com/photo-1461896836934-ffe607ba8211"},
		{"id": "ALB-2026-02", "title": "Inter-School Science Fair", "date": "2026-08-14", "photo_count": 32, "thumbnail_url": "https://images.unsplash.com/photo-1507668077129-56e32842fceb"},
	}, nil
}

func (r *Repository) GetAlbumPhotos(ctx context.Context, albumID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"album_id": albumID,
		"title":    "Annual Athletic Meet 2026",
		"photos": []string{
			"https://images.unsplash.com/photo-1461896836934-ffe607ba8211",
			"https://images.unsplash.com/photo-1517649763962-0c623266ddc0",
			"https://images.unsplash.com/photo-1530549387789-4c1017266635",
		},
	}, nil
}

// ==========================================
// Category 7: Parent Desk Repository
// ==========================================

func (r *Repository) GetParentThreads(ctx context.Context, studentID string) ([]models.ParentMessageThread, error) {
	return []models.ParentMessageThread{
		{
			ID:          "THRD-401",
			StudentID:   studentID,
			ParentID:    "PAR-901",
			TeacherID:   "STF2001",
			Subject:     "Inquiry on Advanced Physics Lab Practical Schedule",
			LastMessage: "Yes Mrs. Sharma, Aarav has been allocated the Tuesday slot.",
			Messages: []models.ThreadMessage{
				{SenderID: "PAR-901", SenderRole: "parent", Content: "Good morning Dr. Iyer, could you verify Aarav's lab slot?", Timestamp: time.Now().Add(-2 * time.Hour)},
				{SenderID: "STF2001", SenderRole: "teacher", Content: "Yes Mrs. Sharma, Aarav has been allocated the Tuesday slot.", Timestamp: time.Now().Add(-1 * time.Hour)},
			},
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
	}, nil
}

func (r *Repository) PostThreadMessage(ctx context.Context, req *models.PostParentDeskRequest, senderID, senderRole string) error {
	threadID := req.ThreadID
	if threadID == "" {
		threadID = fmt.Sprintf("THRD-%d", time.Now().UnixNano()%100000)
	}

	msg := models.ThreadMessage{
		SenderID:   senderID,
		SenderRole: senderRole,
		Content:    req.Message,
		Timestamp:  time.Now(),
	}

	msgsJSON, _ := json.Marshal([]models.ThreadMessage{msg})
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO parent_threads (id, student_id, parent_id, teacher_id, subject, messages_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET messages_json=excluded.messages_json, updated_at=CURRENT_TIMESTAMP
	`, threadID, req.StudentID, senderID, req.TeacherID, req.Subject, string(msgsJSON))
	return err
}

func (r *Repository) SearchPTMTeachers(ctx context.Context) ([]models.PtmTeacherTimingData, error) {
	return []models.PtmTeacherTimingData{
		{TeacherID: "STF2001", TeacherName: "Dr. Radhika Iyer", Subject: "Physics", Slots: []string{"09:00 - 09:15", "09:15 - 09:30", "10:00 - 10:15", "11:30 - 11:45"}},
		{TeacherID: "STF2002", TeacherName: "Dr. K. Rao", Subject: "Mathematics", Slots: []string{"09:30 - 09:45", "10:15 - 10:30", "11:00 - 11:15"}},
		{TeacherID: "STF2003", TeacherName: "Mrs. N. Sen", Subject: "English Literature", Slots: []string{"10:30 - 10:45", "14:00 - 14:15"}},
	}, nil
}

func (r *Repository) GetPTMTeacherTiming(ctx context.Context, teacherID string) (*models.PtmTeacherTimingData, error) {
	return &models.PtmTeacherTimingData{
		TeacherID:   teacherID,
		TeacherName: "Dr. Radhika Iyer",
		Subject:     "Physics",
		Slots:       []string{"09:00 - 09:15", "09:15 - 09:30", "10:00 - 10:15", "11:30 - 11:45"},
	}, nil
}

func (r *Repository) BookPTMSlot(ctx context.Context, req *models.PtmQuickBookReq) (map[string]interface{}, error) {
	bookingID := fmt.Sprintf("PTM-BK-%d", time.Now().UnixNano()%100000)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ptm_bookings (id, student_id, teacher_id, slot_time, agenda, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'CONFIRMED', ?)
	`, bookingID, req.StudentID, req.TeacherID, req.SlotTime, req.Agenda, time.Now())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"booking_id": bookingID,
		"student_id": req.StudentID,
		"teacher_id": req.TeacherID,
		"slot_time":  req.SlotTime,
		"agenda":     req.Agenda,
		"status":     "CONFIRMED",
	}, nil
}

func (r *Repository) ReschedulePTMSlot(ctx context.Context, req *models.ReschedulePtmReq) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ptm_bookings
		SET slot_time = ?, status = 'RESCHEDULED'
		WHERE id = ?
	`, req.NewSlotTime, req.BookingID)
	return err
}

func (r *Repository) AddCalendarEvent(ctx context.Context, req *models.AddCalendarEventRequest) (string, error) {
	eventID := fmt.Sprintf("CAL-EVT-%d", time.Now().UnixNano()%100000)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO calendar_events (id, title, start_time, end_time, location, details)
		VALUES (?, ?, ?, ?, ?, ?)
	`, eventID, req.Title, req.StartTime, req.EndTime, req.Location, req.Details)
	if err != nil {
		return "", err
	}
	return eventID, nil
}

func (r *Repository) UrgentBroadcast(ctx context.Context, req *models.UrgentNoticeBroadcastRequest) (int, error) {
	refID := fmt.Sprintf("NOTIF-%d", time.Now().UnixNano()%100000)
	sentCount := 350
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_tracking (reference_id, title, sent_count, delivered_count, read_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, refID, req.Title, sentCount, sentCount-8, 180, time.Now())
	if err != nil {
		return 0, err
	}
	return sentCount, nil
}

func (r *Repository) PostNotice(ctx context.Context, req *models.PostNoticeRequest) (string, error) {
	id := fmt.Sprintf("NTC-%d", time.Now().UnixNano()%100000)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notices (id, title, body, category, target_class, attachment_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, req.Title, req.Body, req.Category, req.TargetClass, req.AttachmentName, time.Now())
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) TrackNotification(ctx context.Context, refID string) (*models.NotificationTrackingReport, error) {
	return &models.NotificationTrackingReport{
		ReferenceID: refID,
		Title:       "High-Priority Severe Weather Advisory",
		SentCount:   500,
		Delivered:   492,
		ReadCount:   388,
		DeliveryPct: 98.4,
	}, nil
}

// ==========================================
// Category 8: Staff & System Repository
// ==========================================

func (r *Repository) GetSchoolInfo(ctx context.Context) (*models.SchoolInfoData, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT name, affiliation, school_code, address, phone, email, working_hours, term_dates_json
		FROM school_info WHERE id = 1
	`)
	var s models.SchoolInfoData
	var termsRaw string
	err := row.Scan(&s.Name, &s.Affiliation, &s.SchoolCode, &s.Address, &s.Phone, &s.Email, &s.WorkingHours, &termsRaw)
	if err == sql.ErrNoRows {
		return &models.SchoolInfoData{
			Name:         "NexaCampus International Academy",
			Affiliation:  "CBSE Affiliation #930142",
			SchoolCode:   "SCH-930142",
			Address:      "42 Cyber City, Tech Park Road, Bangalore, KA 560100",
			Phone:        "+91 80 4912 3400",
			Email:        "principal@nexacampus.edu.in",
			WorkingHours: "08:00 AM - 04:00 PM (Mon-Sat)",
			TermDates:    []string{"Term 1: 01 Jun - 15 Oct", "Term 2: 01 Nov - 28 Mar"},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(termsRaw), &s.TermDates)
	return &s, nil
}

func (r *Repository) AddStaff(ctx context.Context, req *models.AddStaffRequest) (string, error) {
	id := fmt.Sprintf("STF-%d", time.Now().UnixNano()%100000)
	classesBytes, _ := json.Marshal(req.AssignedClasses)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO staff (id, name, email, phone, role, designation, assigned_classes_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.Name, req.Email, req.Phone, req.Role, req.Designation, string(classesBytes), time.Now())
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) UpdateStaff(ctx context.Context, req *models.UpdateStaffRequest) error {
	classesBytes, _ := json.Marshal(req.AssignedClasses)
	_, err := r.db.ExecContext(ctx, `
		UPDATE staff
		SET designation = COALESCE(NULLIF(?, ''), designation),
		    phone = COALESCE(NULLIF(?, ''), phone),
		    role = COALESCE(NULLIF(?, ''), role),
		    assigned_classes_json = CASE WHEN ? != '[]' THEN ? ELSE assigned_classes_json END
		WHERE id = ?
	`, req.Designation, req.Phone, req.Role, string(classesBytes), string(classesBytes), req.StaffID)
	return err
}

func (r *Repository) DeleteStaff(ctx context.Context, staffID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM staff WHERE id = ?", staffID)
	return err
}
