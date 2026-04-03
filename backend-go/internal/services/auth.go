package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/timlzh/ollama-hack/internal/config"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type AuthService struct {
	db  *database.DB
	cfg *config.Config
}

func NewAuthService(db *database.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(username, password string) (*models.TokenResponse, error) {
	var user models.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE username = $1", username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.HashedPassword) {
		return nil, errors.New("invalid credentials")
	}

	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
	}, nil
}

func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(time.Duration(s.cfg.App.AccessTokenExpireMinutes) * time.Minute)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.App.SecretKey))
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.App.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *AuthService) GetUserByID(userID int) (*models.User, error) {
	var user models.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE id = $1", userID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) GetUserByAPIKey(apiKey string) (*models.User, error) {
	if s.db == nil {
		return nil, errors.New("invalid API key")
	}
	var user models.User
	query := `
		SELECT u.* FROM users u
		INNER JOIN api_keys ak ON u.id = ak.user_id
		WHERE ak.key = $1
	`
	err := s.db.Get(&user, query, apiKey)
	if err != nil {
		return nil, errors.New("invalid API key")
	}

	// Update last_used_at
	_, _ = s.db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE key = $1", apiKey)

	return &user, nil
}
