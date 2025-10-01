package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/fadilmartias/cv-analyzer/internal/config"
	"github.com/fadilmartias/cv-analyzer/internal/domain/fiber/handler"
	"github.com/fadilmartias/cv-analyzer/internal/middleware"
	"github.com/fadilmartias/cv-analyzer/internal/model"
	"github.com/fadilmartias/cv-analyzer/internal/repository"
	"github.com/fadilmartias/cv-analyzer/internal/service"
	"github.com/fadilmartias/cv-analyzer/internal/usecase"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load .env file
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		log.Println("Could not load .env file")
	}

	appConfig := config.LoadAppConfig()

	app := fiber.New(fiber.Config{
		AppName: appConfig.Name,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Status code defaults to 500
			code := fiber.StatusInternalServerError

			// Retrieve the custom status code if it's a *fiber.Error
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			message := err.Error()
			if message == "" {
				message = "Internal Server Error"
			}

			return ctx.Status(code).JSON(fiber.Map{"error": message})
		},
	})
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))
	// Use middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: config.LoadAppConfig().Env != "production",
	}))

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // 1
	}))
	app.Use(pprof.New(pprof.Config{
		Next: func(c *fiber.Ctx) bool {
			return config.LoadAppConfig().Env != "production"
		},
	}))
	app.Use(healthcheck.New())

	app.Use(helmet.New(helmet.Config{
		CrossOriginResourcePolicy: "cross-origin",
	}))

	app.Use(middleware.RateLimiter(50, 1*time.Minute))

	db := ConnectDB()

	jobRepo := repository.NewJobRepository(db)
	evaluationRepo := repository.NewEvaluationRepository(db)
	openRouter := service.NewOpenRouterService()
	gemini, err := service.NewGeminiService(ctx)
	if err != nil {
		log.Fatal(err)
	}
	uc := usecase.NewEvaluationUsecase(evaluationRepo, jobRepo, openRouter, gemini)
	handler := handler.NewEvaluateHandler(uc)

	handler.RegisterRoutes(app)

	// Use context for cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Monitor goroutine count
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Printf("Active goroutines: %d", runtime.NumGoroutine())
		}
	}()

	log.Println("Server running on ", appConfig.Port)
	if err := app.Listen(appConfig.Port); err != nil {
		log.Fatal(err)
	}
}

func ConnectDB() *gorm.DB {
	dbConfig := config.LoadDBConfig()
	appConfig := config.LoadAppConfig()

	// Format DSN untuk MySQL
	// format: "user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		dbConfig.Host,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Name,
		dbConfig.Port,
		dbConfig.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	pgDB, err := db.DB()
	if err != nil {
		log.Fatalf("Could not get database instance: %v", err)
	}
	if appConfig.Env != "production" {
		pgDB.SetMaxIdleConns(5)  // cukup 5 idle
		pgDB.SetMaxOpenConns(10) // max 10 koneksi aktif
		pgDB.SetConnMaxLifetime(30 * time.Minute)
	} else {
		pgDB.SetMaxIdleConns(20)           // simpan 20 koneksi siap pakai
		pgDB.SetMaxOpenConns(200)          // max 200 koneksi aktif
		pgDB.SetConnMaxLifetime(time.Hour) // recycle tiap 1 jam

	}

	// migrasi tabel
	err = db.AutoMigrate(&model.EvaluationTask{}, &model.Job{})
	if err != nil {
		log.Fatal("migration failed: ", err)
	}
	return db
}
