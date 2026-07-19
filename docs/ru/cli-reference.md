# Справочник по CLI

Каждая команда поддерживает `--help` для получения актуального и достоверного
списка флагов (`go-z-ai <command> --help`,
`go-z-ai <command> <subcommand> --help`). Эта страница — структурированный
обзор; считайте `--help` источником истины, если когда-либо возникнут
расхождения.

## Содержание

- [Глобальные флаги](#глобальные-флаги)
- [Чат](#чат)
- [Модели](#модели)
- [Аккаунты, использование и квота](#аккаунты-использование-и-квота)
- [Инструменты для кода (GLM Coding Plan)](#инструменты-для-кода-glm-coding-plan)
- [Файлы и пакетная обработка](#файлы-и-пакетная-обработка)
- [Генерация медиа](#генерация-медиа)
- [Разбор документов и OCR](#разбор-документов-и-ocr)
- [Вспомогательные инструменты поиска](#вспомогательные-инструменты-поиска)
- [Модерация контента](#модерация-контента)
- [Агенты](#агенты)
- [Инструменты (веб-поиск, reader, токенизатор)](#инструменты-веб-поиск-reader-токенизатор)
- [Эндпоинт, совместимый с Anthropic](#эндпоинт-совместимый-с-anthropic)
- [Терминальный UI](#терминальный-ui)

## Глобальные флаги

Эти флаги применяются ко всем командам:

| Flag | Описание |
|---|---|
| `--api-key string` | API-ключ Z.AI (или переменная окружения `ZAI_API_KEY`) |
| `--base-url string` | Базовый URL API (по умолчанию: `https://api.z.ai/api/paas/v4`) |
| `--account string` | Использовать сохранённый аккаунт по имени для этой команды (см. [Аккаунты и квоты](accounts-and-quota.md)) |
| `--china-api-key string` | Ключ open.bigmodel.cn для embeddings/модераций (или `ZAI_CHINA_API_KEY`; при отсутствии берётся `--api-key`) |
| `--region string` | Региональный шлюз для monitor/biz/agents/detection: `global` (api.z.ai, по умолчанию) или `china` (open.bigmodel.cn). Псевдонимы: `cn`, `bigmodel`, `west`. Либо переменная окружения `ZAI_REGION`. Не переопределяет `--base-url`. Неизвестные значения приводят к global. |
| `--config string` | Файл конфигурации (по умолчанию: `.env`) |
| `--version` | Печатает версию (тег, коммит, дату сборки) и выходит. Заполняется ldflags GoReleaser в релизных сборках; иначе `dev`. |

Каждая команда, возвращающая результат, принимает `--format text\|json`
(некоторые по умолчанию используют `json`, если полезная нагрузка ориентирована
на машину — например, `embeddings`, `moderations`). В режиме `json` сообщения о
прогрессе/статусе направляются в stderr, поэтому stdout остаётся корректным
JSON, который можно передать в `jq`.

Установите `--region china` (или `ZAI_REGION=china`), если ваш ключ был выдан
на `open.bigmodel.cn`, чтобы квота/использование, account-info, agents и
определение типа аккаунта попадали на соответствующий хост. Это **не** меняет
базовый URL чата (используйте для этого `--base-url`) и не меняет хост
embedings/модераций (всегда китайская платформа). См.
[Аккаунты и квоты § Региональные шлюзы](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn).

## Чат

```bash
go-z-ai chat create [message] [flags]
go-z-ai chat simple [model] [message]
go-z-ai chat async-result [task-id]
```

`chat create` — основная точка входа:

| Flag | Назначение |
|---|---|
| `--model string` | По умолчанию `glm-5.2` |
| `--stream` | Потоковая передача token за token |
| `--async` | Отправить без ожидания; опрашивать через `chat async-result <task-id>` |
| `--temperature float`, `--top-p float`, `--max-tokens int` | Управление сэмплингом |
| `--system string` | Системное сообщение |
| `--thinking string`, `--effort string` | Режим глубокого мышления и уровень усилий (`max\|xhigh\|high\|medium\|low\|minimal\|none`; `xhigh`→`max` только для GLM-5.2) |
| `--show-reasoning` | Печатать `reasoning_content` в stderr |
| `--json-schema string` | Структурированный вывод: `@file.json` или inline JSON |
| `--tool string` | Объявления инструментов для вызова функций: `@tools.json` или inline JSON-массив |
| `--image string` (повторяемый) | Прикрепить изображение: URL или `@path` к локальному файлу (base64). Требуется визуальная модель (`glm-4.6v`/`glm-4.5v`) |
| `--stop strings` | Стоп-последовательности (повторяемый, максимум 4) |
| `--format text\|json` | Формат вывода |

```bash
go-z-ai chat create "Summarize this in 3 bullets" --model glm-5.2 --stream
go-z-ai chat create "Describe this" --image @photo.jpg --model glm-4.6v
go-z-ai chat create "Extract fields" --json-schema @schema.json
```

Вызовы инструментов печатаются, а не выполняются CLI — см.
[Руководство по библиотеке § Вызов функций](library-guide.md#function-calling)
о Go-цикле `RunWithTools` с автоматическим выполнением.

> **Визуальная модель + вызов инструментов могут вернуть HTTP 401.** Из
> сообщений сообщества (например,
> [claude-code-router#1491](https://github.com/musistudio/claude-code-router/issues/1491))
> следует, что сочетание визуальной модели (`--image` на `glm-4.6v`/`glm-4.5v`)
> с инструментами вызова функций (`--tool`) в одном запросе на некоторых
> конфигурациях GLM отклоняется с 401 — аутентифицированный ключ всё равно
> падает только для этой комбинации. Если столкнулись с этим, разделите
> работу: используйте визуальную модель для шага с изображением и текстовую
> модель (`glm-5.2`) для шага с вызовом инструментов, вместо того чтобы
> отправлять изображения и инструменты вместе. Здесь это пока не воспроизведено
> на живом аккаунте — см. [Дорожная карта](roadmap.md).

## Модели

```bash
go-z-ai models list [--pricing]
go-z-ai models get [model-id]
go-z-ai models text | vision | free
```

## Аккаунты, использование и квота

Подробно описано в [Аккаунты и квоты](accounts-and-quota.md). Краткая справка:

```bash
go-z-ai accounts add <name> --api-key <key> [--type coding_plan|pay_as_you_go]
go-z-ai accounts list [--format json] [--reveal]   # ключи по умолчанию скрыты; --reveal для экспорта
go-z-ai accounts use <name>
go-z-ai accounts show [name] [--format json] [--reveal]
go-z-ai accounts current                            # сокращение для 'accounts show' (активный аккаунт)
go-z-ai accounts quota [--only name...]
go-z-ai accounts usage [--days N] [--today] [--metric model|tool|both]
go-z-ai accounts remove <name> [--yes]

go-z-ai usage quota | summary | account | billing | check [--watch] | detect
go-z-ai account info | status
go-z-ai validate
```

## Инструменты для кода (GLM Coding Plan)

Настраивает Claude Code, OpenCode, Crush, Factory Droid или Cursor на
использование вашего GLM Coding Plan. Полное руководство: [Инструменты для
кода](coding-tools.md).

```bash
go-z-ai coding auth <plan> <key>      # проверить и сохранить учётные данные
go-z-ai coding auth revoke
go-z-ai coding auth reload <tool>     # повторно записать сохранённые учётные данные в конфиг инструмента
go-z-ai coding load <tool>            # записать в конфиг инструмента
go-z-ai coding unload <tool>
go-z-ai coding status
go-z-ai coding tools                  # список поддерживаемых инструментов + статус установки
go-z-ai coding doctor                 # проверка работоспособности

go-z-ai coding mcp add <tool>         # зарегистрировать MCP-сервер Vision от Z.AI
go-z-ai coding mcp status
go-z-ai coding mcp remove <tool>
```

## Файлы и пакетная обработка

```bash
go-z-ai files upload <file> [--purpose batch|code-interpreter|agent|voice-clone-input]
go-z-ai files list [--purpose ...]
go-z-ai files delete <file-id>
go-z-ai files download <file-id> <output-path>

go-z-ai batch create <input-file-id> [--endpoint ...]
go-z-ai batch status <batch-id>
go-z-ai batch list [--after ...] [--limit N]
go-z-ai batch cancel <batch-id>
```

Пакетные задачи обрабатывают множество запросов завершения чата из JSONL-файла
асинхронно — сначала загрузите его, затем создайте пакетную задачу с
полученным file ID.

## Генерация медиа

```bash
# Изображения — модель по умолчанию glm-image (также поддерживается cogview-4-250304)
go-z-ai image generate <prompt> [--model ...] [--size ...] [--quality hd|standard] [--async]
go-z-ai image status <id>
# --quality: hd по умолчанию (~20 с); standard быстрее (~5-10 с).

# Видео — всегда асинхронно (cogvideox-3 | viduq1-text | viduq1-image | vidu2-image | ...)
go-z-ai video generate --prompt "..." [--model ...] [--duration N] [--aspect-ratio ...]
go-z-ai video status <id>

# Аудио
go-z-ai audio transcribe <file>                       # glm-asr, .wav/.mp3, <=25MB, <=30s
go-z-ai audio speech <text> <output-path> [--voice ...] [--speed N] [--format wav|pcm]

# Клонирование голоса (работает в паре с audio speech --voice)
go-z-ai voice clone <voice-name> <sample-file-id> <preview-text>
go-z-ai voice list [--name ...] [--type OFFICIAL|PRIVATE]
go-z-ai voice delete <voice-id>
```

## Разбор документов и OCR

```bash
# Layout OCR — изображение/PDF в Markdown
go-z-ai ocr parse <file-or-url> [--start-page N] [--end-page N]
go-z-ai ocr handwriting <file> [--probability]

# Разбор документов (препроцессинг для RAG/извлечения) — отдельный продукт, не OCR
go-z-ai parser parse <file> <file-type>              # синхронно
go-z-ai parser create <file> <tool-type> <file-type> # асинхронно: lite|expert|prime
go-z-ai parser result <task-id> <format>              # text|download_link
```

`parser` и `ocr` решают разные задачи: OCR извлекает layout/текст из
изображений; parser же предназначен для превращения документов в текст,
готовый для RAG, и поддерживает больше тарифов инструментов.

## Вспомогательные инструменты поиска

```bash
go-z-ai embeddings create <text> [--model embedding-3|embedding-2] [--dimensions N]
go-z-ai rerank <query> <documents...> [--top-n N]
```

Embeddings направляются на `open.bigmodel.cn` — см.
[Аккаунты и квоты § Региональные шлюзы](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn),
почему так и что это значит для аутентификации. Rerank использует `--base-url`
по умолчанию (не привязан к китайскому хосту).

## Модерация контента

```bash
go-z-ai moderations check <text>
```

Также направляется на `open.bigmodel.cn` — та же заметка, что и для embeddings
выше.

## Агенты

```bash
go-z-ai agents invoke <agent-id> <message> [--source-lang ...] [--target-lang ...]
go-z-ai agents async-result <agent-id> <async-id>
```

Вызывает специализированных агентов Z.AI (перевод, генерация слайдов/постеров,
шаблоны видеоэффектов). Примечание: Agents API возвращает HTTP 200, даже когда
вызов падает на бизнес-уровне (например, недостаточный баланс) — CLI
сообщает о такой неудаче из тела ответа, а не как об ошибке команды.

## Инструменты (веб-поиск, reader, токенизатор)

```bash
go-z-ai tools web-search <query> [--engine ...] [--count N]
go-z-ai tools web-reader <url> [--no-images]
go-z-ai tools tokenizer <text> [--model ...]
```

## Эндпоинт, совместимый с Anthropic

```bash
go-z-ai anthropic messages <prompt> [--model glm-4.6] [--max-tokens 1024] \
    [--system ...] [--temperature ...] [--thinking-budget N] [--stream]
```

Вызывает Anthropic-совместимый эндпоинт Z.AI (`/api/anthropic/v1/messages`) —
тот же эндпоинт, на который GLM Coding Plan направляет Claude Code, — вместо
OpenAI-стиля `chat create`. Печатает текст сообщения (или стримит текстовые
дельты с `--stream`); `--thinking-budget N` включает расширенное мышление и
выводит рассуждения в stderr. См.
[Руководство по библиотеке](library-guide.md#anthropic-compatible-messages-api)
для Go API.

## Терминальный UI

```bash
go-z-ai tui
```

Запускает полноэкранный терминальный UI с вкладками Chat, Models, Usage,
Accounts, Coding, Media и Tools — та же функциональность, что и у CLI-команд
выше, в одном интерактивном сеансе.
