package main

import (
	"context"
	"fmt"
	"os"

	"github.com/4vertak/redpen-checker/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run ./cmd/testocr/main.go <путь_к_изображению>")
		os.Exit(1)
	}
	imagePath := os.Args[1]

	text, err := service.RecognizeText(context.Background(), imagePath)
	if err != nil {
		fmt.Printf("Ошибка распознавания: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Распознанный текст:")
	fmt.Println(text)
}
