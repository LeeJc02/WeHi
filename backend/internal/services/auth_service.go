package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"awesomeproject/backend/internal/models"
	"awesomeproject/backend/internal/repositories"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	users    *repositories.UserRepository
	sessions *repositories.SessionRepository
}

func NewAuthService(users *repositories.UserRepository, sessions *repositories.SessionRepository) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

func (s *AuthService) Register(username, displayName, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	password = strings.TrimSpace(password)

	if username == "" || displayName == "" || password == "" {
		return nil, errors.New("username, display_name and password are required")
	}

	if _, err := s.users.FindByUsername(username); err == nil {
		return nil, errors.New("username already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: string(hash),
	}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(username, password string) (*models.Session, *models.User, error) {
	user, err := s.users.FindByUsername(strings.TrimSpace(username))
	if err != nil {
		return nil, nil, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(strings.TrimSpace(password))); err != nil {
		return nil, nil, errors.New("invalid username or password")
	}

	token, err := generateToken()
	if err != nil {
		return nil, nil, err
	}

	session := &models.Session{
		Token:  token,
		UserID: user.ID,
	}
	if err := s.sessions.Create(session); err != nil {
		return nil, nil, err
	}
	return session, user, nil
}

func (s *AuthService) Authenticate(token string) (*models.User, error) {
	session, err := s.sessions.FindByToken(strings.TrimSpace(token))
	if err != nil {
		return nil, errors.New("invalid token")
	}
	return s.users.FindByID(session.UserID)
}

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
