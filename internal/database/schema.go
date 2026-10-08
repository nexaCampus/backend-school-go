package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitSchema ensures all necessary relational tables, indexes, and seed records
// exist in the PostgreSQL database for all 21 modules in the school portal.
func InitSchema(ctx context.Context, pool *pgxpool.Pool) error {
	ddl := `
	-- 1. Students & Profiles
	CREATE TABLE IF NOT EXISTS students (
		id TEXT PRIMARY KEY,
		student_id TEXT UNIQUE NOT NULL,
		dob TEXT NOT NULL,
		full_name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		grade TEXT NOT NULL,
		section TEXT NOT NULL,
		roll_no TEXT NOT NULL,
		blood_group TEXT DEFAULT '',
		father_name TEXT DEFAULT '',
		mother_name TEXT DEFAULT '',
		contact_no TEXT DEFAULT '',
		email TEXT DEFAULT '',
		address TEXT DEFAULT '',
		school_name TEXT DEFAULT '',
		academic_year TEXT DEFAULT '',
		avatar_url TEXT,
		smart_card_balance NUMERIC(10,2) DEFAULT 0,
		canteen_balance NUMERIC(10,2) DEFAULT 0,
		attendance_pct INT DEFAULT 0,
		streak_days INT DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT now(),
		updated_at TIMESTAMPTZ DEFAULT now()
	);

	ALTER TABLE students ADD COLUMN IF NOT EXISTS avatar_url TEXT;

	CREATE INDEX IF NOT EXISTS idx_students_student_id ON students(student_id);
	CREATE INDEX IF NOT EXISTS idx_students_grade_section ON students(grade, section);

	-- 2. Homework & Submissions
	CREATE TABLE IF NOT EXISTS homework (
		id TEXT PRIMARY KEY,
		grade TEXT NOT NULL,
		section TEXT NOT NULL,
		subject TEXT NOT NULL,
		unit TEXT DEFAULT '',
		title TEXT NOT NULL,
		teacher TEXT DEFAULT '',
		due TEXT DEFAULT '',
		due_date TIMESTAMPTZ,
		description TEXT DEFAULT '',
		attachment_name TEXT,
		attachment_meta TEXT,
		max_score NUMERIC(8,2) DEFAULT 100,
		created_at TIMESTAMPTZ DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_homework_grade_section ON homework(grade, section);

	CREATE TABLE IF NOT EXISTS homework_submissions (
		id TEXT PRIMARY KEY,
		homework_id TEXT NOT NULL REFERENCES homework(id) ON DELETE CASCADE,
		student_id TEXT NOT NULL,
		submission_file_url TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'submitted',
		obtained_score NUMERIC(8,2),
		feedback TEXT,
		submitted_at TIMESTAMPTZ DEFAULT now(),
		updated_at TIMESTAMPTZ DEFAULT now(),
		UNIQUE (homework_id, student_id)
	);

	CREATE INDEX IF NOT EXISTS idx_homework_sub_student ON homework_submissions(student_id);

	-- 3. Attendance & Leaves
	CREATE TABLE IF NOT EXISTS attendance (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		date DATE NOT NULL,
		status TEXT NOT NULL DEFAULT 'PRESENT',
		remarks TEXT DEFAULT '',
		UNIQUE (student_id, date)
	);

	CREATE INDEX IF NOT EXISTS idx_attendance_student_date ON attendance(student_id, date);

	CREATE TABLE IF NOT EXISTS leave_applications (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		leave_type TEXT NOT NULL,
		start_date DATE NOT NULL,
		end_date DATE NOT NULL,
		reason TEXT NOT NULL,
		attachment_url TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ DEFAULT now(),
		updated_at TIMESTAMPTZ DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_leave_student ON leave_applications(student_id);

	-- 4. Timetable
	CREATE TABLE IF NOT EXISTS timetable (
		id TEXT PRIMARY KEY,
		grade TEXT NOT NULL,
		section TEXT NOT NULL,
		day_of_week INT NOT NULL,
		period_label TEXT NOT NULL,
		time_label TEXT NOT NULL,
		subject TEXT NOT NULL,
		detail TEXT DEFAULT '',
		room_number TEXT DEFAULT '',
		teacher_name TEXT DEFAULT '',
		is_live BOOLEAN DEFAULT FALSE
	);

	CREATE INDEX IF NOT EXISTS idx_timetable_lookup ON timetable(grade, section, day_of_week);

	-- 5. Fees & Invoices
	CREATE TABLE IF NOT EXISTS fee_items (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		title TEXT NOT NULL,
		subtitle TEXT DEFAULT '',
		amount INT NOT NULL DEFAULT 0,
		due_date TEXT DEFAULT '',
		checked BOOLEAN DEFAULT TRUE,
		cleared BOOLEAN DEFAULT FALSE,
		cleared_meta TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_fee_items_student ON fee_items(student_id);

	CREATE TABLE IF NOT EXISTS fee_history (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		title TEXT NOT NULL,
		payment_date TEXT NOT NULL,
		amount INT NOT NULL DEFAULT 0,
		payment_ref TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_fee_history_student ON fee_history(student_id);

	-- 6. Academic Reports & Report Cards
	CREATE TABLE IF NOT EXISTS academic_reports (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		student_name TEXT NOT NULL,
		term TEXT NOT NULL,
		academic_year TEXT NOT NULL,
		gpa NUMERIC(4,2) DEFAULT 0.0,
		overall_grade TEXT NOT NULL,
		total_scored NUMERIC(8,2) DEFAULT 0,
		total_max NUMERIC(8,2) DEFAULT 500,
		rank INT DEFAULT 1,
		teacher_remarks TEXT DEFAULT '',
		marksheet_url TEXT DEFAULT '',
		subjects JSONB NOT NULL DEFAULT '[]'::jsonb
	);

	CREATE INDEX IF NOT EXISTS idx_academic_reports_student ON academic_reports(student_id, term);

	-- 7. Notices
	CREATE TABLE IF NOT EXISTS notices (
		id TEXT PRIMARY KEY,
		category TEXT NOT NULL DEFAULT 'General',
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		date_label TEXT DEFAULT '',
		author TEXT DEFAULT '',
		is_urgent BOOLEAN DEFAULT FALSE,
		attachment_name TEXT,
		attachment_url TEXT,
		published_at TIMESTAMPTZ DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_notices_published ON notices(published_at DESC);

	-- 8. Helpdesk
	CREATE TABLE IF NOT EXISTS helpdesk_tickets (
		id TEXT PRIMARY KEY,
		ticket_no TEXT NOT NULL,
		student_id TEXT NOT NULL,
		department TEXT NOT NULL,
		subject TEXT NOT NULL,
		message TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'SUBMITTED',
		preview TEXT DEFAULT '',
		attachment_url TEXT,
		resolution_notes TEXT,
		updated_label TEXT DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_helpdesk_student ON helpdesk_tickets(student_id);

	-- 9. Transport Routes & GPS
	CREATE TABLE IF NOT EXISTS transport_routes (
		id TEXT PRIMARY KEY,
		route_number TEXT UNIQUE NOT NULL,
		bus_number TEXT NOT NULL,
		driver_name TEXT NOT NULL,
		driver_phone TEXT NOT NULL,
		current_lat NUMERIC(10,6) DEFAULT 0,
		current_lng NUMERIC(10,6) DEFAULT 0,
		status TEXT DEFAULT 'ON_ROUTE',
		stops JSONB NOT NULL DEFAULT '[]'::jsonb
	);

	-- 10. Library Books, Borrowings & Holds
	CREATE TABLE IF NOT EXISTS library_books (
		id TEXT PRIMARY KEY,
		isbn TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		publisher TEXT DEFAULT '',
		total_copies INT NOT NULL DEFAULT 1,
		available_copies INT NOT NULL DEFAULT 1,
		category TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS library_borrowings (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		book_id TEXT NOT NULL REFERENCES library_books(id) ON DELETE CASCADE,
		borrowed_at TIMESTAMPTZ DEFAULT now(),
		due_at TIMESTAMPTZ DEFAULT (now() + INTERVAL '14 days'),
		returned_at TIMESTAMPTZ,
		fine_amount NUMERIC(8,2) DEFAULT 0,
		status TEXT DEFAULT 'BORROWED'
	);

	CREATE TABLE IF NOT EXISTS library_reservations (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		book_id TEXT NOT NULL,
		title TEXT NOT NULL,
		reserved_at TIMESTAMPTZ DEFAULT now(),
		expires_at TIMESTAMPTZ DEFAULT (now() + INTERVAL '48 hours'),
		status TEXT DEFAULT 'ACTIVE'
	);

	-- 11. Exam Hall Tickets
	CREATE TABLE IF NOT EXISTS exam_hall_tickets (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		student_name TEXT NOT NULL,
		grade TEXT NOT NULL,
		section TEXT NOT NULL,
		roll_no TEXT NOT NULL,
		term TEXT NOT NULL,
		exam_name TEXT NOT NULL,
		exam_center TEXT DEFAULT '',
		seat_number TEXT DEFAULT '',
		schedule JSONB NOT NULL DEFAULT '[]'::jsonb,
		rules JSONB NOT NULL DEFAULT '[]'::jsonb,
		is_active BOOLEAN DEFAULT TRUE
	);

	-- 12. Canteen Ledger & Menu
	CREATE TABLE IF NOT EXISTS canteen_transactions (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		title TEXT NOT NULL,
		amount NUMERIC(10,2) NOT NULL,
		transaction_type TEXT NOT NULL DEFAULT 'DEBIT',
		balance_after NUMERIC(10,2) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT now()
	);

	CREATE TABLE IF NOT EXISTS canteen_menu (
		id TEXT PRIMARY KEY,
		day_of_week TEXT NOT NULL,
		item_name TEXT NOT NULL,
		category TEXT NOT NULL,
		dietary_tag TEXT NOT NULL,
		price NUMERIC(8,2) NOT NULL,
		is_available BOOLEAN DEFAULT TRUE
	);

	-- 13. Parent-Teacher Meeting (PTM)
	CREATE TABLE IF NOT EXISTS ptm_sessions (
		id TEXT PRIMARY KEY,
		grade TEXT NOT NULL,
		title TEXT NOT NULL,
		date TEXT NOT NULL,
		time_range TEXT NOT NULL,
		faculty_name TEXT NOT NULL,
		location TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS ptm_slots (
		id TEXT PRIMARY KEY,
		teacher_id TEXT NOT NULL,
		teacher_name TEXT NOT NULL,
		session_date TEXT NOT NULL,
		time_slot TEXT NOT NULL,
		is_booked BOOLEAN DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS ptm_bookings (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		slot_id TEXT NOT NULL,
		teacher_name TEXT NOT NULL,
		date TEXT NOT NULL,
		time_slot TEXT NOT NULL,
		agenda TEXT DEFAULT '',
		meeting_link TEXT DEFAULT '',
		status TEXT DEFAULT 'CONFIRMED',
		created_at TIMESTAMPTZ DEFAULT now()
	);

	-- 14. Digital Hall Pass
	CREATE TABLE IF NOT EXISTS hall_passes (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		destination TEXT NOT NULL,
		duration_minutes INT DEFAULT 10,
		teacher_name TEXT DEFAULT '',
		status TEXT DEFAULT 'ACTIVE',
		verification_qr TEXT NOT NULL,
		issued_at TIMESTAMPTZ DEFAULT now(),
		expires_at TIMESTAMPTZ NOT NULL
	);

	-- 15. Infirmary & Student Health
	CREATE TABLE IF NOT EXISTS health_profiles (
		student_id TEXT PRIMARY KEY,
		blood_group TEXT DEFAULT 'A+ve',
		allergies TEXT[] DEFAULT '{}',
		conditions TEXT[] DEFAULT '{}',
		emergency_contact_name TEXT DEFAULT '',
		emergency_contact_phone TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS clinic_visits (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		visit_date TIMESTAMPTZ DEFAULT now(),
		temperature TEXT DEFAULT '98.6°F',
		complaint TEXT NOT NULL,
		treatment TEXT NOT NULL,
		first_aid TEXT DEFAULT '',
		nurse_name TEXT DEFAULT 'Nurse Martha'
	);

	CREATE TABLE IF NOT EXISTS medical_consents (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		medication_name TEXT NOT NULL,
		prescription_url TEXT DEFAULT '',
		status TEXT DEFAULT 'AUTHORIZED',
		created_at TIMESTAMPTZ DEFAULT now()
	);

	-- 16. House Merits & Discipline
	CREATE TABLE IF NOT EXISTS house_memberships (
		student_id TEXT PRIMARY KEY,
		house_name TEXT NOT NULL,
		house_color TEXT NOT NULL,
		individual_points INT DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS house_leaderboard (
		house_name TEXT PRIMARY KEY,
		house_color TEXT NOT NULL,
		total_points INT DEFAULT 0,
		rank INT DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS merit_records (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		points INT DEFAULT 0,
		remarks TEXT DEFAULT '',
		awarded_by TEXT DEFAULT '',
		date TIMESTAMPTZ DEFAULT now()
	);

	-- 17. School Store
	CREATE TABLE IF NOT EXISTS store_products (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		category TEXT NOT NULL,
		price NUMERIC(10,2) NOT NULL,
		sizes TEXT[] DEFAULT '{}',
		image_url TEXT DEFAULT '',
		in_stock BOOLEAN DEFAULT TRUE,
		stock_quantity INT DEFAULT 100
	);

	CREATE TABLE IF NOT EXISTS store_orders (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		items JSONB NOT NULL DEFAULT '[]'::jsonb,
		total_amount NUMERIC(10,2) NOT NULL,
		delivery_option TEXT DEFAULT 'locker',
		status TEXT DEFAULT 'PRE_ORDERED',
		created_at TIMESTAMPTZ DEFAULT now()
	);

	-- 18. Campus Lost & Found
	CREATE TABLE IF NOT EXISTS lost_found_items (
		id TEXT PRIMARY KEY,
		student_id TEXT,
		item_name TEXT NOT NULL,
		last_seen_location TEXT NOT NULL,
		image_url TEXT,
		status TEXT DEFAULT 'found',
		created_at TIMESTAMPTZ DEFAULT now()
	);

	-- 19. Event Gallery & Yearbook
	CREATE TABLE IF NOT EXISTS gallery_albums (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		year TEXT NOT NULL,
		cover_url TEXT NOT NULL,
		photo_count INT DEFAULT 0,
		date TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS gallery_photos (
		id TEXT PRIMARY KEY,
		album_id TEXT NOT NULL REFERENCES gallery_albums(id) ON DELETE CASCADE,
		url TEXT NOT NULL,
		caption TEXT DEFAULT ''
	);

	-- 20. Clubs & Competitions
	CREATE TABLE IF NOT EXISTS clubs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		faculty_advisor TEXT DEFAULT '',
		meeting_schedule TEXT DEFAULT '',
		banner_url TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS student_clubs (
		student_id TEXT NOT NULL,
		club_id TEXT NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
		role TEXT DEFAULT 'MEMBER',
		PRIMARY KEY (student_id, club_id)
	);

	CREATE TABLE IF NOT EXISTS club_competitions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		grade TEXT DEFAULT 'All',
		description TEXT NOT NULL,
		deadline TEXT NOT NULL,
		registration_link TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS competition_registrations (
		id TEXT PRIMARY KEY,
		competition_id TEXT NOT NULL,
		student_id TEXT NOT NULL,
		status TEXT DEFAULT 'REGISTERED',
		registered_at TIMESTAMPTZ DEFAULT now(),
		UNIQUE (competition_id, student_id)
	);

	-- Enable Row-Level Security (RLS) on all tables
	ALTER TABLE IF EXISTS students ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS homework ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS homework_submissions ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS attendance ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS leave_applications ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS timetable ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS fee_items ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS fee_history ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS notices ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS helpdesk_tickets ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS bus_routes ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS bus_live_locations ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS library_books ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS library_borrowings ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS library_reservations ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS exam_hall_tickets ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS canteen_menu ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS canteen_transactions ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS ptm_sessions ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS ptm_slots ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS digital_hall_passes ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS health_profiles ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS clinic_visits ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS medical_consents ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS student_merits ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS store_products ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS store_orders ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS lost_found_items ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS gallery_albums ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS gallery_photos ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS clubs ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS student_clubs ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS club_competitions ENABLE ROW LEVEL SECURITY;
	ALTER TABLE IF EXISTS competition_registrations ENABLE ROW LEVEL SECURITY;
	`

	_, err := pool.Exec(ctx, ddl)
	if err != nil {
		return err
	}

	seedDefaultData(ctx, pool)
	return nil
}

func seedDefaultData(ctx context.Context, pool *pgxpool.Pool) {
	var count int
	err := pool.QueryRow(ctx, "SELECT count(*) FROM students").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	log.Println("[INFO] Seeding complete default sample dataset for all 21 modules...")

	// 1. Seed Student Profile
	_, _ = pool.Exec(ctx, `
		INSERT INTO students (
			id, student_id, dob, full_name, display_name, grade, section, roll_no,
			blood_group, father_name, mother_name, contact_no, email, address,
			school_name, academic_year, avatar_url, smart_card_balance, canteen_balance, attendance_pct, streak_days
		) VALUES (
			'stu-10211125', '10211125', '2007-02-10', 'RAJEN SHAW', 'Rajen Shaw', '12', 'E', '20',
			'A+ve', 'Krishna Shaw', 'Geeta Shaw', '917980679513', 'krishnashaw2005@gmail.com',
			'SUBHASGRAM, MANASATALA PARK CHAK, HARINAVI KOL-147', 'Lions Calcutta Gr Vidya Mandir',
			'2026-2027', 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&q=80&w=400',
			1250.00, 450.00, 96, 14
		) ON CONFLICT (student_id) DO NOTHING;
	`)

	// 2. Timetable
	_, _ = pool.Exec(ctx, `
		INSERT INTO timetable (id, grade, section, day_of_week, period_label, time_label, subject, detail, room_number, teacher_name, is_live)
		VALUES
		('tt-1', '12', 'E', 1, 'Period 1', '08:30 - 09:30 AM', 'Physics Lab (Optics & Laser)', 'Lab 3 • Dr. Aris Thorne', 'Lab 3', 'Dr. Aris Thorne', true),
		('tt-2', '12', 'E', 1, 'Period 2', '09:40 - 10:35 AM', 'Mathematics II', 'Calculus • Room 402', 'Room 402', 'Prof. Banerjee', false),
		('tt-3', '12', 'E', 1, 'Period 3', '10:50 - 11:45 AM', 'Chemistry Organic', 'Seminar Hall B', 'Hall B', 'Dr. Roy', false),
		('tt-4', '12', 'E', 1, 'Period 4', '12:00 - 12:55 PM', 'English Literature', 'The Portrait of a Lady', 'Room 402', 'Mrs. Sen', false),
		('tt-5', '12', 'E', 1, 'Period 5', '01:30 - 02:25 PM', 'Computer Science', 'Python OOP & Data Structures', 'Computer Lab 2', 'Mr. Paul', false)
		ON CONFLICT (id) DO NOTHING;
	`)

	// 3. Homework
	_, _ = pool.Exec(ctx, `
		INSERT INTO homework (id, grade, section, subject, unit, title, teacher, due, due_date, description, attachment_name, attachment_meta, max_score)
		VALUES
		('hw-maths-73', '12', 'E', 'Mathematics II', 'Ex 7.3', 'Calculus Problem Set 4', 'Dr. Aris Thorne', 'Due Tomorrow, 11:59 PM', now() + INTERVAL '1 day', 'Complete exercise 7.3 problems 1-15 on differential equations. Attach PDF scan of handwritten notebook solutions showing all substitution steps clearly.', 'Worksheet (2.4 MB)', 'PDF', 100),
		('hw-phy-08', '12', 'E', 'Physics Lab', 'Practical #08', 'Optics Experiment Report', 'Prof. Mukherjee', 'Due: Friday, 26 Sep', now() + INTERVAL '3 days', 'Ray diagram observations from Tuesday lab session. Submit lab manual graphs depicting angle of minimum deviation for prism index calculations.', 'Ray_Diagram_Shaw_12E.pdf', 'Uploaded Sep 23, 4:15 PM', 100),
		('hw-chem-04', '12', 'E', 'Organic Chemistry', 'Unit 4', 'Reaction Mechanism Notes', 'Dr. Roy', 'Due: Next Monday (29 Sep)', now() + INTERVAL '5 days', 'Prepare comparative summary on SN1 vs SN2 nucleophilic substitution kinetics.', NULL, NULL, 50)
		ON CONFLICT (id) DO NOTHING;
	`)

	// 4. Attendance
	_, _ = pool.Exec(ctx, `
		INSERT INTO attendance (id, student_id, date, status, remarks)
		VALUES
		('att-1', '10211125', '2026-10-01', 'PRESENT', 'On time'),
		('att-2', '10211125', '2026-10-02', 'HOLIDAY', 'Gandhi Jayanti'),
		('att-3', '10211125', '2026-10-03', 'PRESENT', 'Full attendance'),
		('att-4', '10211125', '2026-10-04', 'SUNDAY', 'Weekend'),
		('att-5', '10211125', '2026-10-05', 'PRESENT', 'On time')
		ON CONFLICT (student_id, date) DO NOTHING;
	`)

	// 5. Notices
	_, _ = pool.Exec(ctx, `
		INSERT INTO notices (id, category, title, body, date_label, author, is_urgent, attachment_name, published_at)
		VALUES
		('n1', 'Urgent', 'Resumption of Regular Classes & Annual Sports Day Registration',
		'All secondary & senior secondary cohorts (Grades 9 to 12) shall resume in-person curriculum schedules starting this Wednesday.',
		'24 Sep 2026 • 10:30 AM', 'Office of the Principal', true, NULL, now() - INTERVAL '1 day'),
		('n2', 'Academic', 'Parent-Teacher Meeting (PTM) Schedule for Class 12',
		'Dear Parents & Guardians, please find attached the slot timings and venue map for Term 1 Parent-Teacher Review.',
		'24 Sep 2026 • 2:23 PM', 'Biswarup Paul (Academic Coordinator)', false, 'PTM_Schedule_2026.pdf', now() - INTERVAL '2 days')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 6. Fees
	_, _ = pool.Exec(ctx, `
		INSERT INTO fee_items (id, student_id, title, subtitle, amount, due_date, checked, cleared, cleared_meta)
		VALUES
		('fee-1', '10211125', 'Tuition Fee (Quarterly)', 'Mandatory Academic Instruction', 3100, 'Oct 30, 2026', true, false, NULL),
		('fee-2', '10211125', 'STEM Robotics Lab Fee', 'Material & Equipment Maintenance', 250, 'Oct 30, 2026', true, false, NULL),
		('fee-3', '10211125', 'Computer & Digital Library', 'E-Portal License + Campus Wi-Fi', 400, 'Oct 30, 2026', true, false, NULL)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO fee_history (id, student_id, title, payment_date, amount, payment_ref)
		VALUES
		('fh-1', '10211125', 'Term 1 Tuition', 'Jul 14, 2026', 4200, 'Ref #88219')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 7. Academic Report Card
	_, _ = pool.Exec(ctx, `
		INSERT INTO academic_reports (
			id, student_id, student_name, term, academic_year, gpa, overall_grade,
			total_scored, total_max, rank, teacher_remarks, marksheet_url, subjects
		) VALUES (
			'rep-1', '10211125', 'RAJEN SHAW', 'Term 1', '2026-2027', 3.92, 'A+',
			472.0, 500.0, 2, 'Exemplary analytical aptitude in STEM sciences with consistent lab coursework.',
			'https://storage.school.edu/reports/Term1_ReportCard_10211125.pdf',
			'[
				{"subject": "Physics", "score": 96, "total": 100, "grade": "A+", "remarks": "Top 2%"},
				{"subject": "Mathematics II", "score": 98, "total": 100, "grade": "A+", "remarks": "Top 1%"},
				{"subject": "Computer Science", "score": 99, "total": 100, "grade": "A+", "remarks": "Rank 1"},
				{"subject": "Chemistry", "score": 91, "total": 100, "grade": "A", "remarks": "Distinction"},
				{"subject": "English Core", "score": 88, "total": 100, "grade": "A", "remarks": "Very Good"}
			]'::jsonb
		) ON CONFLICT (id) DO NOTHING;
	`)

	// 8. Helpdesk
	_, _ = pool.Exec(ctx, `
		INSERT INTO helpdesk_tickets (id, ticket_no, student_id, department, subject, message, preview, status, updated_label, created_at)
		VALUES
		('tkt-1', 'TKT-8841', '10211125', 'Academic Doubt', 'Optical Bench Experiment #4',
		'Should we attach handwritten ray diagrams on interleaved sheets or ruled page?',
		'Should we attach handwritten ray diagrams on interleaved sheets...', 'IN_REVIEW', 'Expected reply by 4:00 PM tomorrow', now() - INTERVAL '1 day')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 9. Transport Route 14
	_, _ = pool.Exec(ctx, `
		INSERT INTO transport_routes (id, route_number, bus_number, driver_name, driver_phone, current_lat, current_lng, status, stops)
		VALUES (
			'tr-14', '14', 'WB-04-E-8812', 'Mr. Rameshwar Yadav', '+91 98301 44219', 22.438510, 88.397020, 'ON_ROUTE',
			'[
				{"name": "Subhasgram Junction", "time": "07:00 AM", "lat": 22.4278, "lng": 88.4215},
				{"name": "Harinavi Market", "time": "07:15 AM", "lat": 22.4412, "lng": 88.4118},
				{"name": "Garia Metro Station", "time": "07:35 AM", "lat": 22.4645, "lng": 88.3912},
				{"name": "Lions Calcutta Campus Gate", "time": "08:15 AM", "lat": 22.5210, "lng": 88.3650}
			]'::jsonb
		) ON CONFLICT (route_number) DO NOTHING;
	`)

	// 10. Library
	_, _ = pool.Exec(ctx, `
		INSERT INTO library_books (id, isbn, title, author, publisher, total_copies, available_copies, category)
		VALUES
		('bk-1', '978-0131103627', 'Concepts of Physics (Vol 1 & 2)', 'Dr. H.C. Verma', 'Bharati Bhawan', 12, 4, 'Physics'),
		('bk-2', '978-0199147175', 'Problems in General Physics', 'I.E. Irodov', 'Mir Publishers', 8, 2, 'Physics'),
		('bk-3', '978-0070634152', 'Calculus and Analytic Geometry', 'George B. Thomas', 'Pearson', 10, 5, 'Mathematics'),
		('bk-4', '978-0134093413', 'Computer Science with Python 12', 'Sumita Arora', 'Dhanpat Rai', 15, 7, 'Computer Science')
		ON CONFLICT (isbn) DO NOTHING;

		INSERT INTO library_borrowings (id, student_id, book_id, borrowed_at, due_at, fine_amount, status)
		VALUES
		('bor-1', '10211125', 'bk-1', now() - INTERVAL '8 days', now() + INTERVAL '6 days', 0.00, 'BORROWED')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 11. Exam Hall Tickets
	_, _ = pool.Exec(ctx, `
		INSERT INTO exam_hall_tickets (
			id, student_id, student_name, grade, section, roll_no, term,
			exam_name, exam_center, seat_number, schedule, rules, is_active
		) VALUES (
			'ht-1', '10211125', 'RAJEN SHAW', '12', 'E', '20', 'Term 1',
			'Senior Secondary Mid-Term Examination 2026', 'Auditorium Hall C, Lions Campus', 'DESK-C20',
			'[
				{"subject": "Physics Theory", "date": "2026-10-14", "time": "09:00 AM - 12:00 PM", "room": "Hall C - Row 2"},
				{"subject": "Mathematics II", "date": "2026-10-16", "time": "09:00 AM - 12:00 PM", "room": "Hall C - Row 2"},
				{"subject": "Computer Science", "date": "2026-10-21", "time": "09:00 AM - 12:00 PM", "room": "Lab 2"}
			]'::jsonb,
			'["Candidates must carry physical printed copy of this pass and Smart Card ID."]'::jsonb,
			true
		) ON CONFLICT (id) DO NOTHING;
	`)

	// 12. Canteen Menu & Transactions
	_, _ = pool.Exec(ctx, `
		INSERT INTO canteen_transactions (id, student_id, title, amount, transaction_type, balance_after, created_at)
		VALUES
		('ctx-1', '10211125', 'Smart Card Wallet Reload (UPI)', 500.00, 'CREDIT', 520.00, now() - INTERVAL '3 days'),
		('ctx-2', '10211125', 'Cafeteria Lunch Combo Meal', 70.00, 'DEBIT', 450.00, now() - INTERVAL '1 day')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO canteen_menu (id, day_of_week, item_name, category, dietary_tag, price, is_available)
		VALUES
		('m-1', 'All', 'Veg Thali Deluxe', 'Lunch', 'Veg', 70.00, true),
		('m-2', 'All', 'Paneer Kathi Roll', 'Snacks', 'Veg', 45.00, true),
		('m-3', 'All', 'South Indian Idli Sambar', 'Breakfast', 'Veg', 35.00, true),
		('m-4', 'All', 'Fresh Cold Pressed Juice', 'Beverages', 'Vegan', 30.00, true)
		ON CONFLICT (id) DO NOTHING;
	`)

	// 13. PTM Sessions & Slots
	_, _ = pool.Exec(ctx, `
		INSERT INTO ptm_sessions (id, grade, title, date, time_range, faculty_name, location)
		VALUES
		('ptm-s1', '12', 'Term 1 Mid-Year Parent-Teacher Conference', '2026-10-24', '09:00 AM - 03:00 PM', 'Senior Secondary Faculty', 'Block C Auditorium & Online')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO ptm_slots (id, teacher_id, teacher_name, session_date, time_slot, is_booked)
		VALUES
		('slot-1', 't-thorne', 'Dr. Aris Thorne (Physics)', '2026-10-24', '09:30 AM - 09:45 AM', false),
		('slot-2', 't-thorne', 'Dr. Aris Thorne (Physics)', '2026-10-24', '10:00 AM - 10:15 AM', false),
		('slot-3', 't-banerjee', 'Prof. Banerjee (Maths)', '2026-10-24', '10:30 AM - 10:45 AM', false),
		('slot-4', 't-paul', 'Mr. Paul (Computer Science)', '2026-10-24', '11:00 AM - 11:15 AM', false)
		ON CONFLICT (id) DO NOTHING;
	`)

	// 14. Infirmary & Health
	_, _ = pool.Exec(ctx, `
		INSERT INTO health_profiles (student_id, blood_group, allergies, conditions, emergency_contact_name, emergency_contact_phone)
		VALUES ('10211125', 'A+ve', '{"Peanuts (Mild)", "Dust"}', '{"None"}', 'Krishna Shaw', '917980679513')
		ON CONFLICT (student_id) DO NOTHING;

		INSERT INTO clinic_visits (id, student_id, visit_date, temperature, complaint, treatment, first_aid, nurse_name)
		VALUES
		('cv-1', '10211125', now() - INTERVAL '12 days', '99.1°F', 'Mild headache during sports period', 'Administered Paracetamol & 20 min rest', 'Rest & Rehydration', 'Nurse Martha')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 15. House Merits
	_, _ = pool.Exec(ctx, `
		INSERT INTO house_memberships (student_id, house_name, house_color, individual_points)
		VALUES ('10211125', 'Phoenix Red', '#EF4444', 185)
		ON CONFLICT (student_id) DO NOTHING;

		INSERT INTO house_leaderboard (house_name, house_color, total_points, rank)
		VALUES
		('Phoenix Red', '#EF4444', 1420, 1),
		('Pegasus Blue', '#2563EB', 1380, 2),
		('Hydra Green', '#10B981', 1290, 3),
		('Leo Gold', '#F59E0B', 1210, 4)
		ON CONFLICT (house_name) DO NOTHING;

		INSERT INTO merit_records (id, student_id, type, title, points, remarks, awarded_by, date)
		VALUES
		('mr-1', '10211125', 'MERIT', 'First Place - Science Olympiad Regional', 50, 'Outstanding analytical accuracy', 'Dr. Aris Thorne', now() - INTERVAL '5 days'),
		('mr-2', '10211125', 'MERIT', 'Peer Tutoring Excellence in Calculus', 25, 'Helped Grade 11 cohort with integration', 'Prof. Banerjee', now() - INTERVAL '14 days')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 16. School Store
	_, _ = pool.Exec(ctx, `
		INSERT INTO store_products (id, title, category, price, sizes, image_url, in_stock, stock_quantity)
		VALUES
		('sp-1', 'Senior Blazer with School Emblem', 'uniform', 1450.00, '{"36", "38", "40", "42"}', 'https://images.unsplash.com/photo-1594938298603-c8148c4dae35?auto=format&fit=crop&q=80&w=300', true, 40),
		('sp-2', 'Physical Education Track Jersey', 'uniform', 450.00, '{"S", "M", "L", "XL"}', 'https://images.unsplash.com/photo-1576566588028-4147f3842f27?auto=format&fit=crop&q=80&w=300', true, 80),
		('sp-3', 'Hardbound Graph & Laboratory Journal', 'stationery', 120.00, '{"Standard"}', 'https://images.unsplash.com/photo-1589829085413-56de8ae18c73?auto=format&fit=crop&q=80&w=300', true, 150)
		ON CONFLICT (id) DO NOTHING;
	`)

	// 17. Lost & Found
	_, _ = pool.Exec(ctx, `
		INSERT INTO lost_found_items (id, student_id, item_name, last_seen_location, image_url, status, created_at)
		VALUES
		('lf-1', NULL, 'Black Casio Scientific Calculator fx-991EX', 'Physics Lab 3 Desk 4', 'https://images.unsplash.com/photo-1594980596870-8aa52a78d8cd?auto=format&fit=crop&q=80&w=300', 'found', now() - INTERVAL '2 days'),
		('lf-2', NULL, 'Milton Steel Water Bottle (Blue Cap)', 'Football Ground Pavilion', NULL, 'found', now() - INTERVAL '1 day')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 18. Gallery Albums & Photos
	_, _ = pool.Exec(ctx, `
		INSERT INTO gallery_albums (id, title, year, cover_url, photo_count, date)
		VALUES
		('alb-1', 'Annual Inter-House Athletics Meet 2026', '2026', 'https://images.unsplash.com/photo-1461896836934-ffe607ba8211?auto=format&fit=crop&q=80&w=400', 36, 'Sep 2026'),
		('alb-2', 'STEM Robotics Exhibition & Code Fest', '2026', 'https://images.unsplash.com/photo-1485827404703-89b55fcc595e?auto=format&fit=crop&q=80&w=400', 28, 'Aug 2026')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO gallery_photos (id, album_id, url, caption)
		VALUES
		('gp-1', 'alb-1', 'https://images.unsplash.com/photo-1461896836934-ffe607ba8211?auto=format&fit=crop&q=80&w=800', '100m Sprint Finals Senior Boys'),
		('gp-2', 'alb-1', 'https://images.unsplash.com/photo-1517649763962-0c623266ddc0?auto=format&fit=crop&q=80&w=800', 'Trophy Ceremony')
		ON CONFLICT (id) DO NOTHING;
	`)

	// 19. Clubs & Competitions
	_, _ = pool.Exec(ctx, `
		INSERT INTO clubs (id, name, description, faculty_advisor, meeting_schedule, banner_url)
		VALUES
		('cl-1', 'Robotics & Automation Society', 'Building IoT bots, drone navigation, and clean energy prototypes.', 'Dr. Aris Thorne', 'Every Wednesday 3:30 PM • Robotics Lab', 'https://images.unsplash.com/photo-1485827404703-89b55fcc595e?auto=format&fit=crop&q=80&w=400'),
		('cl-2', 'Coding & Algorithmic League', 'Competitive programming in Go and Python, web development, and cyber security.', 'Mr. Biswarup Paul', 'Every Friday 3:30 PM • Computer Lab 2', 'https://images.unsplash.com/photo-1517694712202-14dd9538aa97?auto=format&fit=crop&q=80&w=400')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO student_clubs (student_id, club_id, role)
		VALUES
		('10211125', 'cl-1', 'LEAD'),
		('10211125', 'cl-2', 'MEMBER')
		ON CONFLICT (student_id, club_id) DO NOTHING;

		INSERT INTO club_competitions (id, title, grade, description, deadline, registration_link)
		VALUES
		('cmp-1', 'National Youth Cyber Olympiad 2026', '12', 'Inter-school algorithmic problem solving and reverse engineering challenge.', 'Oct 28, 2026', 'https://olympiad.edu/register/nyco2026'),
		('cmp-2', 'State Clean Energy Model Hackathon', '12', 'Hardware prototype design competition focused on sustainable campus grid.', 'Nov 05, 2026', 'https://statehackathon.org/apply')
		ON CONFLICT (id) DO NOTHING;
	`)

	log.Println("[INFO] Default sample dataset seeded successfully.")
}
