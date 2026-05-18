package main

import (
	"awesomeProject/databaseutil"
	"awesomeProject/handlerutil"
	"awesomeProject/internal"
	"awesomeProject/internal/auth"
	"awesomeProject/internal/bookmark"
	"awesomeProject/internal/form"
	"awesomeProject/internal/jwt"
	"awesomeProject/internal/user"
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const baseURL = "http://localhost:8080"

func main() {
	_ = godotenv.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info("Starting backend service")

	err = databaseutil.MigrationUp("file:///Users/chenxinan/GolandProjects/awesomeProject/internal/database/migrations", "postgresql://postgres:password@localhost:5433/postgres?sslmode=disable", logger)
	if err != nil {
		logger.Fatal("Failed to run database migration", zap.Error(err))
	}

	poolConfig, err := pgxpool.ParseConfig("postgresql://postgres:password@localhost:5433/postgres?sslmode=disable")
	if err != nil {
		logger.Fatal("Failed to parse database URL", zap.Error(err))
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Failed to create database connection pool", zap.Error(err))
	}
	defer dbPool.Close()

	validator := internal.NewValidator()

	formService := form.NewService(logger, dbPool)
	userService := user.NewService(logger, dbPool)
	jwtService := jwt.NewService(logger, 15*time.Minute, 30*time.Minute, dbPool)
	//userStore := user.NewService(logger, dbPool)
	bookmarkService := bookmark.NewService(logger, dbPool)

	formHandler := form.NewHandler(logger, validator, formService)
	authHandler := auth.NewHandler(logger, baseURL, jwtService, userService)
	userHandler := user.NewHandler(logger, userService)
	jwtHandler := jwt.NewHandler(logger, jwtService)
	bookmarkHandler := bookmark.NewHandler(logger, bookmarkService)

	basicMiddleware := handlerutil.NewMiddleware(logger, true)
	jwtMiddleware := jwt.NewMiddleware(logger, jwtService)

	mux := http.NewServeMux()

	// ===== Form 相關 API（CRUD）=====

	// 建立表單：需要登入（JWT 驗證）
	// Recover → JWT 驗證 → form.Handler.Create
	mux.HandleFunc("POST /api/forms", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.Create)))

	// 列出所有表單：公開，不需登入
	// Recover → form.Handler.List
	mux.HandleFunc("GET /api/forms", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.List)))

	// 更新指定表單（{id} 為路徑參數）：需要登入
	// Recover → JWT 驗證 → form.Handler.Update
	mux.HandleFunc("PUT /api/forms/{id}", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.Update)))

	// 刪除指定表單：需要登入
	// Recover → JWT 驗證 → form.Handler.Delete
	mux.HandleFunc("DELETE /api/forms/{id}", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.Delete)))

	// ===== User 相關 API =====

	// 建立使用者：公開（註冊用），不需登入
	// Recover → user.Handler.Create
	mux.HandleFunc("POST /api/users", basicMiddleware.RecoverMiddleware(userHandler.Create))

	// ===== OAuth 登入流程 =====

	// 步驟 1：使用者點「用 Google 登入」打這支 API
	// 後端產生 Google 授權 URL 並 302 重導使用者到 Google 登入頁
	// {provider} = "google"
	// Recover → auth.Handler.Login
	mux.HandleFunc("GET /api/oauth/{provider}", basicMiddleware.RecoverMiddleware(authHandler.Login))

	// 步驟 2：Google 登入完成後，會帶 ?code=xxx 回呼這支 API
	// 後端用 code 換 Google token → 拿 user info → 建立 user → 簽自己的 JWT → 重導回前端
	// Recover → auth.Handler.Callback
	mux.HandleFunc("GET /api/oauth/{provider}/callback", basicMiddleware.RecoverMiddleware(authHandler.Callback))

	// 步驟 3（可選）：debug 用的預設 redirect 目的地
	// 當 Login 沒帶 ?c= 參數時，Callback 結束會把 token 丟到這裡
	// Recover → auth.Handler.DebugToken
	mux.HandleFunc("GET /api/oauth/debug/token", basicMiddleware.RecoverMiddleware(authHandler.DebugToken))

	// ===== Token Refresh =====

	// 帶舊的 refresh_token 換新的 access_token + refresh_token
	// Recover → jwt.Handler.Refresh
	mux.HandleFunc("POST /api/token/refresh", basicMiddleware.RecoverMiddleware(jwtHandler.Refresh))

	// ===== Bookmark 相關 API =====
	mux.HandleFunc("POST /api/bookmarks/{form_id}", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(bookmarkHandler.Toggle)))
	mux.HandleFunc("GET /api/bookmarks", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(bookmarkHandler.List)))
	mux.HandleFunc("GET /api/bookmarks/count", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(bookmarkHandler.CountByUserID)))
	mux.HandleFunc("GET /api/bookmarks/{form_id}/count", basicMiddleware.RecoverMiddleware(bookmarkHandler.CountByFormID))
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	logger.Info("Backend started on :8080")

	err = server.ListenAndServe()
	if err != nil {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}
