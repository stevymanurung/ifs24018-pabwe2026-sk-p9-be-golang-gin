package auth

import (
	"errors"
	"net/mail"
	"strings"

	"app/internal/database"
	"app/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("input tidak valid")
	ErrDuplicateUser      = errors.New("username atau email sudah terdaftar")
	ErrInvalidCredentials = errors.New("kredensial tidak valid")
	ErrUserNotFound       = errors.New("pengguna tidak ditemukan")
)

type Service interface {
	Register(req *RegisterRequest) (*AuthResponse, error)
	Login(req *LoginRequest) (*AuthResponse, error)
	GetProfile(userID uint) (*database.User, error)
	UpdateProfile(userID uint, req *UpdateProfileRequest) (*database.User, error)
}

type service struct {
	repo      Repository
	secretKey string
}

func NewService(repo Repository, secretKey string) Service {
	return &service{
		repo:      repo,
		secretKey: secretKey,
	}
}

func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return false
	}
	return strings.Contains(parts[1], ".")
}

func (s *service) Register(req *RegisterRequest) (*AuthResponse, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" || !isValidEmail(req.Email) || len(req.Password) < 6 {
		return nil, ErrInvalidInput
	}

	// Role validation: only "admin" or "member", otherwise forced to "member"
	role := "member"
	if strings.EqualFold(req.Role, "admin") {
		role = "admin"
	}

	// Duplicate check
	if existing, _ := s.repo.FindByUsername(req.Username); existing != nil {
		return nil, ErrDuplicateUser
	}
	if existing, _ := s.repo.FindByEmail(req.Email); existing != nil {
		return nil, ErrDuplicateUser
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     role,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	token, _ := middleware.GenerateToken(user.ID, user.Username, user.Email, user.Role, s.secretKey)

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		},
	}, nil
}

func (s *service) Login(req *LoginRequest) (*AuthResponse, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.repo.FindByUsernameOrEmail(req.Username)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, _ := middleware.GenerateToken(user.ID, user.Username, user.Email, user.Role, s.secretKey)

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		},
	}, nil
}

func (s *service) GetProfile(userID uint) (*database.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *service) UpdateProfile(userID uint, req *UpdateProfileRequest) (*database.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if req.Email != "" {
		req.Email = strings.TrimSpace(req.Email)
		if !isValidEmail(req.Email) {
			return nil, ErrInvalidInput
		}
		if existing, _ := s.repo.FindByEmail(req.Email); existing != nil && existing.ID != userID {
			return nil, ErrDuplicateUser
		}
		user.Email = req.Email
	}

	if req.Password != "" {
		if len(req.Password) < 6 {
			return nil, ErrInvalidInput
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user.Password = string(hashed)
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}
