package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/4vertak/redpen-checker/internal/config"
	"github.com/4vertak/redpen-checker/internal/storage"
	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
)

func setupTestDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/redpen_test?sslmode=disable"
	}

	if err := os.Setenv("DATABASE_URL", dsn); err != nil {
		panic("failed to set DATABASE_URL: " + err.Error())
	}

	// Применяем миграции
	m, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		panic("failed to init migrations: " + err.Error())
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		panic("failed to apply migrations: " + err.Error())
	}

	// Инициализируем пул соединений
	if err := storage.Connect(); err != nil {
		panic("failed to connect to test db: " + err.Error())
	}

	// Очищаем таблицу teachers перед каждым прогоном
	ctx := context.Background()
	_, err = storage.Pool.Exec(ctx, "DELETE FROM teachers")
	if err != nil {
		panic("failed to clean teachers: " + err.Error())
	}
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/auth/register", RegisterTeacher)
	r.POST("/api/v1/auth/login", LoginTeacher)
	r.POST("/api/v1/auth/forgot-password", ForgotPassword)
	r.POST("/api/v1/auth/reset-password", ResetPassword)
	return r
}

func TestRegisterTeacher_Success(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	body := map[string]string{
		"name":     "Test Teacher",
		"email":    "test@school.ru",
		"password": "123456",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, body["name"], resp["name"])
	assert.Equal(t, body["email"], resp["email"])
	assert.NotEmpty(t, resp["id"])
}

func TestRegisterTeacher_DuplicateEmail(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	body := map[string]string{
		"name":     "Another Teacher",
		"email":    "duplicate@school.ru",
		"password": "123456",
	}
	jsonBody, _ := json.Marshal(body)

	// Первый запрос – успех
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Второй запрос с тем же email – ошибка
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterTeacher_InvalidEmail(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	body := map[string]string{
		"name":     "Test",
		"email":    "invalid-email",
		"password": "123456",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterTeacher_ShortPassword(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	body := map[string]string{
		"name":     "Test",
		"email":    "short@school.ru",
		"password": "123",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginTeacher_Success(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	// Сначала регистрируем пользователя
	regBody := map[string]string{
		"name":     "Login Test",
		"email":    "login@school.ru",
		"password": "123456",
	}
	regJSON, _ := json.Marshal(regBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Теперь выполняем вход
	loginBody := map[string]string{
		"email":    "login@school.ru",
		"password": "123456",
	}
	loginJSON, _ := json.Marshal(loginBody)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["access_token"])
}

func TestLoginTeacher_WrongPassword(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	// Регистрируем
	regBody := map[string]string{
		"name":     "Wrong Pass",
		"email":    "wrong@school.ru",
		"password": "123456",
	}
	regJSON, _ := json.Marshal(regBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Вход с неверным паролем
	loginBody := map[string]string{
		"email":    "wrong@school.ru",
		"password": "wrongpass",
	}
	loginJSON, _ := json.Marshal(loginBody)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginTeacher_NonExistentEmail(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	loginBody := map[string]string{
		"email":    "nobody@school.ru",
		"password": "123456",
	}
	loginJSON, _ := json.Marshal(loginBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestForgotAndResetPassword_Success(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	// 1. Регистрируем пользователя
	regBody := map[string]string{
		"name":     "Reset Test",
		"email":    "reset@school.ru",
		"password": "oldpass",
	}
	regJSON, _ := json.Marshal(regBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(regJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 2. Запрашиваем сброс пароля
	forgotBody := map[string]string{"email": "reset@school.ru"}
	forgotJSON, _ := json.Marshal(forgotBody)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/forgot-password", bytes.NewBuffer(forgotJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// В реальном сценарии токен был бы отправлен на email, но мы его логируем.
	// Для теста мы просто генерируем токен вручную, повторяя логику обработчика.
	// Тестируем сам сценарий, а не способ доставки.
	cfg := config.Load()
	claims := jwt.MapClaims{
		"teacher_id": "fake-id",
		"purpose":    "password_reset",
		"exp":        time.Now().Add(10 * time.Minute).Unix(),
		"iat":        time.Now().Unix(),
	}

	var teacherID string
	err := storage.Pool.QueryRow(
		context.Background(),
		`SELECT id FROM teachers WHERE email = $1`,
		"reset@school.ru",
	).Scan(&teacherID)
	assert.NoError(t, err)
	claims["teacher_id"] = teacherID
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	assert.NoError(t, err)

	// 3. Сбрасываем пароль с токеном
	resetBody := map[string]string{
		"token":        tokenString,
		"new_password": "newpass123",
	}
	resetJSON, _ := json.Marshal(resetBody)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(resetJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Входим с новым паролем
	loginBody := map[string]string{
		"email":    "reset@school.ru",
		"password": "newpass123",
	}
	loginJSON, _ := json.Marshal(loginBody)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["access_token"])

	// 5. Старый пароль больше не работает
	loginBodyOld := map[string]string{
		"email":    "reset@school.ru",
		"password": "oldpass",
	}
	loginOldJSON, _ := json.Marshal(loginBodyOld)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginOldJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
