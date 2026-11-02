package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/zpol/vscale/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Service struct {
	db     *gorm.DB
	jwtKey []byte
	ttl    time.Duration
}

func NewService(db *gorm.DB, jwtSecret string, ttl time.Duration) *Service {
	if ttl == 0 {
		ttl = 72 * time.Hour
	}
	return &Service{db: db, jwtKey: []byte(jwtSecret), ttl: ttl}
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *Service) IssueToken(user models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtKey)
}

func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return s.jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func RandomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		claims, err := s.ParseToken(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	set := map[string]struct{}{}
	for _, r := range roles {
		set[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		rs, _ := role.(string)
		if _, ok := set[rs]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// AgentAuth validates X-Vscale-Token for devices or hubs.
func AgentAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := c.GetHeader("X-Vscale-Token")
		if tok == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing agent token"})
			return
		}
		var device models.Device
		if err := db.Where("agent_token = ? AND status = ?", tok, models.StatusActive).First(&device).Error; err == nil {
			c.Set("agentType", "device")
			c.Set("deviceId", device.ID)
			c.Set("vscaleId", device.VscaleID)
			c.Next()
			return
		}
		var hub models.Hub
		if err := db.Where("agent_token = ? AND status = ?", tok, models.StatusActive).First(&hub).Error; err == nil {
			c.Set("agentType", "hub")
			c.Set("hubId", hub.ID)
			c.Set("vscaleId", hub.VscaleID)
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
	}
}

func UserID(c *gin.Context) uint {
	v, _ := c.Get("userId")
	id, _ := v.(uint)
	return id
}

func Role(c *gin.Context) string {
	v, _ := c.Get("role")
	r, _ := v.(string)
	return r
}

func IsGlobalAdmin(c *gin.Context) bool {
	return Role(c) == models.RoleAdmin
}
