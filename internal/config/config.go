package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr         string
	DBHost           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBPort           string
	JWTSecret        string
	JWTTTL           time.Duration
	DefaultAdminUser string
	DefaultAdminPass string
	DefaultAdminMail string
	GlobalPool       string
	VscalePrefixLen  int
	CORSOrigins      string
	PublicURL        string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func Load() Config {
	ttlHours, _ := strconv.Atoi(getenv("JWT_TTL_HOURS", "72"))
	prefix, _ := strconv.Atoi(getenv("VSCALE_PREFIX_LEN", "24"))
	return Config{
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		DBHost:           getenv("DB_HOST", "localhost"),
		DBUser:           getenv("DB_USER", "vscale"),
		DBPassword:       getenv("DB_PASSWORD", "vscale"),
		DBName:           getenv("DB_NAME", "vscale"),
		DBPort:           getenv("DB_PORT", "5432"),
		JWTSecret:        getenv("JWT_SECRET", "change-me-in-production"),
		JWTTTL:           time.Duration(ttlHours) * time.Hour,
		DefaultAdminUser: getenv("DEFAULT_ADMIN_USER", "admin"),
		DefaultAdminPass: getenv("DEFAULT_ADMIN_PASSWORD", "admin123"),
		DefaultAdminMail: getenv("DEFAULT_ADMIN_EMAIL", "admin@vscale.local"),
		// Keep clear of Tailscale CGNAT (100.64.0.0/10).
		GlobalPool:      getenv("VSCALE_GLOBAL_POOL", "10.66.0.0/16"),
		VscalePrefixLen: prefix,
		CORSOrigins:      getenv("CORS_ORIGINS", "*"),
		PublicURL:        getenv("PUBLIC_URL", "http://localhost:8080"),
	}
}

func (c Config) DSN() string {
	return "host=" + c.DBHost +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" port=" + c.DBPort +
		" sslmode=disable"
}
