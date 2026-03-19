package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/config"
	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const CurrentAdminKey = "current_admin"

type Claims struct {
	AdminID uint64 `json:"aid"`
	jwt.RegisteredClaims
}

type Service struct {
	repo *repository.Repository
	cfg  config.Config
}

func NewService(repo *repository.Repository, cfg config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) EnsureSeed() error {
	_, err := s.repo.FindAdminByUsername("root")
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.repo.CreateAdminUser(&repository.AdminUser{
		Username:           "root",
		PasswordHash:       string(hash),
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
}

func (s *Service) Login(username, password string) (*contracts.AdminAuthPayload, error) {
	adminUser, err := s.repo.FindAdminByUsername(strings.TrimSpace(username))
	if err != nil {
		return nil, apperr.Unauthorized("INVALID_ADMIN_CREDENTIALS", "invalid admin username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(strings.TrimSpace(password))); err != nil {
		return nil, apperr.Unauthorized("INVALID_ADMIN_CREDENTIALS", "invalid admin username or password")
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		AdminID: adminUser.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.cfg.JWTIssuer + ":admin",
			Subject:   fmt.Sprintf("%d", adminUser.ID),
		},
	}).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}
	return &contracts.AdminAuthPayload{
		AccessToken: token,
		Admin: contracts.AdminProfile{
			ID:                 adminUser.ID,
			Username:           adminUser.Username,
			MustChangePassword: adminUser.MustChangePassword,
		},
	}, nil
}

func (s *Service) Me(adminID uint64) (*contracts.AdminProfile, error) {
	adminUser, err := s.repo.FindAdminByID(adminID)
	if err != nil {
		return nil, err
	}
	return &contracts.AdminProfile{
		ID:                 adminUser.ID,
		Username:           adminUser.Username,
		MustChangePassword: adminUser.MustChangePassword,
	}, nil
}

func (s *Service) ChangePassword(adminID uint64, currentPassword, newPassword string) error {
	adminUser, err := s.repo.FindAdminByID(adminID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(strings.TrimSpace(currentPassword))); err != nil {
		return apperr.Unauthorized("INVALID_ADMIN_CREDENTIALS", "invalid current password")
	}
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 6 {
		return apperr.BadRequest("INVALID_ADMIN_PASSWORD", "new password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdateAdminPassword(adminID, string(hash), false)
}

func (s *Service) ParseAccessToken(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, apperr.Unauthorized("INVALID_ADMIN_TOKEN", "invalid admin token")
	}
	return claims, nil
}

func (s *Service) Middleware(requirePasswordChanged bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			c.JSON(http.StatusUnauthorized, contracts.Envelope{Code: http.StatusUnauthorized, Message: "missing Authorization header"})
			c.Abort()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, contracts.Envelope{Code: http.StatusUnauthorized, Message: "invalid Authorization header"})
			c.Abort()
			return
		}
		claims, err := s.ParseAccessToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, contracts.Envelope{Code: http.StatusUnauthorized, Message: err.Error()})
			c.Abort()
			return
		}
		adminUser, err := s.repo.FindAdminByID(claims.AdminID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, contracts.Envelope{Code: http.StatusUnauthorized, Message: "admin no longer exists"})
			c.Abort()
			return
		}
		if requirePasswordChanged && adminUser.MustChangePassword {
			c.JSON(http.StatusForbidden, contracts.Envelope{
				Code:      http.StatusForbidden,
				Message:   "admin must change password first",
				ErrorCode: "ADMIN_PASSWORD_CHANGE_REQUIRED",
			})
			c.Abort()
			return
		}
		c.Set(CurrentAdminKey, adminUser)
		c.Next()
	}
}

func CurrentAdmin(c *gin.Context) (*repository.AdminUser, bool) {
	value, ok := c.Get(CurrentAdminKey)
	if !ok {
		return nil, false
	}
	adminUser, ok := value.(*repository.AdminUser)
	if !ok {
		return nil, false
	}
	return adminUser, true
}
