package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var globalSQLite *sql.DB

// SQLitePragmas define performance-critical tuning for high-throughput concurrency.
var SQLitePragmas = []string{
	"PRAGMA journal_mode = WAL;",
	"PRAGMA synchronous = NORMAL;",
	"PRAGMA cache_size = -64000;",   // ~64 MB cache
	"PRAGMA mmap_size = 268435456;", // 256 MB memory-mapped I/O
	"PRAGMA temp_store = MEMORY;",
	"PRAGMA busy_timeout = 5000;",
	"PRAGMA foreign_keys = ON;",
}

// InitSQLite initializes modernc.org/sqlite database with WAL pragmas and creates tables.
func InitSQLite(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = "nexacampus.db"
	}

	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for sqlite: %w", err)
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize connection pooling for WAL mode
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite ping failed: %w", err)
	}

	// Execute WAL and concurrency PRAGMAs
	for _, pragma := range SQLitePragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			log.Printf("[WARN] Failed executing pragma %q: %v", pragma, err)
		}
	}

	log.Printf("[INFO] SQLite initialized with WAL mode at: %s", dbPath)

	if err := initSQLiteTables(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}

	globalSQLite = db
	return db, nil
}

// GetSQLite returns the global SQLite connection pool.
func GetSQLite() *sql.DB {
	return globalSQLite
}

func initSQLiteTables(ctx context.Context, db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS students (
		student_id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		class_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		roll_no TEXT NOT NULL,
		admission_id TEXT DEFAULT '',
		dob TEXT NOT NULL,
		father_name TEXT DEFAULT '',
		mother_name TEXT DEFAULT '',
		blood_group TEXT DEFAULT '',
		mobile TEXT DEFAULT '',
		email TEXT DEFAULT '',
		address TEXT DEFAULT '',
		profile_photo_url TEXT DEFAULT '',
		medical_notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS parent_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id TEXT NOT NULL,
		parent_id TEXT NOT NULL,
		UNIQUE(student_id, parent_id)
	);

	CREATE TABLE IF NOT EXISTS timetables (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		day_of_week TEXT NOT NULL,
		period_no INTEGER NOT NULL,
		subject TEXT NOT NULL,
		teacher TEXT NOT NULL,
		room TEXT NOT NULL,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		is_break INTEGER DEFAULT 0,
		UNIQUE(class_id, section_id, day_of_week, period_no)
	);

	CREATE TABLE IF NOT EXISTS lab_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lab_name TEXT NOT NULL,
		subject TEXT NOT NULL,
		teacher TEXT NOT NULL,
		room TEXT NOT NULL,
		day_of_week TEXT NOT NULL,
		timing TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS syllabi (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id TEXT NOT NULL,
		subject TEXT NOT NULL,
		chapter_no INTEGER NOT NULL,
		title TEXT NOT NULL,
		completed INTEGER DEFAULT 0,
		target_date TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS diary_remarks (
		id TEXT PRIMARY KEY,
		class_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		student_id TEXT DEFAULT '',
		teacher TEXT NOT NULL,
		remark TEXT NOT NULL,
		conduct TEXT DEFAULT 'Neutral',
		date TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS homework_assignments (
		id TEXT PRIMARY KEY,
		class_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		subject TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		teacher TEXT NOT NULL,
		due_date TEXT NOT NULL,
		attachment_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS homework_submissions (
		homework_id TEXT NOT NULL,
		student_id TEXT NOT NULL,
		file_url TEXT NOT NULL,
		obtained_score REAL DEFAULT 0,
		submitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(homework_id, student_id)
	);

	CREATE TABLE IF NOT EXISTS attendance_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id TEXT NOT NULL,
		class_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		date TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(student_id, date)
	);

	CREATE TABLE IF NOT EXISTS leave_requests (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		leave_type TEXT NOT NULL,
		start_date TEXT NOT NULL,
		end_date TEXT NOT NULL,
		reason TEXT NOT NULL,
		attachment_url TEXT DEFAULT '',
		status TEXT DEFAULT 'PENDING',
		review_remarks TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS exam_schedules (
		id TEXT PRIMARY KEY,
		class_id TEXT NOT NULL,
		exam_name TEXT NOT NULL,
		subject TEXT NOT NULL,
		date TEXT NOT NULL,
		shift TEXT NOT NULL,
		room TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS exam_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id TEXT NOT NULL,
		exam_name TEXT NOT NULL,
		subject TEXT NOT NULL,
		marks_scored REAL NOT NULL,
		total_marks REAL NOT NULL,
		percentile REAL DEFAULT 0,
		grade TEXT DEFAULT '',
		comments TEXT DEFAULT '',
		UNIQUE(student_id, exam_name, subject)
	);

	CREATE TABLE IF NOT EXISTS payments (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		order_id TEXT NOT NULL,
		payment_id TEXT DEFAULT '',
		amount REAL NOT NULL,
		currency TEXT DEFAULT 'INR',
		status TEXT NOT NULL,
		purpose TEXT DEFAULT '',
		receipt_ref TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tax_receipts (
		receipt_no TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		payment_id TEXT NOT NULL,
		gstin TEXT NOT NULL,
		school_gstin TEXT NOT NULL,
		taxable_amount REAL NOT NULL,
		cgst REAL NOT NULL,
		sgst REAL NOT NULL,
		igst REAL NOT NULL,
		total_amount REAL NOT NULL,
		date TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS canteen_wallets (
		student_id TEXT PRIMARY KEY,
		balance REAL DEFAULT 0.0,
		daily_limit REAL DEFAULT 500.0,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS canteen_orders (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		items_json TEXT NOT NULL,
		total_amount REAL NOT NULL,
		token_number TEXT NOT NULL,
		status TEXT DEFAULT 'PREPARING',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS hall_passes (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		issued_by TEXT NOT NULL,
		destination TEXT NOT NULL,
		valid_until DATETIME NOT NULL,
		status TEXT DEFAULT 'ACTIVE',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS health_records (
		student_id TEXT PRIMARY KEY,
		blood_group TEXT DEFAULT '',
		allergies TEXT DEFAULT '',
		chronic_conditions TEXT DEFAULT '',
		last_checkup TEXT DEFAULT '',
		infirmary_visits TEXT DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS store_products (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		category TEXT NOT NULL,
		price REAL NOT NULL,
		sizes TEXT DEFAULT '',
		in_stock INTEGER DEFAULT 1,
		image_url TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS store_carts (
		student_id TEXT PRIMARY KEY,
		items_json TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS store_orders (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		items_json TEXT NOT NULL,
		total_amount REAL NOT NULL,
		status TEXT DEFAULT 'CONFIRMED',
		payment_id TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS library_books (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		isbn TEXT NOT NULL,
		category TEXT NOT NULL,
		available_copies INTEGER DEFAULT 1,
		total_copies INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS library_borrowings (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		book_id TEXT NOT NULL,
		borrowed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		due_date DATETIME NOT NULL,
		returned_at DATETIME,
		status TEXT DEFAULT 'BORROWED'
	);

	CREATE TABLE IF NOT EXISTS clubs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		coordinator TEXT NOT NULL,
		schedule TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS club_memberships (
		club_id TEXT NOT NULL,
		student_id TEXT NOT NULL,
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(club_id, student_id)
	);

	CREATE TABLE IF NOT EXISTS houses (
		name TEXT PRIMARY KEY,
		color TEXT NOT NULL,
		points INTEGER DEFAULT 0,
		captain TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS house_point_logs (
		id TEXT PRIMARY KEY,
		house_name TEXT NOT NULL,
		points INTEGER NOT NULL,
		reason TEXT NOT NULL,
		awarded_by TEXT NOT NULL,
		student_id TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS lost_items (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		category TEXT NOT NULL,
		location TEXT NOT NULL,
		item_type TEXT NOT NULL, -- LOST or FOUND
		reporter_id TEXT NOT NULL,
		image_url TEXT DEFAULT '',
		claimed_by TEXT DEFAULT '',
		status TEXT DEFAULT 'OPEN',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS photo_albums (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		date TEXT NOT NULL,
		thumbnail_url TEXT NOT NULL,
		photos_json TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS parent_threads (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		parent_id TEXT NOT NULL,
		teacher_id TEXT NOT NULL,
		subject TEXT NOT NULL,
		messages_json TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ptm_teacher_timings (
		teacher_id TEXT PRIMARY KEY,
		teacher_name TEXT NOT NULL,
		subject TEXT NOT NULL,
		slots_json TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS ptm_bookings (
		id TEXT PRIMARY KEY,
		student_id TEXT NOT NULL,
		teacher_id TEXT NOT NULL,
		slot_time TEXT NOT NULL,
		agenda TEXT NOT NULL,
		status TEXT DEFAULT 'CONFIRMED',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS calendar_events (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		location TEXT DEFAULT '',
		details TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS notices (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		category TEXT DEFAULT 'GENERAL',
		target_class TEXT DEFAULT '',
		attachment_name TEXT DEFAULT '',
		urgent INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notification_tracking (
		reference_id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		sent_count INTEGER DEFAULT 0,
		delivered_count INTEGER DEFAULT 0,
		read_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS school_info (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		name TEXT NOT NULL,
		affiliation TEXT NOT NULL,
		school_code TEXT NOT NULL,
		address TEXT NOT NULL,
		phone TEXT NOT NULL,
		email TEXT NOT NULL,
		working_hours TEXT NOT NULL,
		term_dates_json TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS staff (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		phone TEXT NOT NULL,
		role TEXT NOT NULL,
		designation TEXT NOT NULL,
		assigned_classes_json TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		reference_no TEXT PRIMARY KEY,
		actor_id TEXT NOT NULL,
		actor_role TEXT NOT NULL,
		action TEXT NOT NULL,
		target_resource TEXT NOT NULL,
		payload_hash TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return err
	}

	return seedSQLiteData(ctx, db)
}

func seedSQLiteData(ctx context.Context, db *sql.DB) error {
	// Seed School Info if absent
	var count int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM school_info").Scan(&count)
	if count == 0 {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO school_info (id, name, affiliation, school_code, address, phone, email, working_hours, term_dates_json)
			VALUES (1, 'NexaCampus International Academy', 'CBSE Affiliation #930142', 'SCH-930142',
				'42 Cyber City, Tech Park Road, Bangalore, KA 560100', '+91 80 4912 3400',
				'principal@nexacampus.edu.in', '08:00 AM - 04:00 PM (Mon-Sat)',
				'["Term 1: 01 Jun - 15 Oct", "Term 2: 01 Nov - 28 Mar"]')
		`)
	}

	// Seed Sample Student if absent
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM students").Scan(&count)
	if count == 0 {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO students (student_id, name, class_id, section_id, roll_no, admission_id, dob, father_name, mother_name, blood_group, mobile, email, address, profile_photo_url, medical_notes)
			VALUES ('STU1001', 'Aarav Sharma', '10', 'A', '101', 'ADM-2023-0881', '2009-08-15', 'Vikram Sharma', 'Pooja Sharma', 'O+', '+91 98765 43210', 'aarav.sharma@nexacampus.edu.in', 'Flat 402, Green Glen Layout, Bangalore', 'https://images.unsplash.com/photo-1534528741775-53994a69daeb', 'Asthma inhaler in backpack')
		`)
		_, _ = db.ExecContext(ctx, `
			INSERT INTO canteen_wallets (student_id, balance, daily_limit)
			VALUES ('STU1001', 1250.0, 500.0)
		`)
		_, _ = db.ExecContext(ctx, `
			INSERT INTO health_records (student_id, blood_group, allergies, chronic_conditions, last_checkup, infirmary_visits)
			VALUES ('STU1001', 'O+', 'Peanuts', 'Asthma', '2026-06-10', 'Infirmary visit on 2026-08-12: Mild headache, prescribed rest.')
		`)
	}

	// Seed Staff if absent
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM staff").Scan(&count)
	if count == 0 {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO staff (id, name, email, phone, role, designation, assigned_classes_json)
			VALUES ('STF2001', 'Dr. Radhika Iyer', 'radhika.iyer@nexacampus.edu.in', '+91 98111 22334', 'teacher', 'Senior Physics HOD', '["10-A", "10-B", "12-A"]')
		`)
	}

	// Seed Houses if absent
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM houses").Scan(&count)
	if count == 0 {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO houses (name, color, points, captain) VALUES
			('Phoenix', 'Red', 450, 'Rohan Verma'),
			('Pegasus', 'Blue', 510, 'Ananya Roy'),
			('Hydra', 'Green', 420, 'Karan Malhotra'),
			('Orion', 'Yellow', 495, 'Sneha Patel');
		`)
	}

	// Seed Store Products if absent
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM store_products").Scan(&count)
	if count == 0 {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO store_products (id, title, category, price, sizes, in_stock, image_url) VALUES
			('PRD-101', 'Complete Summer Uniform Set', 'Uniforms', 1450.0, 'S,M,L,XL', 80, 'https://images.unsplash.com/photo-1576995853123-5a10305d93c0'),
			('PRD-102', 'Grade 10 NCERT Science Textbook', 'Books', 320.0, 'One Size', 150, 'https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c'),
			('PRD-103', 'Official Campus Sports Jersey', 'Sportswear', 750.0, 'S,M,L,XL', 65, 'https://images.unsplash.com/photo-1508214751196-bcfd4ca60f91');
		`)
	}

	return nil
}
