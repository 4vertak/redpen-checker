package handler

import (
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