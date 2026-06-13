package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/4vertak/redpen-checker/internal/handler"
	"github.com/4vertak/redpen-checker/internal/storage"
)

func runMigrations() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/redpen?sslmode=disable"
	}
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Fatalf("Ошибка инициализации миграций: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Ошибка применения миграций: %v", err)
	}
	log.Println("Миграции успешно применены")
}

func main() {
	if err := storage.Connect(); err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer storage.Close()

	runMigrations()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/api/v1/auth/register", handler.RegisterTeacher)

	log.Printf("Сервер запущен на порту %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}