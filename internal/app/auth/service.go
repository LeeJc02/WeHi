package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const CurrentUserKey = "current_user"

type SessionState struct {
	ID         string `json:"id"`
	UserID     uint64 `json:"user_id"`
	DeviceID   string `json:"device_id"`
	UserAgent  string `json:"user_agent"`
	Refresh    string `json:"refresh"`
	ExpiresAt  string `json:"expires_at"`
	LastSeenAt string `json:"last_seen_at"`
}

type Claims struct {
	UserID    uint64 `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type Service struct {
	repo  *repository.Repository
	redis *redis.Client
	cfg   config.Config
}

func NewService(repo *repository.Repository, redisClient *redis.Client, cfg config.Config) *Service {
	return &Service{repo: repo, redis: redisClient, cfg: cfg}
}

func (s *Service) Register(username, displayName, password string) (contracts.UserProfile, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	password = strings.TrimSpace(password)
	if len(username) < 3 || len(displayName) < 2 || password == "" {
		return contracts.UserProfile{}, apperr.BadRequest("INVALID_ARGUMENT", "username/display_name/password do not meet minimum requirements")
	}
	if _, err := s.repo.FindUserByUsername(username); err == nil {
		return contracts.UserProfile{}, apperr.Conflict("USERNAME_ALREADY_EXISTS", "username already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return contracts.UserProfile{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return contracts.UserProfile{}, err
	}
	user := &repository.User{
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.repo.CreateUser(user); err != nil {
		return contracts.UserProfile{}, err
	}
	return toProfile(user), nil
}

func (s *Service) Login(ctx context.Context, username, password, deviceID, userAgent string) (contracts.AuthPayload, error) {
	user, err := s.repo.FindUserByUsername(strings.TrimSpace(username))
	if err != nil {
		return contracts.AuthPayload{}, apperr.Unauthorized("INVALID_CREDENTIALS", "invalid username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(strings.TrimSpace(password))); err != nil {
		return contracts.AuthPayload{}, apperr.Unauthorized("INVALID_CREDENTIALS", "invalid username or password")
	}
	return s.issueSession(ctx, user, normalizedDeviceID(deviceID), userAgent)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (contracts.AuthPayload, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	state, err := s.readSessionByRefresh(ctx, refreshToken)
	if err != nil {
		return contracts.AuthPayload{}, apperr.Unauthorized("INVALID_REFRESH_TOKEN", "refresh token is invalid or expired")
	}
	user, err := s.repo.FindUserByID(state.UserID)
	if err != nil {
		return contracts.AuthPayload{}, err
	}
	if err := s.deleteRefresh(ctx, refreshToken); err != nil {
		return contracts.AuthPayload{}, err
	}
	return s.issueSessionWithID(ctx, user, state.ID, state.DeviceID, state.UserAgent)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	state, err := s.readSessionByRefresh(ctx, refreshToken)
	if err != nil {
		return nil
	}
	return s.deleteSession(ctx, state.UserID, state.ID, refreshToken)
}

func (s *Service) LogoutAll(ctx context.Context, userID uint64) error {
	sessionIDs, err := s.redis.SMembers(ctx, sessionSetKey(userID)).Result()
	if err != nil {
		return err
	}
	for _, sessionID := range sessionIDs {
		raw, err := s.redis.Get(ctx, sessionKey(userID, sessionID)).Result()
		if err != nil {
			continue
		}
		var state SessionState
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			continue
		}
		if err := s.deleteSession(ctx, userID, sessionID, state.Refresh); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListSessions(ctx context.Context, userID uint64, currentSessionID string) ([]contracts.SessionInfo, error) {
	sessionIDs, err := s.redis.SMembers(ctx, sessionSetKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	result := make([]contracts.SessionInfo, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		raw, err := s.redis.Get(ctx, sessionKey(userID, sessionID)).Result()
		if err != nil {
			continue
		}
		var state SessionState
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			continue
		}
		result = append(result, contracts.SessionInfo{
			ID:         state.ID,
			DeviceID:   state.DeviceID,
			UserAgent:  state.UserAgent,
			LastSeenAt: state.LastSeenAt,
			ExpiresAt:  state.ExpiresAt,
			Current:    currentSessionID == state.ID,
		})
	}
	return result, nil
}

func (s *Service) ParseAccessToken(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, apperr.Unauthorized("INVALID_ACCESS_TOKEN", "invalid access token")
	}
	return claims, nil
}

func (s *Service) Middleware() gin.HandlerFunc {
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
		user, err := s.repo.FindUserByID(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, contracts.Envelope{Code: http.StatusUnauthorized, Message: "user no longer exists"})
			c.Abort()
			return
		}
		c.Set(CurrentUserKey, user)
		c.Set("session_id", claims.SessionID)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (*repository.User, string, bool) {
	userValue, ok := c.Get(CurrentUserKey)
	if !ok {
		return nil, "", false
	}
	sessionID, _ := c.Get("session_id")
	user, ok := userValue.(*repository.User)
	if !ok {
		return nil, "", false
	}
	session, _ := sessionID.(string)
	return user, session, true
}

func (s *Service) TouchSession(ctx context.Context, claims *Claims) error {
	raw, err := s.redis.Get(ctx, sessionKey(claims.UserID, claims.SessionID)).Result()
	if err != nil {
		return err
	}
	var state SessionState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return err
	}
	state.LastSeenAt = time.Now().UTC().Format(time.RFC3339)
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	expireAt, _ := time.Parse(time.RFC3339, state.ExpiresAt)
	return s.redis.Set(ctx, sessionKey(claims.UserID, claims.SessionID), payload, time.Until(expireAt)).Err()
}

func (s *Service) issueSession(ctx context.Context, user *repository.User, deviceID, userAgent string) (contracts.AuthPayload, error) {
	return s.issueSessionWithID(ctx, user, randomToken(16), deviceID, userAgent)
}

func (s *Service) issueSessionWithID(ctx context.Context, user *repository.User, sessionID, deviceID, userAgent string) (contracts.AuthPayload, error) {
	refreshToken := randomToken(32)
	now := time.Now().UTC()
	expiresAt := now.Add(s.cfg.RefreshTokenTTL)
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:    user.ID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    s.cfg.JWTIssuer,
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return contracts.AuthPayload{}, err
	}

	state := SessionState{
		ID:         sessionID,
		UserID:     user.ID,
		DeviceID:   deviceID,
		UserAgent:  userAgent,
		Refresh:    refreshToken,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
		LastSeenAt: now.Format(time.RFC3339),
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return contracts.AuthPayload{}, err
	}
	if err := s.redis.Set(ctx, sessionKey(user.ID, sessionID), payload, s.cfg.RefreshTokenTTL).Err(); err != nil {
		return contracts.AuthPayload{}, err
	}
	if err := s.redis.SAdd(ctx, sessionSetKey(user.ID), sessionID).Err(); err != nil {
		return contracts.AuthPayload{}, err
	}
	if err := s.redis.Set(ctx, refreshKey(refreshToken), payload, s.cfg.RefreshTokenTTL).Err(); err != nil {
		return contracts.AuthPayload{}, err
	}
	return contracts.AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toProfile(user),
	}, nil
}

func (s *Service) readSessionByRefresh(ctx context.Context, refreshToken string) (*SessionState, error) {
	raw, err := s.redis.Get(ctx, refreshKey(refreshToken)).Result()
	if err != nil {
		return nil, err
	}
	var state SessionState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Service) deleteRefresh(ctx context.Context, refreshToken string) error {
	return s.redis.Del(ctx, refreshKey(refreshToken)).Err()
}

func (s *Service) deleteSession(ctx context.Context, userID uint64, sessionID, refreshToken string) error {
	if err := s.redis.Del(ctx, refreshKey(refreshToken), sessionKey(userID, sessionID)).Err(); err != nil {
		return err
	}
	return s.redis.SRem(ctx, sessionSetKey(userID), sessionID).Err()
}

func toProfile(user *repository.User) contracts.UserProfile {
	return contracts.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}
}

func refreshKey(refresh string) string {
	return "refresh:" + refresh
}

func sessionKey(userID uint64, sessionID string) string {
	return fmt.Sprintf("session:%d:%s", userID, sessionID)
}

func sessionSetKey(userID uint64) string {
	return fmt.Sprintf("sessions:%d", userID)
}

func normalizedDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "browser"
	}
	return deviceID
}

func randomToken(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
