package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/auth"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// bcrypt ignores input beyond 72 bytes, see GenerateFromPassword comment.
const maxPasswordBytes = 72

type AuthService struct {
	hospitals  repository.HospitalRepository
	staff      repository.StaffRepository
	tokens     auth.TokenManager
	bcryptCost int
	// Prevent enumeration.
	dummyHash []byte
}

type RegisterInput struct {
	Username     string
	Password     string
	HospitalCode string
}

type LoginInput struct {
	Username     string
	Password     string
	HospitalCode string
}

type LoginResult struct {
	Token     string
	ExpiresIn time.Duration
}

func NewAuthService(
	hospitals repository.HospitalRepository,
	staff repository.StaffRepository,
	tokens auth.TokenManager,
	bcryptCost int,
) *AuthService {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcryptCost)
	if err != nil {
		panic(fmt.Sprintf("generate dummy hash: %v", err))
	}

	return &AuthService{
		hospitals:  hospitals,
		staff:      staff,
		tokens:     tokens,
		bcryptCost: bcryptCost,
		dummyHash:  dummyHash,
	}
}

func (s *AuthService) Register(ctx context.Context, inp RegisterInput) (*model.Staff, error) {
	if len(inp.Password) > maxPasswordBytes {
		return nil, apperr.Validation(fmt.Sprintf("password must be at most %d bytes", maxPasswordBytes))
	}

	hospital, err := s.findHospital(ctx, inp.HospitalCode)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(inp.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	st := &model.Staff{
		HospitalID:   hospital.ID,
		Username:     strings.TrimSpace(inp.Username),
		PasswordHash: string(hash),
	}
	if err := s.staff.Create(ctx, st); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, apperr.ErrUsernameTaken
		}
		return nil, err
	}

	return st, nil
}

func (s *AuthService) Login(ctx context.Context, inp LoginInput) (*LoginResult, error) {
	hospital, err := s.findHospital(ctx, inp.HospitalCode)
	if err != nil {
		return nil, err
	}

	st, err := s.staff.GetByUsername(ctx, hospital.ID, strings.TrimSpace(inp.Username))
	if errors.Is(err, repository.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(inp.Password))
		return nil, apperr.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(st.PasswordHash), []byte(inp.Password)); err != nil {
		return nil, apperr.ErrInvalidCredentials
	}

	token, expiresIn, err := s.tokens.Sign(st.ID, st.HospitalID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, ExpiresIn: expiresIn}, nil
}

func (s *AuthService) findHospital(ctx context.Context, code string) (*model.Hospital, error) {
	h, err := s.hospitals.GetByCode(ctx, strings.TrimSpace(code))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperr.ErrHospitalNotFound
	}
	if err != nil {
		return nil, err
	}

	return h, nil
}
