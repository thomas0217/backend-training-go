package jwt

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const secret = "default_secret"

type Service struct {
	logger                 *zap.Logger
	expiration             time.Duration
	refreshTokenExpiration time.Duration
	querier                Querier // 操作 refresh_tokens 表的介面
}

type Querier interface {
	// 從sql.go裡面的method
	Create(ctx context.Context, arg CreateParams) (RefreshToken, error)
	Use(ctx context.Context, id uuid.UUID) (RefreshToken, error)
}

func NewService(logger *zap.Logger, expiration, refreshExp time.Duration, db DBTX) *Service {
	return &Service{
		logger:                 logger,
		expiration:             expiration,
		refreshTokenExpiration: refreshExp,
		querier:                New(db),
	}
}

type claims struct {
	Message string
	ID      string
	Email   string
	UserID  uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// 簽access token(jwt)
func (s *Service) New(ctx context.Context, userID uuid.UUID) (string, error) {
	jwtID := uuid.New()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID:  userID,
		Message: "This is a Backend-Training JWT token",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "Backend-Training",
			Subject:   "Backend-Training Token",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        jwtID.String(),
		},
	})

	// 用 secret 對 token 簽章 + Base64Url 編碼成 "xxx.yyy.zzz" 格式字串
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		s.logger.Error("Failed to sign token", zap.Error(err))
		return "", err
	}

	s.logger.Debug("Generated new JWT token", zap.String("user_id", userID.String()))
	return tokenString, nil
}

// Parse：驗證 access token，成功後回傳 claims 裡帶的 user id
func (s *Service) Parse(ctx context.Context, tokenString string) (uuid.UUID, error) {
	// Bearer xxx
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// 解讀token 含header, claims, signature
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		// 不同錯誤分開 log，方便除錯與監控
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			s.logger.Warn("Failed to parse JWT token due to malformed structure, this is not a JWT token", zap.String("error", err.Error()))
			return uuid.Nil, err
		case errors.Is(err, jwt.ErrSignatureInvalid):
			s.logger.Warn("Failed to parse JWT token due to invalid signature", zap.String("error", err.Error()))
			return uuid.Nil, err
		case errors.Is(err, jwt.ErrTokenExpired):
			expiredTime, getErr := token.Claims.GetExpirationTime()
			if getErr != nil {
				s.logger.Warn("Failed to parse JWT token due to expired timestamp", zap.String("error", err.Error()))
			} else {
				s.logger.Warn("Failed to parse JWT token due to expired timestamp", zap.String("error", err.Error()), zap.Time("expired_at", expiredTime.Time))
			}
			return uuid.Nil, err
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			notBeforeTime, getErr := token.Claims.GetNotBefore()
			if getErr != nil {
				s.logger.Warn("Failed to parse JWT token due to not valid yet timestamp", zap.String("error", err.Error()))
			} else {
				s.logger.Warn("Failed to parse JWT token due to not valid yet timestamp", zap.String("error", err.Error()), zap.Time("not_valid_yet", notBeforeTime.Time))
			}
			return uuid.Nil, err
		default:
			s.logger.Error("Failed to parse or validate JWT token", zap.Error(err))
			return uuid.Nil, err
		}
	}
	//type assertion c應該要是*claims型態 如果是的話才能用 c.UserID 等
	c, ok := token.Claims.(*claims)
	if !ok {
		s.logger.Warn("Invalid JWT token claims")
		return uuid.Nil, errors.New("invalid token claims")
	}

	s.logger.Debug("Parsed JWT token successfully", zap.String("user_id", c.UserID.String()))
	return c.UserID, nil
}

// NewRefreshToken：產生一張 refresh token 並寫進 DB
func (s *Service) NewRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	expiration := time.Now().Add(s.refreshTokenExpiration)

	// 呼叫 sqlc 產的 Create()
	token, err := s.querier.Create(ctx, CreateParams{
		UserID:         userID,
		ExpirationTime: pgtype.Timestamptz{Time: expiration, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create refresh token", zap.Error(err))
		return "", err
	}

	s.logger.Debug("Generated new refresh token", zap.String("user_id", userID.String()))
	// token.ID 是 DB 用 gen_random_uuid() 產的 UUID，把它轉成字串就是 refresh token
	return token.ID.String(), nil
}

// Refresh：用舊的 refresh token 換一組新的 access + refresh token
func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, string, error) {

	id, err := uuid.Parse(refreshToken)
	if err != nil {
		s.logger.Warn("Invalid refresh token format", zap.String("token", refreshToken))
		return "", "", err
	}

	// 標記為用過 use回傳RefreshToken
	row, err := s.querier.Use(ctx, id)
	if err != nil {
		s.logger.Warn("Refresh token invalid or expired", zap.Error(err))
		return "", "", err
	}

	// 新的access token 這個token綁定原本來換token的user
	accessToken, err := s.New(ctx, row.UserID)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := s.NewRefreshToken(ctx, row.UserID)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}
