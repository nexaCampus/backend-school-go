package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config encapsulates runtime parameters, storage modes, and third-party credentials.
type Config struct {
	Port               string
	StorageMode        string // "hybrid" (Supabase+Firebase) or "sqlite_local"
	DatabaseURL        string // Supabase / PostgreSQL URL
	SQLitePath         string // Path to local SQLite database file
	JWTSecret          string
	SupabaseURL        string
	SupabaseServiceKey string
	SupabaseAnonKey    string
	RazorpayKeyID      string
	RazorpayKeySecret  string
	FirebaseProjectID  string
	CORSOrigins        []string
	GOGC               int
	MemoryLimitMB      int64
}

// Load populates configuration from environment variables, falling back to sensible defaults.
func Load() *Config {
	loadDotEnv(".env")

	port := getEnv("PORT", "8080")
	storageMode := strings.ToLower(getEnv("STORAGE_MODE", "hybrid"))
	databaseURL := getEnv("DATABASE_URL", "")
	sqlitePath := getEnv("SQLITE_PATH", "nexacampus.db")
	jwtSecret := getEnv("JWT_SECRET", "super_secret_school_jwt_key_2026_stitch")

	supabaseURL := getEnv("SUPABASE_URL", "https://qckzgqekfvqmxuoeyhhj.supabase.co")
	supabaseServiceKey := getEnv("SUPABASE_SERVICE_ROLE_KEY", "")
	supabaseAnonKey := getEnv("SUPABASE_ANON_KEY", "")

	razorpayKeyID := getEnv("RAZORPAY_KEY_ID", "rzp_test_school_2026")
	razorpayKeySecret := getEnv("RAZORPAY_KEY_SECRET", "secret_key_school_2026")
	firebaseProjectID := getEnv("FIREBASE_PROJECT_ID", "qckzgqekfvqmxuoeyhhj")

	corsRaw := getEnv("CORS_ORIGINS", "*")
	corsOrigins := strings.Split(corsRaw, ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	gogc, _ := strconv.Atoi(getEnv("GOGC", "60"))
	if gogc <= 0 {
		gogc = 60
	}

	memLimitMB, _ := strconv.ParseInt(getEnv("MEM_LIMIT_MB", "700"), 10, 64)
	if memLimitMB <= 0 {
		memLimitMB = 700
	}

	isProduction := getEnv("ENV", "") == "production" ||
		getEnv("ENVIRONMENT", "") == "production" ||
		getEnv("RENDER", "") == "true"

	if isProduction {
		if jwtSecret == "super_secret_school_jwt_key_2026_stitch" || len(jwtSecret) < 32 {
			log.Fatalf("[FATAL] Insecure configuration: JWT_SECRET must be configured with a random secure secret of at least 32 characters in production mode!")
		}
	}

	return &Config{
		Port:               port,
		StorageMode:        storageMode,
		DatabaseURL:        databaseURL,
		SQLitePath:         sqlitePath,
		JWTSecret:          jwtSecret,
		SupabaseURL:        supabaseURL,
		SupabaseServiceKey: supabaseServiceKey,
		SupabaseAnonKey:    supabaseAnonKey,
		RazorpayKeyID:      razorpayKeyID,
		RazorpayKeySecret:  razorpayKeySecret,
		FirebaseProjectID:  firebaseProjectID,
		CORSOrigins:        corsOrigins,
		GOGC:               gogc,
		MemoryLimitMB:      memLimitMB,
	}
}

func getEnv(key, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(val) == "" {
		return defaultVal
	}
	return strings.TrimSpace(val)
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if _, exists := os.LookupEnv(k); !exists {
				os.Setenv(k, v)
			}
		}
	}
}
