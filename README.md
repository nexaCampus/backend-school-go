# EduPortal / Stitch School Portal Backend (Go)

Production-grade, high-performance Go backend service for the **Stitch School Portal / EduPortal** mobile and web applications (`nexaCampus/android-school-app`). Engineered for zero-crash execution on Render's free tier (consuming less than 30MB RAM) with direct connection pooling to PostgreSQL on Supabase via `pgxpool`.

---

## 1. Tech Stack & Architecture

- **Language:** Go 1.23+
- **HTTP Router:** [`github.com/go-chi/chi/v5`](https://github.com/go-chi/chi) with `middleware` (Logger, Recoverer, Compress, RequestID, RealIP, CORS).
- **Database Driver & Pool:** [`github.com/jackc/pgx/v5`](https://github.com/jackc/pgx) with connection pooling via `pgxpool` (tuned for Supabase transaction poolers on Port 6543).
- **JWT Authentication:** [`github.com/golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) with HMAC-SHA256 signing, 7-day access tokens, and 30-day token rotation.
- **Containerization:** Multi-stage Docker build producing an Alpine image (< 20MB).
- **Database:** Supabase PostgreSQL with automated schema initialization and rich demo data seeding.

---

## 2. API Route Directory (All 21 Client UI Modules)

All endpoints accept and return JSON payloads under `/v1` (with `/healthz` served at root):

### 1. System & Authentication (`LoginVerificationScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | None | Render keep-warm / container health check (`200 OK`, body `"OK"`). |
| `POST` | `/v1/auth/verify` | `{ "student_id": "...", "dob": "YYYY-MM-DD" }` | Validates credentials and returns 7-day JWT token + student profile. |
| `POST` | `/v1/auth/refresh` | `{ "refresh_token": "..." }` | Rotates expired access tokens without re-login. |

### 2. Student Profile & Identity (`StudentProfileScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/students/{id}` | Path param: `student_id` | Fetches personal details, guardian contacts, blood group, and class info. |
| `PUT` | `/v1/students/{id}/avatar` | `{ "avatar_url": "..." }` | Updates profile avatar after uploading to Supabase Storage. |
| `GET` | `/v1/students/{id}/digital-badge` | Path param: `student_id` | Returns dynamic cryptographic QR payload for campus gate checks. |

### 3. Home Dashboard (`HomeDashboardScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/dashboard/summary` | `?student_id={id}&grade={grade}&section={section}` | Aggregates attendance gauge, today's periods, pending homework count, and urgent notices in a single roundtrip. |

### 4. Homework & Diary (`HomeworkDiaryScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/homework` | `?grade={grade}&section={section}&status={all\|pending\|submitted}` | Homework sorted by due date with student submission status. |
| `GET` | `/v1/homework/{id}` | Path param: `homework_id` | Detailed instructions, reference materials, and teacher attachments. |
| `POST` | `/v1/homework/{id}/submit` | `{ "student_id": "...", "submission_file_url": "..." }` | Records student submission timestamp and file link. |

### 5. Attendance & Leave Applications (`AttendanceLeaveScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/attendance` | `?student_id={id}&month={1-12}&year={year}` | Monthly status matrix (`PRESENT`, `ABSENT`, `HOLIDAY`, `SUNDAY`). |
| `GET` | `/v1/attendance/summary` | `?student_id={id}` | Total working days, days present/absent/late, and overall percentage. |
| `POST` | `/v1/attendance/leave` | `{ "student_id": "...", "leave_type": "...", "start_date": "...", "end_date": "...", "reason": "...", "attachment_url": "..." }` | Submits formal leave request for school approval. |
| `GET` | `/v1/attendance/leave` | `?student_id={id}` | Past leave applications and approval statuses. |

### 6. Timetable & Bell Schedule (`TimetableCalendarScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/timetable` | `?grade={grade}&section={section}&day={1-6}` | Period schedule (subject, teacher, timing, classroom, recess breaks). |

### 7. Fees & Academic Performance (`AcademicsFeeScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/fees` | `?student_id={id}` | Invoice items, total due, amount paid, and receipts. |
| `GET` | `/v1/academics/report-card` | `?student_id={id}&term={term_name}` | Subject marks, GPA, teacher remarks, and marksheet download URL. |

### 8. Notice Board & Circulars (`NoticeBoardScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/notices` | `?category={all\|urgent\|academic\|event}&limit={n}` | Official circulars with priority tags and PDF attachments. |

### 9. Helpdesk & Support (`WriteToSchoolScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `POST` | `/v1/helpdesk/tickets` | `{ "student_id": "...", "department": "...", "subject": "...", "message": "...", "attachment_url": "..." }` | Opens an inquiry ticket with school administration. |
| `GET` | `/v1/helpdesk/tickets` | `?student_id={id}` | Active and resolved ticket threads with staff replies. |

### 10. School Transport & Live GPS (`LiveBusTrackingScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/transport/routes/{route}` | Path param: `route_number` | Bus route metadata, driver contact, stops, and student ETA. |
| `POST` | `/v1/transport/routes/{route}/location` | `{ "lat": 22.438510, "lng": 88.397020 }` | Driver device push for real-time GPS coordinates. |

### 11. School Library (`SchoolLibraryScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/library/books` | `?query={keyword}&category={category}` | Book inventory and copy availability. |
| `GET` | `/v1/library/borrowings` | `?student_id={id}` | Borrowed books, return countdown, and accrued fines. |
| `POST` | `/v1/library/reserve` | `{ "student_id": "...", "book_id": "..." }` | Places a temporary hold on a library book. |

### 12. Exam Hall Tickets (`HallTicketReportCardScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/exams/hall-ticket` | `?student_id={id}&term={term_name}` | Center, seat number, exam timetable, and entry rules. |

### 13. Canteen & Smart Card Wallet (`CanteenWalletScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/canteen/wallet` | `?student_id={id}` | Balance and cafeteria transaction history. |
| `GET` | `/v1/canteen/menu` | `?day={day_name}` | Available cafeteria menu with dietary tags and pricing. |

### 14. Parent-Teacher Meeting (PTM) Scheduler (`PtmBookingScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/ptm/sessions` | `?grade={grade}` | Upcoming PTM dates and participating faculty. |
| `GET` | `/v1/ptm/slots` | `?teacher_id={id}&date={date}` | Available 15-minute conference time slots. |
| `POST` | `/v1/ptm/book` | `{ "student_id": "...", "slot_id": "...", "agenda": "..." }` | Confirms appointment and generates video link. |

### 15. Digital Hall Pass Engine (`DigitalHallPassScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `POST` | `/v1/hallpass/request` | `{ "student_id": "...", "destination": "...", "duration_minutes": 10 }` | Sends pass request to active period teacher. |
| `GET` | `/v1/hallpass/active` | `?student_id={id}` | Active pass status, countdown expiry, and verification QR. |

### 16. Student Infirmary & Medical Dashboard (`StudentInfirmaryScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/health/profile` | `?student_id={id}` | Blood group, verified allergies, emergency contacts. |
| `GET` | `/v1/health/clinic-visits` | `?student_id={id}` | Historical nurse clinic visits, temperatures, first aid. |
| `POST` | `/v1/health/consent` | `{ "student_id": "...", "medication_name": "...", "prescription_url": "..." }` | Parent authorization for nurse to dispense midday medicine. |

### 17. House System, Merits & Discipline (`HouseMeritScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/merits/summary` | `?student_id={id}` | House affiliation, merit points, and house leaderboard. |
| `GET` | `/v1/merits/records` | `?student_id={id}` | Commendation badges and disciplinary notices. |

### 18. School Store & Inventory (`SchoolStoreScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/store/products` | `?category={uniform\|books\|stationery}` | Products, sizes, prices, and stock status. |
| `POST` | `/v1/store/order` | `{ "student_id": "...", "items": [...], "delivery_option": "locker\|desk" }` | Places pre-order billed to student smart card. |

### 19. Campus Lost & Found (`LostAndFoundScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/lost-found/items` | `?status={found\|claimed}` | Lists found articles with image URLs and locations. |
| `POST` | `/v1/lost-found/report` | `{ "student_id": "...", "item_name": "...", "last_seen_location": "...", "image_url": "..." }` | Submits report for a lost item. |

### 20. Event Gallery & Yearbook (`EventYearbookScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/gallery/albums` | `?year={year}` | Lists event photo albums. |
| `GET` | `/v1/gallery/albums/{id}` | Path param: `album_id` | Image URLs within the selected event album. |

### 21. Clubs & Societies (`ClubSocietiesScreen`)
| Method | Endpoint | Payload / Query | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/v1/clubs/enrolled` | `?student_id={id}` | Enrolled clubs and upcoming meeting schedules. |
| `GET` | `/v1/clubs/competitions` | `?grade={grade}` | Inter-school competitions, deadlines, registration links. |
| `POST` | `/v1/clubs/competitions/{id}/register` | `{ "student_id": "..." }` | 1-click event entry registration. |

---

## 3. Database Schema Migration

The backend includes automatic runtime schema migration in [`internal/database/schema.go`](file:///e:/nexaCampus/backend-school-go/internal/database/schema.go).

To manually execute the complete schema in the Supabase SQL Editor:
Open [`supabase/migrations/02_school_portal_complete.sql`](file:///e:/nexaCampus/backend-school-go/supabase/migrations/02_school_portal_complete.sql) and run the script. It creates all tables, composite indexes, and seeds production-grade sample records matching `Rajen Shaw` (ID: `10211125`, DOB: `2007-02-10`, Grade `12-E`).

---

## 4. Environment Configuration

```env
# Render Server Port (Auto-injected by Render)
PORT=8080

# Supabase Credentials (Project Ref: qckzgqekfvqmxuoeyhhj)
SUPABASE_URL=https://qckzgqekfvqmxuoeyhhj.supabase.co
SUPABASE_SERVICE_ROLE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InFja3pncWVrZnZxbXh1b2V5aGhqIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImlhdCI6MTc5MDk1MDk3OSwiZXhwIjoyMTA2NTI2OTc5fQ.NcWdyNsQAq52CoL6ZUtgqTkwKsqbcemaVq5IfFSb8zg
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InFja3pncWVrZnZxbXh1b2V5aGhqIiwicm9sZSI6ImFub24iLCJpYXQiOjE3OTA5NTA5NzksImV4cCI6MjEwNjUyNjk3OX0.GIc6F4ZvlyUBX2skR2agiGjlkQe4Cs4NFlLDO8YrYBU

# Direct PostgreSQL Connection String for pgxpool (Transaction Mode: Port 6543)
DATABASE_URL=postgresql://postgres.qckzgqekfvqmxuoeyhhj:[YOUR_DATABASE_PASSWORD]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres?sslmode=require

# App Security
JWT_SECRET=super_secret_school_jwt_key_2026_stitch

# CORS Allowed Origins
CORS_ORIGINS=*
```

---

## 5. Development & Verification

```bash
# Run unit tests
go test -v ./...

# Static analysis
go vet ./...

# Run server
go run ./cmd/server
```
