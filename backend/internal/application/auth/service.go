package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend/internal/domain"
	"backend/internal/security"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrValidation         = errors.New("validation failed")
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)

type ClientMeta struct {
	UserAgent string
	ClientIP  string
}

type RegisterInput struct {
	Username    string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Username string
	Password string
}

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type Service struct {
	users      domain.UserRepository
	sessions   domain.SessionRepository
	hasher     *security.PasswordHasher
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(
	users domain.UserRepository,
	sessions domain.SessionRepository,
	hasher *security.PasswordHasher,
	accessTTL, refreshTTL time.Duration,
) *Service {
	return &Service{
		users:      users,
		sessions:   sessions,
		hasher:     hasher,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *Service) RegistrationOpen(ctx context.Context) (bool, error) {
	count, err := s.users.Count(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput, meta ClientMeta) (*domain.User, *TokenPair, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	displayName := strings.TrimSpace(input.DisplayName)
	if !usernamePattern.MatchString(username) {
		return nil, nil, fmt.Errorf("%w: username must be 3-32 chars using a-z, 0-9, _ or -", ErrValidation)
	}
	if len(input.Password) < 8 || len(input.Password) > 128 {
		return nil, nil, fmt.Errorf("%w: password must be 8-128 chars", ErrValidation)
	}
	if displayName == "" {
		displayName = username
	}
	if len(displayName) > 64 {
		return nil, nil, fmt.Errorf("%w: display name must be at most 64 chars", ErrValidation)
	}

	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	if count > 0 {
		return nil, nil, domain.ErrRegistrationClosed
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	id, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("generate user id: %w", err)
	}
	user := &domain.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Status:       domain.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	tokens, err := s.createSession(ctx, user.ID, meta)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput, meta ClientMeta) (*domain.User, *TokenPair, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	if username == "" || input.Password == "" {
		return nil, nil, ErrInvalidCredentials
	}

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !user.IsActive() {
		return nil, nil, ErrInvalidCredentials
	}

	ok, err := s.hasher.Verify(input.Password, user.PasswordHash)
	if err != nil || !ok {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.createSession(ctx, user.ID, meta)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, meta ClientMeta) (*domain.User, *TokenPair, error) {
	if refreshToken == "" {
		return nil, nil, ErrUnauthorized
	}

	session, err := s.sessions.FindByRefreshTokenHash(ctx, security.HashToken(refreshToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if !session.IsRefreshValid(time.Now().UTC()) {
		return nil, nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if !user.IsActive() {
		return nil, nil, ErrUnauthorized
	}

	accessToken, refreshTokenNew, accessExpiresAt, refreshExpiresAt, err := s.generateTokens()
	if err != nil {
		return nil, nil, err
	}
	if err := s.sessions.Rotate(
		ctx,
		session.ID,
		security.HashToken(accessToken),
		security.HashToken(refreshTokenNew),
		accessExpiresAt,
		refreshExpiresAt,
	); err != nil {
		return nil, nil, err
	}

	return user, &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshTokenNew,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, accessToken string) (*domain.User, *domain.Session, error) {
	if accessToken == "" {
		return nil, nil, ErrUnauthorized
	}

	session, err := s.sessions.FindByAccessTokenHash(ctx, security.HashToken(accessToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if !session.IsAccessValid(time.Now().UTC()) {
		return nil, nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	if !user.IsActive() {
		return nil, nil, ErrUnauthorized
	}

	_ = s.sessions.Touch(ctx, session.ID)
	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	err := s.sessions.Revoke(ctx, sessionID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Service) createSession(ctx context.Context, userID uuid.UUID, meta ClientMeta) (*TokenPair, error) {
	accessToken, refreshToken, accessExpiresAt, refreshExpiresAt, err := s.generateTokens()
	if err != nil {
		return nil, err
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}
	now := time.Now().UTC()
	session := &domain.Session{
		ID:               sessionID,
		UserID:           userID,
		AccessTokenHash:  security.HashToken(accessToken),
		RefreshTokenHash: security.HashToken(refreshToken),
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		UserAgent:        meta.UserAgent,
		ClientIP:         meta.ClientIP,
		CreatedAt:        now,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *Service) generateTokens() (string, string, time.Time, time.Time, error) {
	accessToken, err := security.GenerateToken()
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	refreshToken, err := security.GenerateToken()
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	now := time.Now().UTC()
	return accessToken, refreshToken, now.Add(s.accessTTL), now.Add(s.refreshTTL), nil
}
