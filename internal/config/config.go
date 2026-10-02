package config

import (
	"bufio"
	"os"
	"strings"
)

// Config represents all application configuration loaded from environment variables.
type Config struct {
	Port                  string
	DatabaseURL           string
	JWTSecret             string
	SupabaseURL           string
	SupabaseServiceKey    string
	SupabaseAnonKey       string
	CORSOrigins           []string
}

// Load reads configuration from environment variables, falling back to .env if present.
func Load() *Config {
	loadDotEnv(".env")

	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "")
	jwtSecret := getEnv("JWT_SECRET", "super_secret_school_jwt_key_2026_stitch")
	supabaseURL := getEnv("SUPABASE_URL", "https://qckzgqekfvqmxuoeyhhj.supabase.co")
	supabaseServiceKey := getEnv("SUPABASE_SERVICE_ROLE_KEY", "")
	supabaseAnonKey := getEnv("SUPABASE_ANON_KEY", "")

	corsRaw := getEnv("CORS_ORIGINS", "*")
	corsOrigins := strings.Split(corsRaw, ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		SupabaseURL:        supabaseURL,
		SupabaseServiceKey: supabaseServiceKey,
		SupabaseAnonKey:    supabaseAnonKey,
		CORSOrigins:        corsOrigins,
	}
}

func getEnv(key, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(val) == "" {
		return defaultVal
	}
	return strings.TrimSpace(val)
}

// loadDotEnv loads key-value pairs from a simple .env file into the environment
// if the environment variables are not already set.
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
