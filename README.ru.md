# Клиент Z.AI API

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/view.html?uri=github.com/SamyRai/go-z-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

CLI и клиентская библиотека на Go для платформы Z.AI (Zhipu AI / BigModel):
завершение чата, модели, изображения, видео, аудио, эмбеддинги, модерация,
ререйтинг, агенты, пакетные задачи, разбор файлов, управление учётными
записями и квотами GLM Coding Plan, а также порт `@z_ai/coding-helper` на Go —
для подключения Claude Code, OpenCode, Crush, Factory Droid и Cursor к вашему
GLM Coding Plan.

## Установка

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Требуется Go 1.26.4+ и [API-ключ Z.AI](https://z.ai/manage-apikey). Сборка из
исходников, первичная аутентификация и устранение неполадок:
**[Начало работы →](docs/ru/getting-started.md)**

## Быстрый пример

```bash
export ZAI_API_KEY=your_api_key_here
zai-client chat create "Объясни горутины одним абзацем" --stream
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Объясни горутины одним абзацем"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Больше готовых примеров — потоковая передача, асинхронный опрос изображений,
эндпоинт Anthropic `/v1/messages` — в каталоге [`examples/`](examples/).

## Документация

**[Полный указатель документации →](docs/ru/README.md)**

| | |
|---|---|
| [Начало работы](docs/ru/getting-started.md) | [Справочник по CLI](docs/ru/cli-reference.md) |
| [Аккаунты и квоты](docs/ru/accounts-and-quota.md) | [Инструменты для кода](docs/ru/coding-tools.md) |
| [Руководство по библиотеке](docs/ru/library-guide.md) | [Обработка ошибок](docs/ru/error-handling.md) |
| [Архитектура](docs/ru/architecture.md) | [Дорожная карта и ограничения](docs/ru/roadmap.md) |
| [Участие в проекте](CONTRIBUTING.md) | [Политика безопасности](SECURITY.md) |
| [Кодекс поведения](CODE_OF_CONDUCT.md) | [Журнал изменений](CHANGELOG.md) |

## Что покрывается

Чат (потоковая передача, структурированный вывод, глубокое мышление, вызов
функций, визуальный ввод), совместимый с Anthropic эндпоинт `/v1/messages`,
модели, изображения, видео, аудио (транскрипция + TTS + клонирование голоса),
OCR и разбор документов, эмбеддинги, модерация, ререйтинг, агенты, файлы,
пакетные задачи, управление использованием/квотой/несколькими аккаунтами GLM
Coding Plan, а также полноэкранный терминальный интерфейс
(`zai-client tui`). Полный список команд — в
[справочнике по CLI](docs/ru/cli-reference.md), Go API — в
[руководстве по библиотеке](docs/ru/library-guide.md).

## Связь с официальными SDK

Z.AI / Zhipu выпускают официальные SDK для **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
и **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4)).
Официального SDK для Go **нет** — `go-z-ai` заполняет этот пробел и поверх того
же API добавляет CLI, TUI, переключение региональных шлюзов
(`api.z.ai` ↔ `open.bigmodel.cn`) и управление несколькими аккаунтами GLM
Coding Plan.

> ℹ️ `zai-claude-config.json` в корне репозитория — это **шаблон** с
> плейсхолдерами (`"your-zai-api-key-here"`), который используется командой
> `zai-client coding load claude-code`. Это не настоящий конфиг, и в нём нет
> реальных учётных данных.

## Участие в проекте

См. [CONTRIBUTING.md](CONTRIBUTING.md) — в частности, соглашение проекта о
живой верификации (запись реальных API-вызовов в кассеты вместо
рукописных фикстур), если вы добавляете или изменяете сервис.

## Лицензия

Apache License 2.0 — см. [LICENSE](LICENSE).

## Поддержка

- **Документация Z.AI API**: [https://docs.z.ai](https://docs.z.ai)
- **Issues**: [GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **Безопасность**: см. [SECURITY.md](SECURITY.md) — пожалуйста, не
  сообщайте об уязвимостях через публичные issues.
