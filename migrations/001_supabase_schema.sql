-- NexaCampus PostgreSQL / Supabase Schema

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

CREATE INDEX IF NOT EXISTS idx_students_student_id ON students(student_id);
CREATE INDEX IF NOT EXISTS idx_students_grade_section ON students(grade, section);

-- 2. Academics & Timetable
CREATE TABLE IF NOT EXISTS timetables (
    id BIGSERIAL PRIMARY KEY,
    class_id TEXT NOT NULL,
    section_id TEXT NOT NULL,
    day_of_week TEXT NOT NULL,
    period_no INT NOT NULL,
    subject TEXT NOT NULL,
    teacher TEXT NOT NULL,
    room TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    is_break BOOLEAN DEFAULT FALSE,
    UNIQUE(class_id, section_id, day_of_week, period_no)
);

CREATE TABLE IF NOT EXISTS lab_schedules (
    id BIGSERIAL PRIMARY KEY,
    lab_name TEXT NOT NULL,
    subject TEXT NOT NULL,
    teacher TEXT NOT NULL,
    room TEXT NOT NULL,
    day_of_week TEXT NOT NULL,
    timing TEXT NOT NULL
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
    created_at TIMESTAMPTZ DEFAULT now()
);

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

CREATE TABLE IF NOT EXISTS homework_submissions (
    id TEXT PRIMARY KEY,
    homework_id TEXT NOT NULL,
    student_id TEXT NOT NULL,
    submitted_at TIMESTAMPTZ DEFAULT now(),
    submission_file_url TEXT NOT NULL,
    score NUMERIC(8,2),
    feedback TEXT,
    UNIQUE(homework_id, student_id)
);

CREATE TABLE IF NOT EXISTS attendance (
    id BIGSERIAL PRIMARY KEY,
    student_id TEXT NOT NULL,
    date DATE NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(student_id, date)
);

CREATE TABLE IF NOT EXISTS leave_requests (
    id TEXT PRIMARY KEY,
    student_id TEXT NOT NULL,
    leave_type TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    reason TEXT NOT NULL,
    attachment_url TEXT,
    status TEXT DEFAULT 'pending',
    review_remarks TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 3. Payments & GST
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    student_id TEXT NOT NULL,
    order_id TEXT NOT NULL,
    payment_id TEXT DEFAULT '',
    amount NUMERIC(10,2) NOT NULL,
    currency TEXT DEFAULT 'INR',
    status TEXT NOT NULL,
    purpose TEXT DEFAULT '',
    receipt_ref TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tax_receipts (
    receipt_no TEXT PRIMARY KEY,
    student_id TEXT NOT NULL,
    payment_id TEXT NOT NULL,
    gstin TEXT NOT NULL,
    school_gstin TEXT NOT NULL,
    taxable_amount NUMERIC(10,2) NOT NULL,
    cgst NUMERIC(10,2) NOT NULL,
    sgst NUMERIC(10,2) NOT NULL,
    igst NUMERIC(10,2) NOT NULL,
    total_amount NUMERIC(10,2) NOT NULL,
    date DATE NOT NULL
);

-- 4. Audit Log
CREATE TABLE IF NOT EXISTS audit_logs (
    reference_no TEXT PRIMARY KEY,
    actor_id TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    action TEXT NOT NULL,
    target_resource TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT now()
);

-- 5. Row-Level Security (RLS) Policies
ALTER TABLE students ENABLE ROW LEVEL SECURITY;
ALTER TABLE timetables ENABLE ROW LEVEL SECURITY;
ALTER TABLE lab_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE diary_remarks ENABLE ROW LEVEL SECURITY;
ALTER TABLE homework ENABLE ROW LEVEL SECURITY;
ALTER TABLE attendance ENABLE ROW LEVEL SECURITY;
ALTER TABLE leaves ENABLE ROW LEVEL SECURITY;
ALTER TABLE exam_dates ENABLE ROW LEVEL SECURITY;
ALTER TABLE exam_timetable ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    EXECUTE 'CREATE POLICY "service_role_students" ON students FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_timetables" ON timetables FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_lab_schedules" ON lab_schedules FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_diary_remarks" ON diary_remarks FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_homework" ON homework FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_attendance" ON attendance FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_leaves" ON leaves FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_exam_dates" ON exam_dates FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_exam_timetable" ON exam_timetable FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_payments" ON payments FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_tax_receipts" ON tax_receipts FOR ALL TO service_role USING (true) WITH CHECK (true)';
    EXECUTE 'CREATE POLICY "service_role_audit_logs" ON audit_logs FOR ALL TO service_role USING (true) WITH CHECK (true)';
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

