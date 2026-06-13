# RedPen — AI-ассистент для проверки тетрадей (Красная Ручка)

[![CI](https://github.com/4vertak/redpen-checker/actions/workflows/ci.yml/badge.svg)](https://github.com/4vertak/redpen-checker/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**RedPen** — это эксперимент (гипотеза), который проверяет, может ли связка облачных AI‑сервисов помочь учителю начальных классов проверять тетради быстрее и объективнее.

Прямо сейчас продукта ещё нет. Есть работающий фундамент, открытый код и желание проверить, насколько точно GigaChat распознает детский почерк, а DeepSeek — находит ошибки и даёт понятные пояснения.

## Для кого

- **Учителя начальных классов**, которые хотят тратить меньше времени на рутину и больше на общение с семьёй.
- **Go‑разработчики**, которые ищут живой open‑source проект с понятной архитектурой, CI/CD и реальной социальной ценностью.

Подробнее о пользователях — [документ персон](./docs/target-audience-and-personas.md).

## Что должно получиться в MVP

1. Учитель входит в систему (email + пароль).
2. Загружает фото тетради, указывает предмет и задание.
3. Система распознаёт рукописный текст (GigaChat API) и анализирует ошибки (DeepSeek API).
4. Учитель видит результат, может поправить распознанный текст, изменить ошибки и выставить оценку.
5. Результат сохраняется в истории проверок.

Пакетная проверка, импорт классов из CSV, дашборды — в планах на будущие версии.

## Архитектура

Веб-приложение на **Go + Gin** работает как оркестратор: получает фото → отправляет в GigaChat API → распознанный текст → отправляет в DeepSeek API → возвращает учителю результат с ошибками и пояснениями. Всё взаимодействие в MVP — синхронное.

## Технологический стек

- **Язык:** Go 1.26+
- **Веб-фреймворк:** Gin
- **База данных:** PostgreSQL (драйвер pgx/v5)
- **Миграции:** golang-migrate
- **AI-сервисы:** GigaChat API (OCR), DeepSeek API (анализ текста)
- **Контейнеризация:** Docker, Docker Compose
- **CI/CD:** GitHub Actions (линтер + тесты)

## Быстрый старт (локально)

### Предварительные требования
- Go 1.26+
- Docker и Docker Compose
- Ключи API:
  - [GigaChat API](https://developers.sber.ru/)
  - [DeepSeek API](https://platform.deepseek.com/)

### 1. Клонирование
```bash
git clone https://github.com/4vertak/redpen-checker.git
cd redpen-checker
```

### 2. Переменные окружения
Создайте файл `.env` в корне проекта:
```
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/redpen?sslmode=disable
GIGACHAT_API_KEY=ваш_ключ
DEEPSEEK_API_KEY=ваш_ключ
```

### 3. Запуск через Docker Compose
```bash
docker compose up -d
```
Проверьте: [http://localhost:8080/health](http://localhost:8080/health)

### 4. Ручной запуск (для разработки)
```bash
go run ./cmd/server/main.go
```

## Документация

- [Целевая аудитория и персоны](./docs/target-audience-and-personas.md)
- [Функциональные и нефункциональные требования](./docs/product-requirements.md)
- [Архитектура](./docs/architecture.md)
- Аналитика:
  - [Глоссарий](./docs/analysis/glossary.md)
  - [ER-диаграмма](./docs/analysis/er-diagram.puml)
  - [Сценарий проверки одной работы](./docs/analysis/use-cases/single-check.md)
  - [Диаграмма последовательности](./docs/analysis/sequences/single-check-sequence.puml)

## Как помочь

Мы на старте и будем рады любой помощи! Посмотрите [CONTRIBUTING.md](./CONTRIBUTING.md), чтобы узнать о правилах оформления кода и процессе создания PR. Актуальные задачи и бэклог ведутся на [GitHub Projects](https://github.com/4vertak/redpen-checker/projects).

Для новичков есть задачи с меткой [`good first issue`](https://github.com/4vertak/redpen-checker/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)

## Лицензия

MIT © 2026 RedPen Project

---
