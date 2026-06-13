package handler

import (
	"net/http"

	"github.com/4vertak/redpen-checker/internal/storage"

	"github.com/gin-gonic/gin"
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