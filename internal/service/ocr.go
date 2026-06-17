package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"github.com/4vertak/redpen-checker/internal/config"
)

const (
	handwrittenModel    = "handwritten"
	pageModel           = "page"
	confidenceThreshold = 0.70 // СНИЖЕН до 0.70 для детского почерка
	minLineLength       = 2    // СНИЖЕН до 2
)

// RecognizeText предобрабатывает изображение и отправляет в Yandex Vision OCR.
func RecognizeText(ctx context.Context, imagePath string) (string, error) {

	cfg := config.Load()

	if cfg.YandexVisionAPIKey == "" || cfg.YandexFolderID == "" {
		return "", fmt.Errorf("Yandex Vision credentials not configured")
	}

	log.Printf("Начало распознавания: %s", imagePath)

	// 1. Предобработка через Python. Костыль но быстро)
	processedPaths, err := preprocessImage(imagePath)
	if err != nil {
		return "", fmt.Errorf("ошибка предобработки: %w", err)
	}
	log.Printf("Предобработка завершена. Файлов: %d", len(processedPaths))

	// 2. Распознаем каждую страницу
	var allText []string
	for i, procPath := range processedPaths {
		log.Printf("Распознавание страницы %d/%d: %s", i+1, len(processedPaths), procPath)
		text, err := recognizeSingleImage(ctx, cfg, procPath)
		if err != nil {
			log.Printf("Ошибка распознавания %s: %v", procPath, err)
			return "", fmt.Errorf("ошибка распознавания %s: %w", procPath, err)
		}
		log.Printf("Страница %d: получено символов: %d", i+1, len(text))
		if len(text) > 0 {
			allText = append(allText, text)
		}
		os.Remove(procPath)
	}

	if len(allText) == 0 {
		log.Printf("ВНИМАНИЕ: все страницы вернули пустой результат после фильтрации!")
	}

	return mergeAndDeduplicate(allText), nil
}

// preprocessImage вызывает Python-скрипт
func preprocessImage(imagePath string) ([]string, error) {

	pythonCmd := "python"
	if _, err := exec.LookPath(pythonCmd); err != nil {
		pythonCmd = "python3"
	}

	cmd := exec.Command(pythonCmd, "preprocess.py", imagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Python error: %v\nOutput:\n%s", err, string(output))
		return nil, fmt.Errorf("Python-скрипт завершился с ошибкой: %w", err)
	}

	log.Printf("Python output:\n%s", string(output))

	var paths []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OUTPUT:") {
			path := strings.TrimPrefix(line, "OUTPUT:")
			path = strings.TrimSpace(path)
			if path != "" {
				paths = append(paths, path)
			}
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("Python-скрипт не вернул путей. Проверь лог выше")
	}

	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return nil, fmt.Errorf("Python вернул путь, но файл не существует: %s", p)
		}
	}

	return paths, nil
}

// recognizeSingleImage распознает одно изображение с fallback-моделью.
func recognizeSingleImage(ctx context.Context, cfg *config.Config, imagePath string) (string, error) {
	imageData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", fmt.Errorf("не удалось прочитать файл %s: %w", imagePath, err)
    }
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Пробуем handwritten
	log.Printf("Пробую модель: handwritten")
	result, err := tryRecognize(ctx, cfg, encoded, handwrittenModel)
	if err != nil {
		log.Printf("handwritten error: %v", err)
	}

	// Если handwritten вернул слишком мало текста - пробуем page
	if err != nil || len(strings.TrimSpace(result.filteredText)) < 10 {
		log.Printf("handwritten дал мало текста (%d символов), пробую page", len(result.filteredText))
		pageResult, pageErr := tryRecognize(ctx, cfg, encoded, pageModel)
		if pageErr == nil && len(strings.TrimSpace(pageResult.filteredText)) > len(strings.TrimSpace(result.filteredText)) {
			result = pageResult
			err = nil
		}
	}

	if err != nil {
		return "", err
	}

	// КЛЮЧЕВОЙ МОМЕНТ: если фильтрация всё отбросила - используем rawText
	if len(strings.TrimSpace(result.filteredText)) == 0 && len(result.rawText) > 0 {
		log.Printf("Фильтрация отбросила всё! Возвращаю rawText (%d символов)", len(result.rawText))
		return result.rawText, nil
	}

	return result.filteredText, nil
}

// RecognizeResult содержит и сырой, и отфильтрованный текст
type RecognizeResult struct {
	rawText      string
	filteredText string
}

// tryRecognize отправляет запрос к Yandex Vision и парсит ответ.
func tryRecognize(ctx context.Context, cfg *config.Config, encoded, model string) (RecognizeResult, error) {
	// Определяем mimeType по содержимому
	mimeType := "image/jpeg"
	if len(encoded) > 10 && strings.HasPrefix(encoded, "iVBORw0KGgo") {
		mimeType = "image/png"
	}

	reqBody := map[string]interface{}{
		"mimeType":      mimeType,
		"languageCodes": []string{"ru", "en"},
		"model":         model,
		"content":       encoded,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return RecognizeResult{}, fmt.Errorf("ошибка маршалинга: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://ocr.api.cloud.yandex.net/ocr/v1/recognizeText",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return RecognizeResult{}, err
	}
	req.Header.Set("Authorization", "Api-Key "+cfg.YandexVisionAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-folder-id", cfg.YandexFolderID)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return RecognizeResult{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return RecognizeResult{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return RecognizeResult{}, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if os.Getenv("DEBUG_OCR") == "true" {
		log.Printf("Raw API response for model %s:\n%s", model, string(respBody))
	}

	var result struct {
		Result struct {
			TextAnnotation struct {
				FullText string `json:"fullText"`
				Blocks   []struct {
					Lines []struct {
						Text  string `json:"text"`
						Words []struct {
							Text string `json:"text"`
						} `json:"words"`
					} `json:"lines"`
				} `json:"blocks"`
			} `json:"textAnnotation"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return RecognizeResult{}, fmt.Errorf("ошибка парсинга: %w", err)
	}

	annotation := result.Result.TextAnnotation
	filtered := filterAndCleanText(annotation)

	// Логируем статистику
	totalLines := 0
	for _, b := range annotation.Blocks {
		totalLines += len(b.Lines)
	}
	filteredLines := len(strings.Split(filtered, "\n"))
	if filtered != "" {
		log.Printf("[%s] Строк всего: %d, после фильтра: %d, avgConfidence: см. ниже", model, totalLines, filteredLines)
	}

	return RecognizeResult{
		rawText:      annotation.FullText,
		filteredText: filtered,
	}, nil
}

// filterAndCleanText удаляет шум на основе эвристик.
func filterAndCleanText(annotation struct {
    FullText string `json:"fullText"`
    Blocks   []struct {
        Lines []struct {
            Text  string `json:"text"`
            Words []struct {
                Text string `json:"text"`
            } `json:"words"`
        } `json:"lines"`
    } `json:"blocks"`
}) string {
    var clean []string
    for _, block := range annotation.Blocks {
        for _, line := range block.Lines {
            text := strings.TrimSpace(line.Text)
            // 1. Длина строки не менее 3 символов
            if len([]rune(text)) < 3 {
                continue
            }
            // 2. Не должно быть только одиночных символов
            if len(text) <= 4 && strings.Contains(text, ".") {
                continue
            }
            clean = append(clean, text)
        }
    }
    return strings.Join(clean, "\n")
}

// mergeAndDeduplicate объединяет тексты и убирает дубликаты.
func mergeAndDeduplicate(texts []string) string {
    var allLines []string
    seen := make(map[string]bool)
    for _, text := range texts {
        lines := strings.Split(text, "\n")
        for _, line := range lines {
            line = strings.TrimSpace(line)
            if line == "" || seen[line] {
                continue
            }
            seen[line] = true
            allLines = append(allLines, line)
        }
    }
    return strings.Join(allLines, "\n")
}