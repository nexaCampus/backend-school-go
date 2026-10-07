-- NexaCampus High-Throughput SQLite Schema (WAL Optimized)
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;
PRAGMA mmap_size = 268435456;
PRAGMA temp_store = MEMORY;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;

-- 1. Students & Profiles
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

CREATE INDEX IF NOT EXISTS idx_students_class_sec ON students(class_id, section_id);

-- 2. Timetables & Class Operations
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

-- 3. Examinations & Results
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

-- 4. Financials & GST
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

-- 5. Canteen, Health, Store & Library
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

-- 6. Campus Life, Houses, Lost & Found
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
    item_type TEXT NOT NULL,
    reporter_id TEXT NOT NULL,
    image_url TEXT DEFAULT '',
    claimed_by TEXT DEFAULT '',
    status TEXT DEFAULT 'OPEN',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 7. Parent Desk, PTM & Notifications
CREATE TABLE IF NOT EXISTS parent_threads (
    id TEXT PRIMARY KEY,
    student_id TEXT NOT NULL,
    parent_id TEXT NOT NULL,
    teacher_id TEXT NOT NULL,
    subject TEXT NOT NULL,
    messages_json TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

-- 8. Staff & Audit Log
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
