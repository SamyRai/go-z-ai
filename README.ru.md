# go-z-ai

**CLI**, **библиотека** и **TUI** на Go для платформы Z.AI (Zhipu AI /
BigModel) — все возможности моделей GLM в одном инструменте, плюс порт
`@z_ai/coding-helper` на Go, который подключает Claude Code, OpenCode, Crush,
Factory Droid и Cursor к вашему GLM Coding Plan.

**English** | [简体中文](README.zh.md) | **Русский** | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/viewer/?uri=github.com/SamyRai/go-z-ai)
[![Latest release](https://img.shields.io/github/v/release/SamyRai/go-z-ai)](https://github.com/SamyRai/go-z-ai/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Быстрый пример

```bash
# 1. Настройка (подойдёт любой вариант — env-переменная, файл .env или --config <файл>)
export ZAI_API_KEY=your_api_key_here
# или: cp .env.example .env, затем отредактируйте .env

# 2. Использование CLI
go-z-ai chat create "Объясни горутины одним абзацем" --stream
```

```go
// …или импортируйте библиотеку — CLI не требуется.
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Объясни горутины одним абзацем"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Больше готовых программ — потоковая передача, асинхронный опрос изображений,
эндпоинт Anthropic `/v1/messages` — в каталоге [`examples/`](examples/).

## Возможности

- **Чат** — потоковая передача, структурированный вывод (JSON Schema), глубокое
  мышление, вызов функций/инструментов, визуальный ввод (`glm-4.6v`/`glm-4.5v`)
  и **совместимый с Anthropic эндпоинт `/v1/messages`** (тот самый, к которому
  обращаются Claude Code и Cursor при подключении к GLM Coding Plan).
- **Медиа** — генерация изображений, генерация видео (всегда асинхронно),
  транскрипция аудио, TTS и клонирование голоса GLM-TTS.
- **Анализ документов** — OCR макета, OCR рукописного текста и парсер документов
  для RAG-предобработки.
- **Поиск** — эмбеддинги, ререйтинг, встроенные инструменты веб-поиска /
  веб-ридера / токенизатора.
- **Модерации** — модерация контента через эндпоинт китайской платформы.
- **Агенты** — специализированные агенты Z.AI (перевод, генерация
  слайдов/постеров, видеоэффекты).
- **Пакетная обработка и файлы** — пакетные задачи JSONL для завершения чата,
  загрузка/список/скачивание файлов.
- **GLM Coding Plan** — мониторинг квоты/использования, управление несколькими
  аккаунтами и `go-z-ai coding` для подключения Claude Code, OpenCode, Crush,
  Factory Droid и Cursor к вашей подписке.
- **DX** — полноэкранный терминальный интерфейс (`go-z-ai tui`), переключение
  региональных шлюзов (`api.z.ai` ↔ `open.bigmodel.cn`), автоматический повтор с
  экспоненциальной задержкой и джиттером, а также типизированный `APIError` с
  маппингом всех кодов ошибок Z.AI.

## Установка

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Это создаст бинарник `go-z-ai` в `$GOPATH/bin`.

```bash
# Необязательный короткий псевдоним: ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Требуется Go 1.26.4+ и [API-ключ Z.AI](https://z.ai/manage-apikey/apikey-list). Сборка из
исходников, первичная аутентификация и устранение неполадок:
**[Начало работы →](docs/ru/getting-started.md)**

## Как CLI

Один бинарник `go-z-ai` покрывает весь функционал. Каждая команда поддерживает
`--help`; краткий обзор:

```bash
go-z-ai chat create "..." --stream          # чат (потоковая передача, инструменты, визуальный ввод, структурированный вывод)
go-z-ai anthropic messages "..." --stream   # совместимый с Anthropic /v1/messages
go-z-ai image|video|audio|voice ...         # генерация медиа, транскрипция, TTS, клонирование
go-z-ai ocr|parser ...                      # OCR + разбор документов
go-z-ai embeddings|rerank|moderations ...   # поиск + модерация контента
go-z-ai models list                         # каталог моделей + цены
go-z-ai accounts add|use|quota|usage ...    # несколько аккаунтов + мониторинг GLM Coding Plan
go-z-ai coding auth|load|doctor|mcp ...     # подключение Claude Code / Cursor / и т.д. к GLM Coding Plan
go-z-ai tui                                 # полноэкранный терминальный интерфейс (всё перечисленное выше)
go-z-ai validate                            # проверка ключа одним реальным вызовом
```

Каждая команда, возвращающая результат, принимает `--format text|json` (JSON
идёт в stdout, прогресс-сообщения в stderr, поэтому можно передавать в `jq`).

→ Полный список команд: **[Справочник по CLI](docs/ru/cli-reference.md)**

## Как Go-библиотека

`pkg/client` — единственный публично импортируемый пакет; всё, что находится в
`internal/`, — детали реализации. Повтор, таймаут, выбор регионального шлюза и
маппинг ошибок централизованы — сервисы никогда не создают собственный
`http.Client` и не выполняют сырые запросы.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Опционально: BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey, Region
})
```

Сервисы, все следуют шаблону `c.<Service>().<Method>(ctx, …)`:

| Доступ | Что покрывает |
|---|---|
| `c.Chat()` | Completion, потоковая передача, async, `RunWithTools` |
| `c.Anthropic()` | `/v1/messages` по протоколу Anthropic (Create, CreateStream) |
| `c.Models()` | List, Get, фильтры text/vision/free |
| `c.Images()` / `c.Videos()` | Изображения (sync/async), видео (всегда async) |
| `c.Audio()` / `c.Voice()` | Транскрипция, TTS, клонирование голоса |
| `c.Layout()` / `c.FileParser()` | OCR + документ-в-текст для RAG |
| `c.Files()` / `c.Batch()` | Загрузка, пакетные задачи |
| `c.Agents()` | Специализированные агенты Z.AI |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Поиск + модерация |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Usage()` / `c.Quota()` / `c.Account()` / `c.Detection()` | Мониторинг GLM Coding Plan |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Общий опрос для асинхронных задач |

→ Полное API с примерами: **[Руководство по библиотеке](docs/ru/library-guide.md)**
→ Сгенерированный референс: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Конфигурация

Три способа указать учётные данные, разрешаются в следующем порядке приоритета
(высший побеждает):

| Метод | Когда использовать |
|---|---|
| флаг `--api-key <key>` | Разовые вызовы, скрипты, CI |
| флаг `--account <name>` | Переключение между [сохранёнными аккаунтами](docs/ru/accounts-and-quota.md) |
| env-переменная `ZAI_API_KEY` (или файл `.env`) | Повседневное локальное использование в shell |
| Активный аккаунт из хранилища аккаунтов | После `go-z-ai accounts use <name>` |

Файл `.env` — наиболее частый вариант: скопируйте аннотированный шаблон и
отредактируйте его:

```bash
cp .env.example .env
# или укажите любой файл: go-z-ai --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # переопределить эндпоинт чата
# ZAI_REGION=china                                   # если ваш ключ выпущен на open.bigmodel.cn
# ZAI_CHINA_API_KEY=...                              # отдельные учётные данные bigmodel.cn
# ZAI_ENV=production
```

→ Полный референс (несколько аккаунтов, региональные шлюзы, окна квот):
**[Аккаунты и квоты](docs/ru/accounts-and-quota.md)**

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
> `go-z-ai coding load claude-code`. Это не настоящий конфиг, и в нём нет
> реальных учётных данных.

## Участие в проекте

См. [CONTRIBUTING.md](CONTRIBUTING.md) — в частности, соглашение проекта о
живой верификации (запись реальных API-вызовов в кассеты вместо рукописных
фикстур), если вы добавляете или изменяете сервис.

## Лицензия

Apache License 2.0 — см. [LICENSE](LICENSE).

## Поддержка

- **Документация Z.AI API**: [https://docs.z.ai](https://docs.z.ai)
- **Issues**: [GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **Безопасность**: см. [SECURITY.md](SECURITY.md) — пожалуйста, не
  сообщайте об уязвимостях через публичные issues.
