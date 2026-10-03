# go-z-ai

**CLI**, **библиотека** и **TUI** на Go для платформы Z.AI (Zhipu AI /
BigModel) — все возможности моделей GLM в одном инструменте, плюс порт
`@z_ai/coding-helper` на Go, который подключает Claude Code, Codex, OpenCode, Crush
и Factory Droid к вашему GLM Coding Plan.

[English](README.md) | [简体中文](README.zh.md) | **Русский** | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

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
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv() // читает ZAI_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	req := client.ChatRequest{
		Model:    client.DefaultModel,
		Messages: []client.Message{{Role: "user", Content: "Explain goroutines in one paragraph"}},
	}
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			log.Fatal(err)
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
}
```

Больше готовых программ — вызов инструментов, структурированный вывод,
визуальный ввод, асинхронный опрос изображений, эндпоинт Anthropic
`/v1/messages` — в каталоге [`examples/`](examples/).

## Возможности

- **Чат** — потоковая передача (в библиотеке — итератор Go), управление
  рассуждением (`reasoning_effort`, включение/выключение thinking), вызов
  функций/инструментов (плюс встроенные типы инструментов `web_search`,
  `retrieval` и `mcp`), вывод в виде JSON-объекта со схемой в промпте,
  мультимодальный ввод (изображение, видео, файл) и **совместимый с Anthropic
  эндпоинт `/v1/messages`** (тот самый, к которому обращается Claude Code при
  подключении к GLM Coding Plan).
- **Модели** — курируемый каталог (контекстное окно, максимальный вывод, цены,
  возможности, допустимые уровни рассуждения) за командой `go-z-ai models` и
  `client.CatalogEntry`; значения по умолчанию перечислены
  [ниже](#модели-по-умолчанию).
- **Медиа** — генерация изображений, генерация видео (всегда асинхронно, с
  необязательной постановкой в очередь вне пиковых часов), транскрипция аудио,
  TTS и клонирование голоса GLM-TTS.
- **Анализ документов** — OCR макета, OCR рукописного текста и парсер документов
  для RAG-предобработки.
- **Поиск** — эмбеддинги, ререйтинг, встроенные инструменты веб-поиска /
  веб-ридера / токенизатора.
- **Модерации** — модерация контента через эндпоинт китайской платформы.
  Эмбеддинги и модерации обслуживаются `open.bigmodel.cn` и могут быть
  ограничены правами аккаунта; см.
  [Дорожная карта и ограничения](docs/ru/roadmap.md).
- **Агенты** — специализированные агенты Z.AI (перевод, генерация
  слайдов/постеров, видеоэффекты).
- **Пакетная обработка и файлы** — пакетные задачи JSONL для завершения чата,
  загрузка/список/скачивание/удаление файлов.
- **GLM Coding Plan** — определение типа аккаунта, мониторинг квоты/использования
  на основе кредитов, управление несколькими аккаунтами и `go-z-ai coding` для
  подключения Claude Code, Codex, OpenCode, Crush и Factory Droid к вашей
  подписке и регистрации четырёх официальных MCP-серверов Z.AI (Vision,
  веб-поиск, веб-ридер, Zread).
- **DX** — полноэкранный терминальный интерфейс (`go-z-ai tui`: вкладки чата,
  моделей, использования, аккаунтов, кодинга, медиа и инструментов), один
  переключатель `--region` (`api.z.ai` ↔ `open.bigmodel.cn`), выбирающий все
  эндпоинты, автоматический повтор с экспоненциальной задержкой и джиттером (с
  учётом `Retry-After`), а также типизированный `APIError` с маппингом кодов
  ошибок Z.AI (неизвестные коды сводятся к HTTP-статусу).

### Модели по умолчанию

Вызывающий код берёт значения по умолчанию из констант `pkg/client`, а не
зашивает ID моделей:

| Константа | Модель | Для чего |
|---|---|---|
| `client.DefaultModel` | `glm-5.3` | Чат, эндпоинт Anthropic, токенизатор. Контекст 1M, уровни рассуждения `low`/`high`/`max` |
| `client.DefaultFastModel`, `client.DefaultVisionModel` | `glm-5.3-flash` | Быстрый недорогой уровень; нативно мультимодальная (ввод изображений, видео, файлов) |
| `client.DefaultOCRModel` | `glm-ocr` | `ocr`, `Layout().Parse` |
| `client.DefaultASRModel` | `glm-asr-2512` | `audio transcribe` (клипы до 30 с) |
| `client.DefaultTTSModel` | `glm-tts` | `audio speech` |
| `client.ModelGLMImage`, `client.ModelCogView4` | `glm-image` (по умолчанию), `cogview-4-250304` | `image generate` |
| `client.VideoModels` | `cogvideox-3` (по умолчанию), `viduq1-*`, `vidu2-*` | `video generate` |

Каталог также охватывает `glm-5.3-flashx`, `glm-5.2`, семейство GLM-4.x и
бесплатные уровни (`go-z-ai models free`). Это курируемый снимок (последнее
обновление 2026-10-02) — `go-z-ai models list` показывает контекст,
максимальный вывод, цены, возможности и уровни рассуждения, а актуальные
значения `/models` имеют приоритет над значениями каталога.

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
go-z-ai chat create "..." --stream          # чат (потоковая передача, уровень рассуждения, инструменты, ввод изображений/видео/файлов, вывод JSON)
go-z-ai anthropic messages "..." --stream   # совместимый с Anthropic /v1/messages
go-z-ai responses create "..." --stream     # протокол OpenAI Responses (/api/v1, его использует Codex)
go-z-ai image|video|audio|voice ...         # генерация медиа, транскрипция, TTS, клонирование
go-z-ai ocr|parser ...                      # OCR + разбор документов
go-z-ai embeddings|rerank|moderations ...   # поиск + модерация контента
go-z-ai models list                         # каталог моделей: контекст, цены, возможности, уровни рассуждения
go-z-ai account detect|status               # тип ключа (coding plan / оплата по использованию), регион, состояние
go-z-ai accounts add|use|quota|usage ...    # несколько аккаунтов + мониторинг GLM Coding Plan
go-z-ai usage quota                         # окна квоты GLM Coding Plan для текущего ключа
go-z-ai coding auth|load|doctor|mcp ...     # подключение Claude Code / Codex / OpenCode / Crush / Factory Droid к GLM Coding Plan
go-z-ai tui                                 # полноэкранный терминальный интерфейс (всё перечисленное выше)
go-z-ai validate                            # проверка работоспособности ключа (бесплатный запрос)
```

Большинство команд, выводящих результат, принимают `--format text|json` (JSON
идёт в stdout, прогресс-сообщения в stderr, поэтому можно передавать в `jq`).
Корневой флаг `--region global|china` (env `ZAI_REGION`) выбирает все
эндпоинты; `--base-url` переопределяет только корень чата/PaaS.

### Инструменты для кода

`go-z-ai coding` — порт Z.AI `@z_ai/coding-helper` на Go. Поддерживаемые
инструменты: `claude-code`, `codex`, `opencode`, `crush` и `factory-droid`.

```bash
go-z-ai coding auth glm_coding_plan_global <key>   # проверить и сохранить ключ плана (или glm_coding_plan_china)
go-z-ai coding load claude-code                    # записать конфигурацию инструмента (также: codex, opencode, crush, factory-droid)
go-z-ai coding mcp add claude-code                 # зарегистрировать официальные MCP-серверы (--server для выбора)
go-z-ai coding doctor                              # проверка состояния; при проблемах завершается с ненулевым кодом
```

- Для Claude Code уровень haiku сопоставляется с `client.DefaultFastModel`, а
  уровни sonnet/opus — с `client.DefaultModel`, с суффиксом Claude Code `[1m]`
  (контекст 1M). Флаги `--haiku`, `--sonnet`, `--opus`, `--no-model-mapping`,
  `--auto-compact-window`, `--max-thinking-tokens` и `--max-output-tokens`
  настраивают это в `coding load` и `coding auth`.
- `coding mcp add|remove <tool>` регистрирует четыре официальных MCP-сервера
  Z.AI: `zai-mcp-server` (Vision; запускается локально через `npx`, требуется
  Node.js), `web-search-prime`, `web-reader` и `zread` (размещённые,
  аутентификация ключом плана). По умолчанию регистрируются все серверы,
  которые поддерживает инструмент — Codex принимает только Vision
  (openai/codex#14793); `coding mcp status` показывает, что есть у каждого
  инструмента. Вызовы MCP расходуют квоту плана.
- Для Codex `coding load codex` записывает провайдер ZAI, работающий по
  протоколу Responses (`/api/v1`), в `~/.codex/config.toml`, а метаданные
  модели — в `~/.codex/models.json`.

→ Полный список команд: **[Справочник по CLI](docs/ru/cli-reference.md)**

## Как Go-библиотека

`pkg/client` — это библиотека (только стандартная библиотека Go); `pkg/observe` —
необязательный адаптер хуков OpenTelemetry. Всё, что находится в `internal/`, —
детали реализации. Повтор, таймаут, выбор регионального шлюза и маппинг ошибок
централизованы — сервисы никогда не создают собственный `http.Client` и не
выполняют сырые запросы.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

// Из окружения: ZAI_API_KEY, ZAI_API_BASE_URL, ZAI_REGION,
// ZAI_CHINA_API_KEY, ZAI_MONITOR_TIMEZONE.
c, err := client.NewClientFromEnv()

// Или явно:
c, err = client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Опционально: Region, BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey,
    // UserAgent, Hooks
})
```

Сервисы, все следуют шаблону `c.<Service>().<Method>(ctx, …)`:

| Доступ | Что покрывает |
|---|---|
| `c.Chat()` | `Create`, `Stream` (итератор), `CreateAsync`, `RunWithTools` |
| `c.Anthropic()` | `/v1/messages` по протоколу Anthropic (`Create`, `Stream`) |
| `c.Responses()` | Протокол OpenAI Responses по `/api/v1` — поверхность Codex (`Create`, `Stream`) |
| `c.Models()` | List, Get, фильтры text/vision/free |
| `c.Images()` / `c.Videos()` | Изображения (`Generate`, `GenerateAsync`), видео (всегда async) |
| `c.Audio()` / `c.Voice()` | Транскрипция, TTS, клонирование голоса |
| `c.Layout()` / `c.FileParser()` | OCR + документ-в-текст для RAG |
| `c.Files()` / `c.Batch()` | Загрузка, пакетные задачи |
| `c.Agents()` | Специализированные агенты Z.AI |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Поиск + модерация |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Quota()` / `c.Account()` / `c.Detection()` | Квота и использование GLM Coding Plan, информация об аккаунте, определение типа аккаунта |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Общий опрос для асинхронных задач |

Что стоит знать:

- Значения по умолчанию берутся из каталога: `client.DefaultModel`,
  `client.DefaultFastModel`, `client.DefaultVisionModel` и константы
  OCR/ASR/TTS. `ChatRequest.ReasoningEffort` проверяется по допустимым для
  модели уровням (`client.EffortLow`, `EffortHigh`, `EffortMax` для GLM-5.3);
  `client.CatalogEntry(model)` и `Pricing.Cost(usage)` дают размеры контекста и
  цены.
- `Chat().Stream`, `Anthropic().Stream` и `Responses().Stream` возвращают
  итераторы `iter.Seq2`. Выход из цикла закрывает поток, а встроенный в поток
  чанк с ошибкой завершает его с `*APIError`.
- У Z.AI нет формата ответа `json_schema`. Для структурированного вывода задайте
  `ResponseFormat: client.JSONObjectFormat()` и поместите
  `client.JSONSchemaPrompt(schema)` в системный промпт.
- `Config.Region` (`client.RegionGlobal` или `client.RegionChina`) определяет
  все URL; `Config.BaseURL` по умолчанию равен `Region.PaaSBaseURL()`.

→ Полное API с примерами: **[Руководство по библиотеке](docs/ru/library-guide.md)**
→ Сгенерированный референс: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Конфигурация

Учётные данные разрешаются в следующем порядке приоритета
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
# ZAI_REGION=china                                   # если ваш ключ выпущен на open.bigmodel.cn; выбирает все эндпоинты
# ZAI_CHINA_API_KEY=...                              # отдельные учётные данные bigmodel.cn (эмбеддинги/модерации)
# ZAI_MONITOR_TIMEZONE=UTC                           # часовой пояс API квот/использования (по умолчанию UTC+8)
```

Функция библиотеки `client.NewClientFromEnv()` читает те же переменные из
окружения процесса (файл `.env` она не загружает).

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
> плейсхолдерами (`"your-zai-api-key-here"`), который иллюстрирует настройки,
> записываемые командой `go-z-ai coding load claude-code`. Программа его не
> читает, это не настоящий конфиг, и в нём нет реальных учётных данных;
> актуальное сопоставление моделей можно получить командой
> `go-z-ai coding load claude-code`.
>
> ⚠️ **Политика использования.** Coding-эндпоинт Z.AI ограничен
> «официально поддерживаемыми инструментами» и запрещает доступ через SDK;
> см. [Coding Tools — Compliance](docs/en/coding-tools.md#compliance--usage-policy-).
> `go-z-ai` отправляет идентифицирующий заголовок `User-Agent` в каждом
> запросе, а его подкоманда `coding` подключает официально поддерживаемые
> инструменты. Прямое обращение к `pkg/client` против coding-эндпоинта из
> сторонней интеграции выполняется на ваш риск до официального включения в
> список.

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
