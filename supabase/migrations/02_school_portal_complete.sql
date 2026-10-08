-- ============================================================================
-- EduPortal / Stitch School Portal: Complete Schema & Sample Dataset
-- Compatible with Supabase Postgres 15+ and pgxpool transaction mode (port 6543)
-- Covers all 21 screens and functionality modules in nexaCampus/android-school-app
-- ============================================================================

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

-- 21. Row-Level Security (RLS) Enablement & Service Role Policies
DO $$
DECLARE
    t text;
    tables text[] := ARRAY[
        'students', 'homework', 'homework_submissions', 'attendance', 'leave_applications',
        'timetable', 'fee_items', 'fee_history', 'notices', 'helpdesk_tickets',
        'bus_routes', 'bus_live_locations', 'library_books', 'library_borrowings',
        'library_reservations', 'exam_hall_tickets', 'canteen_menu', 'canteen_transactions',
        'ptm_sessions', 'ptm_slots', 'digital_hall_passes', 'health_profiles',
        'clinic_visits', 'medical_consents', 'student_merits', 'store_products',
        'store_orders', 'lost_found_items', 'gallery_albums', 'gallery_photos',
        'clubs', 'student_clubs', 'club_competitions', 'competition_registrations'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE IF EXISTS %I ENABLE ROW LEVEL SECURITY;', t);
        BEGIN
            EXECUTE format('CREATE POLICY %I ON %I FOR ALL TO service_role USING (true) WITH CHECK (true);', 'service_role_' || t, t);
        EXCEPTION WHEN duplicate_object THEN
            NULL;
        END;
    END LOOP;
END $$;

