// Aphrodite API server — bootstrap and dependency wiring only.
//
// @title           Aphrodite API
// @version         0.1.0
// @description     Go BFF for users, business workflows, and localized Strapi editorial content.
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Aphrodite session token from POST /v1/users/login — send as "Bearer <token>"
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	commentpg "aphrodite/internal/comment/infra/postgres"
	commenttransport "aphrodite/internal/comment/transport/http"
	commentuc "aphrodite/internal/comment/usecase"
	"aphrodite/internal/content"
	contentstrapi "aphrodite/internal/content/strapi"
	contenttransport "aphrodite/internal/content/transport/http"
	postpg "aphrodite/internal/post/infra/postgres"
	posttransport "aphrodite/internal/post/transport/http"
	postuc "aphrodite/internal/post/usecase"
	"aphrodite/internal/shared/httpx/handlers"
	"aphrodite/internal/shared/httpx/middleware"
	sharedpg "aphrodite/internal/shared/postgres"
	sharedredis "aphrodite/internal/shared/redis"
	usercrypto "aphrodite/internal/user/infra/crypto"
	userpg "aphrodite/internal/user/infra/postgres"
	usertoken "aphrodite/internal/user/infra/token"
	usertransport "aphrodite/internal/user/transport/http"
	useruc "aphrodite/internal/user/usecase"
	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"

	_ "aphrodite/docs"
)

func main() {
	os.Exit(run())
}

type redisPinger interface {
	Ping(context.Context) error
}

type runDeps struct {
	loadConfig                 func()
	initLogger                 func(debug bool)
	connectPostgres            func(config.DatabaseConfig) (*gorm.DB, error)
	autoMigrate                func(*gorm.DB) error
	waitForStrapi              func(context.Context, config.StrapiConfig) error
	newRedis                   func(config.RedisConfig) (redisPinger, error)
	notifyContext              func(context.Context, ...os.Signal) (context.Context, context.CancelFunc)
	listenAndServe             func(*http.Server) error
	exitOnUnexpectedServeError func(int)
}

func defaultRunDeps() runDeps {
	return runDeps{
		loadConfig:      config.Load,
		initLogger:      logger.Init,
		connectPostgres: sharedpg.Connect,
		autoMigrate:     autoMigrate,
		waitForStrapi:   waitForStrapi,
		newRedis: func(cfg config.RedisConfig) (redisPinger, error) {
			return sharedredis.New(cfg)
		},
		notifyContext: signal.NotifyContext,
		listenAndServe: func(srv *http.Server) error {
			return srv.ListenAndServe()
		},
		exitOnUnexpectedServeError: os.Exit,
	}
}

func run() int {
	return runWithDeps(defaultRunDeps())
}

func runWithDeps(deps runDeps) int {
	deps.loadConfig()
	if deps.initLogger != nil {
		deps.initLogger(config.C.Debug)
	}

	// ── Platform adapters ───────────────────────────────────────────────────────
	db, err := deps.connectPostgres(config.C.Database)
	if err != nil {
		slog.Error("postgres connection failed", "err", err)
		return 1
	}

	if migrate := deps.autoMigrate; migrate != nil {
		if err := migrate(db); err != nil {
			slog.Error("application schema initialization failed", "err", err)
			return 1
		}
	} else if err := autoMigrate(db); err != nil {
		slog.Error("application schema initialization failed", "err", err)
		return 1
	}

	if config.C.Strapi.BaseURL != "" {
		wait := deps.waitForStrapi
		if wait == nil {
			wait = waitForStrapi
		}
		if err := wait(context.Background(), config.C.Strapi); err != nil {
			slog.Error("strapi readiness failed", "err", err)
			return 1
		}
	}

	redisClient, err := deps.newRedis(config.C.Redis)
	if err != nil {
		slog.Error("redis connection failed", "err", err)
		return 1
	}

	var contentHandler *contenttransport.Handler
	if config.C.Strapi.BaseURL != "" {
		contentClient, err := contentstrapi.New(contentstrapi.ConfigFromApp(config.C.Strapi))
		if err != nil {
			slog.Error("strapi client setup failed", "err", err)
			return 1
		}
		var contentReader content.SlugReader = contentClient
		if cache, ok := redisClient.(interface {
			Get(context.Context, string) (string, error)
			Set(context.Context, string, interface{}, time.Duration) error
		}); ok {
			contentReader = content.NewCachedSlugReader(contentClient, cache, config.C.Strapi.CacheTTL)
		}
		contentHandler = contenttransport.NewHandler(contentReader, config.C.Strapi.DefaultLocale, config.C.Strapi.FallbackLocale)
	}

	// ── user context ────────────────────────────────────────────────────────────
	userRepo := userpg.New(db)
	sessionRepo := userpg.NewSessionRepository(db)
	hasher := usercrypto.NewBcryptHasher(config.C.Auth.BcryptCost)
	tokens, err := usertoken.NewJWT(usertoken.JWTConfig{
		Secret:         config.C.Auth.JWTSecret,
		TTL:            config.C.Auth.AccessTokenTTL,
		MinSecretBytes: config.C.Auth.JWTMinSecretBytes,
	})
	if err != nil {
		slog.Error("token issuer setup failed", "err", err)
		return 1
	}

	registerUser := useruc.NewRegisterUser(userRepo, hasher, config.C.Auth.SuperAdminKey, nil, nil)
	authenticateUser := useruc.NewAuthenticateUser(userRepo, hasher, tokens)
	getUserProfile := useruc.NewGetUserProfile(userRepo)
	listUsers := useruc.NewListUsers(userRepo, useruc.ListConfig{
		DefaultLimit: config.C.Pagination.UserDefaultLimit,
		MaxLimit:     config.C.Pagination.UserMaxLimit,
	})
	updateUser := useruc.NewUpdateUser(userRepo, nil)
	changePassword := useruc.NewChangePassword(userRepo, hasher, nil)
	userHandler := usertransport.NewHandler(registerUser, authenticateUser, getUserProfile, listUsers, updateUser, changePassword)
	cmsSessions := useruc.NewCMSSessionManager(userRepo, hasher, sessionRepo, config.C.Session.TTL)
	cmsHandler := usertransport.NewCMSHandler(cmsSessions, config.C.Session.CookieName, config.C.Session.TTL, config.C.Session.Secure)

	// ── post context ────────────────────────────────────────────────────────────
	postRepo := postpg.New(db)
	postHandler := posttransport.NewHandler(
		postuc.NewCreatePost(postRepo, postuc.CreatePostConfig{
			TitleMaxLength:   config.C.Validation.PostTitleMaxLength,
			ContentMaxLength: config.C.Validation.PostContentMaxLength,
		}, nil, nil),
		postuc.NewGetPost(postRepo),
		postuc.NewListPosts(postRepo, postuc.ListConfig{
			DefaultLimit: config.C.Pagination.PostDefaultLimit,
			MaxLimit:     config.C.Pagination.PostMaxLimit,
		}),
		postuc.NewUpdatePost(postRepo, postuc.UpdatePostConfig{
			TitleMaxLength:   config.C.Validation.PostTitleMaxLength,
			ContentMaxLength: config.C.Validation.PostContentMaxLength,
		}, nil),
		postuc.NewDeletePost(postRepo),
	)

	// ── comment context ─────────────────────────────────────────────────────────
	commentRepo := commentpg.New(db)
	commentHandler := commenttransport.NewHandler(
		commentuc.NewAddComment(commentRepo, commentuc.AddCommentConfig{
			ContentMaxLength: config.C.Validation.CommentContentMaxLength,
		}, nil, nil),
		commentuc.NewListComments(commentRepo, commentuc.ListConfig{
			DefaultLimit: config.C.Pagination.CommentDefaultLimit,
			MaxLimit:     config.C.Pagination.CommentMaxLimit,
		}),
		commentuc.NewUpdateComment(commentRepo, commentuc.UpdateCommentConfig{
			ContentMaxLength: config.C.Validation.CommentContentMaxLength,
		}, nil),
		commentuc.NewDeleteComment(commentRepo),
	)

	// ── HTTP server ─────────────────────────────────────────────────────────────
	if !config.C.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger())

	healthHandler := handlers.NewHealthHandler(db, redisClient)
	r.GET("/healthz", healthHandler.Check)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authMW := usertransport.AuthMiddleware(tokens)

	v1 := r.Group("/v1")
	usertransport.Register(v1, userHandler, tokens)
	usertransport.RegisterCMS(v1, cmsHandler)
	if contentHandler != nil {
		contenttransport.RegisterRoutes(v1, contentHandler)
	}
	posttransport.Register(v1, postHandler, authMW)
	commenttransport.Register(v1, commentHandler, authMW)

	srv := &http.Server{
		Addr:         ":" + config.C.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := deps.notifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", config.C.Port, "env", config.C.Env)
		if err := deps.listenAndServe(srv); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			deps.exitOnUnexpectedServeError(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "err", err)
	}
	slog.Info("server stopped")
	return 0
}

func autoMigrate(db *gorm.DB) error {
	for _, step := range []struct {
		name string
		fn   func() error
	}{
		{"user", func() error { return userpg.AutoMigrate(db) }},
		{"post", func() error { return postpg.AutoMigrate(db) }},
		{"comment", func() error { return commentpg.AutoMigrate(db) }},
	} {
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s schema: %w", step.name, err)
		}
	}
	return nil
}
