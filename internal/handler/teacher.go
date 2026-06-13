package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/4vertak/redpen-checker/internal/config"
	"github.com/4vertak/redpen-checker/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func RegisterTeacher(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	var teacher struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	err = storage.Pool.QueryRow(
		c.Request.Context(),
		`INSERT INTO teachers (name, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, email`,
		req.Name, req.Email, string(hashedPassword),
	).Scan(&teacher.ID, &teacher.Name, &teacher.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Учитель с таким email уже существует"})
		return
	}
	c.JSON(http.StatusCreated, teacher)
}

func LoginTeacher(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var teacher struct {
		ID           string
		PasswordHash string
	}
	err := storage.Pool.QueryRow(
		c.Request.Context(),
		`SELECT id, password_hash FROM teachers WHERE email = $1`,
		req.Email,
	).Scan(&teacher.ID, &teacher.PasswordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(teacher.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	claims := jwt.MapClaims{
		"teacher_id": teacher.ID,
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	cfg := config.Load()
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": tokenString})
}

func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Если такой email зарегистрирован, на него отправлена ссылка для сброса пароля"})

	// Проверяем, существует ли учитель
	var teacherID string
	err := storage.Pool.QueryRow(
		context.Background(),
		`SELECT id FROM teachers WHERE email = $1`,
		req.Email,
	).Scan(&teacherID)
	if err != nil {
		// Пользователь не найден – ничего не делаем
		return
	}

	// Генерируем временный токен (10 минут)
	cfg := config.Load()
	claims := jwt.MapClaims{
		"teacher_id": teacherID,
		"purpose":    "password_reset",
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
		"iat":        time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		log.Printf("Ошибка создания токена сброса для %s: %v", req.Email, err)
		return
	}

	// Временно выводим токен в логи
	log.Printf("Ссылка для сброса пароля для %s: http://localhost:8080/api/v1/auth/reset-password?token=%s", req.Email, tokenString)
}

func ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Парсим токен
	cfg := config.Load()
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недействительный или истёкший токен"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["purpose"] != "password_reset" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недействительный токен"})
		return
	}

	teacherID, ok := claims["teacher_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недействительный токен"})
		return
	}

	// Хешируем новый пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	// Обновляем пароль
	_, err = storage.Pool.Exec(
		context.Background(),
		`UPDATE teachers SET password_hash = $1, updated_at = now() WHERE id = $2`,
		string(hashedPassword), teacherID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления пароля"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменён"})
}
