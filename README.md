# URL Shortener

Сервис сокращения ссылок на Go. Pet-проект с многослойной архитектурой, HTTP API, персистентным хранилищем и CI.

> **Статус:** проект в активной разработке. Базовый функционал работает; сейчас добавляются новые возможности.

## Требования

- Go 1.26+

## Быстрый старт

### Сервер

```bash
go run ./cmd/server
```

Сервер поднимается на `localhost:8080`, короткие ссылки формируются с базой `http://localhost:8080`, данные сохраняются в `tmp/short-url-db.json`.

### Клиент

В отдельном терминале (сервер должен быть запущен):

```bash
go run ./cmd/client
```

Клиент запрашивает длинный URL через stdin и отправляет его на сервер. Ответ — короткая ссылка.

## Конфигурация сервера

### Флаги

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `-a` | `localhost:8080` | Адрес и порт HTTP-сервера |
| `-b` | `http://localhost:8080` | Базовый URL для формирования коротких ссылок |
| `-f` | `tmp/short-url-db.json` | Путь к файлу персистентного хранилища |

Пример:

```bash
go run ./cmd/server -a localhost:9090 -b http://localhost:9090 -f data/urls.json
```

### Переменные окружения

Переменные окружения переопределяют значения флагов:

| Переменная | Описание |
|------------|----------|
| `SERVER_ADDRESS` | Адрес сервера (аналог `-a`) |
| `BASE_URL` | Базовый URL (аналог `-b`) |
| `FILE_STORAGE_PATH` | Путь к файлу хранилища (аналог `-f`) |

Пример:

```bash
SERVER_ADDRESS=localhost:9090 BASE_URL=http://localhost:9090 go run ./cmd/server
```

Примечание:

При установке `FILE_STORAGE_PATH=""`, данные сохраняются в оперативной памяти.

## HTTP API

| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/` | Сократить URL. Тело — `text/plain`, ответ — короткая ссылка (`201 Created`) |
| `POST` | `/api/shorten` | Сократить URL. Тело — `{"url":"..."}`, ответ — `{"result":"..."}` (`201 Created`) |
| `GET` | `/{id}` | Редирект на исходный URL (`307 Temporary Redirect`) |

Примеры с `curl` (bash):

```bash
# text/plain
curl -X POST http://localhost:8080/ -d "https://example.com"

# JSON
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'

# JSON с gzip-сжатием ответа
curl -X POST http://localhost:8080/api/shorten \
  -H "Accept-Encoding: gzip" \
  -H "Content-Type: application/json" \
  -d '{"url":"http://example.com"}' \
  --compressed -i

# редирект
curl -L http://localhost:8080/{id}
```

## Структура проекта

```
cmd/                    — точки входа приложения
  server/               — HTTP-сервер
  client/               — CLI-клиент
internal/               — внутренние модули приложения
  config/               — конфигурация приложения
    db/                 — конфигурация подключения к БД (TBD)
  handler/              — HTTP-обработчики
  service/              — бизнес-логика
  repository/           — работа с хранилищем данных и внешними сервисами
  middleware/           — HTTP-middleware
  logger/               — структурированное логирование
  model/                — доменные структуры
  myjson/               — JSON DTO и сериализация
  parser/               — парсеры
  urlutils/             — утилиты для работы с URL
api/                    — контракт API, OpenAPI/Swagger (TBD)
migrations/             — миграции БД (TBD)
pkg/                    — переиспользуемые пакеты (TBD)
```

## Что реализовано

### Архитектура

- Разделение на слои: **handler → service → repository** с интерфейсами между ними.
- Роутинг через **chi**; dependency injection в `main`.
- Конфигурация через **флаги командной строки** и **переменные окружения** (env имеет приоритет).

### Сокращение и редирект

- Генерация короткого ID (8 символов, `crypto/rand`) с редиректом на исходный URL.
- Два формата API: plain text (`POST /`) и JSON (`POST /api/shorten`).
- Нормализация URL: валидация, автодобавление `http://` при отсутствии схемы.

### Хранение данных

- **In-memory** репозиторий как fallback.
- **File-backed** репозиторий: JSON-lines файл, данные переживают перезапуск сервера.
- Плагинная фабрика `repository.New` — выбор бэкенда по конфигурации.

### Наблюдаемость и производительность

- Структурированное логирование HTTP-запросов (**zap**): URI, метод, статус, длительность, размер ответа.
- Middleware **gzip**: распаковка `Content-Encoding: gzip` в запросах, сжатие ответов по `Accept-Encoding`.
- Парсер **Accept-Encoding** с учётом q-значений и wildcard.
- JSON-сериализация через **easyjson**.

### Тестирование и CI

- Unit-тесты для handler, repository, middleware, parser (table-driven, testify/mock).
- GitHub Actions: автотесты по инкрементам, statictest, PostgreSQL в CI для будущих итераций.

### CLI-клиент

- Консольный клиент на **resty** для демонстрации работы с API.

## Сборка

```bash
go build -o shortener ./cmd/server
go build -o client ./cmd/client
```

## Тесты

```bash
go test ./...
```
