# Платформа для чата в реальном времени

Простое приложение для чата в реальном времени, построенное на Go, с WebSocket, аутентификацией пользователей и сохранением историй сообщений.

## Возможности

- **Аутентификация пользователей** - Регистрация и вход с JWT токенами
- **Обмен сообщениями в реальном времени** - Мгновенные сообщения на основе WebSocket
- **История сообщений** - Постоянное хранение сообщений в SQLite
- **Пользователи онлайн** - Живой список подключенных пользователей
- **Индикаторы печати** - Статус печати в реальном времени
- **Адаптивный интерфейс** - Чистый интерфейс с использованием Bootstrap

## Быстрый старт

### Требования
- Go 1.19+
- Git

### Установка

1. Клонируйте и перейдите в проект:
   ```bash
   git clone https://github.com/tonboek/realtime_chat_platform.git
   cd realtime_chat_platform
   ```

2. Установите зависимости:
   ```bash
   go mod tidy
   ```

3. Запустите приложение:
   ```bash
   go run cmd/main.go
   ```

4. Откройте `http://localhost:8080` в браузере

## Структура проекта

```
realtime_chat_platform/
├── cmd/main.go              # Точка входа приложения
├── internal/
│   ├── database/database.go # Настройка базы данных
│   ├── handlers/            # HTTP обработчики
│   ├── models/user.go       # Модели данных
│   └── websocket/           # WebSocket логика
├── web/
│   ├── static/              # CSS и JavaScript
│   └── templates/           # HTML шаблоны
└── README.md
```

## API Endpoints

- `POST /api/register` - Регистрация пользователя
- `POST /api/login` - Вход пользователя
- `GET /api/messages` - История сообщений
- `GET /api/users/online` - Пользователи онлайн
- `GET /api/ws` - WebSocket соединение

## Технологический стек

- **Backend**: Go, Gin framework, Gorilla WebSocket
- **База данных**: SQLite с GORM
- **Frontend**: HTML, CSS, JavaScript, Bootstrap
- **Аутентификация**: JWT токены с хешированием паролей bcrypt