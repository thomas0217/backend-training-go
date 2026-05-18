package auth

import (
	"awesomeProject/internal/auth/oauthprovider"
	"awesomeProject/internal/user"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type OAuthProvider interface {
	Name() string
	Config() *oauth2.Config
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (oauthprovider.UserInfo, error)
}

// jwt/service.go裡面實作的function
type jwtService interface {
	New(ctx context.Context, userID uuid.UUID) (string, error)
	NewRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)
}

// user/service.go裡面實作的function
type UserStore interface {
	ExistsUser(ctx context.Context, email string) (bool, error)
	Create(ctx context.Context, email string) (user.User, error)
	GetByEmail(ctx context.Context, email string) (user.User, error)
}
type Handler struct {
	logger     *zap.Logger
	baseURL    string
	jwtService jwtService
	provider   map[string]OAuthProvider
	userStore  UserStore
}

func NewHandler(logger *zap.Logger, baseURL string, jwtService jwtService, userStore UserStore) *Handler {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	return &Handler{
		logger:     logger,
		baseURL:    baseURL,
		jwtService: jwtService,
		userStore:  userStore,
		provider: map[string]OAuthProvider{
			"google": oauthprovider.NewGoogleConfig(
				clientID,
				clientSecret,
				fmt.Sprintf("%s/api/oauth/google/callback", baseURL)),
		},
	}
}

// 導向google登入網站
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	provider := h.provider[providerName]
	if provider == nil {
		h.logger.Warn("No such provider", zap.String("provider", providerName))
		http.Error(w, "Unsupported OAuth2 provider", http.StatusBadRequest)
		return
	}
	//GET /api/oauth/google?c=http://localhost:3000/auth-receiver&r=/cart
	//c=http://localhost:3000/auth-receiver 接收access.refresh token的地方
	//r = /cart google登入完回去的地方
	redirectTo := r.URL.Query().Get("c")
	frontendRedirectTo := r.URL.Query().Get("r")
	if redirectTo == "" {
		redirectTo = fmt.Sprintf("%s/api/oauth/debug/token", h.baseURL)
	}
	//讓之後的 Callback 階段知道要把產出的 Token 送往哪裡。
	if frontendRedirectTo != "" {
		redirectTo = fmt.Sprintf("%s?r=%s", redirectTo, frontendRedirectTo)
	}
	//redirect
	authURL := provider.Config().AuthCodeURL(redirectTo, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	h.logger.Info("Redirecting to Google OAuth2", zap.String("url", authURL))
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	provider := h.provider[providerName]
	if provider == nil {
		h.logger.Warn("No such provider", zap.String("provider", providerName))
		http.Error(w, "Unsupported OAuth2 provider", http.StatusBadRequest)
		return
	}
	//把原本login的時候寫好的位址拿出來
	state := r.URL.Query().Get("state")
	redirectTo := state
	if redirectTo == "" {
		redirectTo = fmt.Sprintf("%s/api/oauth/debug/token", h.baseURL)
	}

	authError := r.URL.Query().Get("error")
	if authError != "" {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, authError)
		h.logger.Warn("OAuth2 callback returned error", zap.String("error", authError))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	//callback回來時拿到的code
	code := r.URL.Query().Get("code")
	if code == "" {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, "missing_code")
		h.logger.Warn("Missing code in callback")
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	//code換token
	token, err := provider.Exchange(r.Context(), code)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to exchange code for token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	//用token拿userInfo
	userInfo, err := provider.GetUserInfo(r.Context(), token)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to get user info", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	// 確認使用者存在 不存在就建立
	haveUser, err := h.userStore.ExistsUser(r.Context(), userInfo.Email)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to check if user exists", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	var currentUser user.User
	if !haveUser {
		currentUser, err = h.userStore.Create(r.Context(), userInfo.Email)
		if err != nil {
			redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
			h.logger.Error("Failed to create user", zap.Error(err))
			http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
			return
		}
	} else {
		currentUser, err = h.userStore.GetByEmail(r.Context(), userInfo.Email)
		if err != nil {
			redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
			h.logger.Error("Failed to get user", zap.Error(err))
			http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
			return
		}
	}

	// 簽 access token
	accessToken, err := h.jwtService.New(r.Context(), currentUser.ID)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to create access token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	// 簽 refresh token
	refreshToken, err := h.jwtService.NewRefreshToken(r.Context(), currentUser.ID)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to create refresh token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	redirectTo = fmt.Sprintf("%s?access_token=%s&refresh_token=%s", redirectTo, accessToken, refreshToken)

	http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
	h.logger.Info("OAuth2 callback successful", zap.String("user_email", userInfo.Email))
}

func (h *Handler) DebugToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte(`{"message":"Login successful"}`))
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
