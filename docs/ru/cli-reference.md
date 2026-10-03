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
- [Эндпоинт Responses (протокол Codex)](#эндпоинт-responses-протокол-codex)
- [Терминальный UI](#терминальный-ui)

## Глобальные флаги

Эти флаги применяются ко всем командам:

| Flag | Описание |
|---|---|
| `--api-key string` | API-ключ Z.AI (или переменная окружения `ZAI_API_KEY`) |
| `--account string` | Использовать сохранённый аккаунт по имени для этой команды (см. [Аккаунты и квоты](accounts-and-quota.md)) |
| `--region string` | Региональный шлюз: `global` (api.z.ai, по умолчанию) или `china` (open.bigmodel.cn). Псевдонимы: `cn`, `bigmodel`, `west`. Либо переменная окружения `ZAI_REGION`. Неизвестные значения приводят к global. |
| `--base-url string` | Корень chat/PaaS API (или переменная окружения `ZAI_API_BASE_URL`). По умолчанию — корень региона, например `https://api.z.ai/api/paas/v4` |
| `--china-api-key string` | Ключ open.bigmodel.cn для Embeddings/Moderations (или `ZAI_CHINA_API_KEY`; при отсутствии берётся `--api-key`) |
| `--monitor-timezone string` | Часовой пояс, в котором работает API квоты/использования (monitor) (или `ZAI_MONITOR_TIMEZONE`; по умолчанию CST/UTC+8). Названия IANA, `UTC` или смещения вроде `+8` |
| `--config string` | Файл конфигурации (по умолчанию: `.env`) |
| `-v`, `--version` | Печатает версию и выходит. Релизные сборки (ldflags GoReleaser) печатают тег; сборки для разработки печатают `dev` с коммитом и датой сборки. |

Большинство команд, возвращающих результат, принимают `--format text|json`
(команды `models` называют текстовый режим `table`; `embeddings` и
`moderations` по умолчанию используют `json`). У нескольких команд, которые
лишь выполняют действие, `--format` нет: `accounts add|use|remove`,
`coding auth|load|unload|doctor`, `coding mcp add|remove`, `audio speech`,
`files download` и `validate`. Сообщения о прогрессе и статусе идут в stderr,
поэтому с `--format json` stdout остаётся корректным JSON, который можно
передать в `jq`.

`--region` (или `ZAI_REGION`) выбирает шлюз для каждого эндпоинта, с которым
общается CLI: чат и остальные PaaS-сервисы, Anthropic-совместимая поверхность,
квота/использование, аккаунт (biz), агенты и определение типа аккаунта.
Установите `china`, если ваш ключ был выдан на `open.bigmodel.cn`.
`--base-url` переопределяет только корень chat/PaaS и там побеждает значение
региона по умолчанию. Embeddings и Moderations всегда используют
`open.bigmodel.cn`. Когда не заданы ни `--region`, ни `ZAI_REGION`,
применяется регион сохранённого аккаунта. См.
[Аккаунты и квоты § Региональные шлюзы](accounts-and-quota.md#региональные-шлюзы-apizai--openbigmodelcn).

## Чат

```bash
go-z-ai chat create <message> [flags]
go-z-ai chat async-result <task-id>
```

`chat create` — основная точка входа. Настройки сэмплинга (`--temperature`,
`--top-p`, `--max-tokens`) по умолчанию берутся от самой модели, если оставить
`0`.

| Flag | Назначение |
|---|---|
| `--model string` | По умолчанию — модель чата по умолчанию из каталога (`client.DefaultModel`, сейчас `glm-5.3`) |
| `--system string` | Системное сообщение |
| `--stream` | Потоковая передача token за token. С `--format json` — один фрагмент на строку |
| `--async` | Отправить без ожидания; опрашивать через `chat async-result <task-id>` |
| `--temperature float`, `--top-p float`, `--max-tokens int` | Управление сэмплингом (`0` = значение модели по умолчанию) |
| `--do-sample` | Сэмплировать (по умолчанию `true`); `--do-sample=false` — жадное декодирование |
| `--stop strings` | Стоп-последовательность (API учитывает одну) |
| `--thinking string` | `enabled` или `disabled` (модели GLM-5.3 всегда рассуждают) |
| `--effort string` | Reasoning effort: `max\|xhigh\|high\|medium\|low\|minimal\|none`. Проверяется по записи модели в каталоге: GLM-5.3 принимает `low`, `high`, `max`; GLM-5.2 принимает каждый уровень |
| `--show-reasoning` | Печатать рассуждения (в stderr в текстовом режиме) |
| `--json` | Запросить ответ в виде JSON-объекта |
| `--json-schema string` | Ответ в виде JSON-объекта, соответствующего schema: `@file.json` или inline JSON. У Z.AI нет формата ответа `json_schema`, поэтому schema добавляется в системное сообщение |
| `--tool string` | Объявления инструментов для вызова функций: `@tools.json` или inline JSON-массив |
| `--tool-stream` | Потоково передавать аргументы вызова инструмента по мере генерации (GLM-4.6+) |
| `--image string`, `--video string`, `--file string` (повторяемые) | Прикрепить изображение, видео или документ: URL или `@path` к локальному файлу (base64) |
| `--format text\|json` | Формат вывода |

```bash
go-z-ai chat create "Summarize this in 3 bullets" --stream
go-z-ai chat create "Plan a migration" --effort max --show-reasoning
go-z-ai chat create "Extract fields" --json-schema @schema.json --format json
go-z-ai chat create "Describe this" --image @photo.jpg --model glm-5.3-flash
```

Вложения требуют модель с соответствующей возможностью (`video`, `file`,
`vision` в `go-z-ai models list`). Модель чата по умолчанию в каталоге
только текстовая; `glm-5.3-flash` (`client.DefaultVisionModel`) нативно
мультимодальна (вход: изображение, видео, файл).

Вызовы инструментов печатаются, а не выполняются CLI — см.
[Руководство по библиотеке § Вызов функций](library-guide.md#вызов-функций)
о Go-цикле `RunWithTools` с автоматическим выполнением.

`chat async-result`, `image status` и `video status` используют один
обработчик: в текстовом режиме статус задачи идёт в stderr, а результат
(сообщение, URL изображений или URL видео) — в stdout; `--format json` печатает
полный результат задачи.

> **Визуальная модель + вызов инструментов могут вернуть HTTP 401.** Из
> сообщений сообщества (например,
> [claude-code-router#1491](https://github.com/musistudio/claude-code-router/issues/1491))
> следует, что сочетание визуальной модели (`--image` на `glm-4.6v`/`glm-4.5v`)
> с инструментами вызова функций (`--tool`) в одном запросе на некоторых
> конфигурациях GLM отклоняется с 401 — аутентифицированный ключ всё равно
> падает только для этой комбинации. Если столкнулись с этим, разделите
> работу: используйте визуальную модель для шага с изображением и текстовую
> модель для шага с вызовом инструментов, вместо того чтобы отправлять
> изображения и инструменты вместе. Здесь это не воспроизведено на живом
> аккаунте.

## Модели

```bash
go-z-ai models list
go-z-ai models get <model-id>
go-z-ai models text | vision | free
```

Эндпоинт `/models` возвращает «голые» ID; размер контекста, максимальный
вывод, цены (USD за 1M token), возможности (`text`, `vision`, `video`, `file`,
`thinking`, `tools`, `code`, `ocr`, `audio`) и допустимые уровни reasoning
effort берутся из курируемого каталога в `pkg/client/models_catalog.go`.
Модели, которых нет в каталоге, всё равно отображаются, с `-` вместо неизвестных
значений. `models get` также показывает уровни effort.

## Аккаунты, использование и квота

Подробно описано в [Аккаунты и квоты](accounts-and-quota.md). Краткая справка:

```bash
go-z-ai accounts add <name> --api-key <key> [--type coding_plan|pay_as_you_go] [--region global|china] [--base-url-override URL] [--force]
go-z-ai accounts list [--format json] [--reveal]   # keys masked by default; --reveal for export
go-z-ai accounts use <name>
go-z-ai accounts show [name] [--format json] [--reveal]
go-z-ai accounts current                            # shorthand for 'accounts show' (active account)
go-z-ai accounts quota [--only name...]
go-z-ai accounts usage [--days N] [--today] [--metric model|tool|both] [--only name...]
go-z-ai accounts remove <name> [--yes]

go-z-ai account detect                              # type and region of the current key (free)
go-z-ai account status [--watch 5m]                 # can the key authenticate and spend?
go-z-ai account balance                             # pay-as-you-go wallet balance
go-z-ai account subscriptions                       # GLM Coding Plan subscriptions

go-z-ai usage quota                                 # Coding Plan quota windows, reset times, pace
go-z-ai validate                                    # free request (list models) to check the key
```

- `accounts add` определяет тип и регион аккаунта бесплатным опросом, если не
  задан `--type`. `--region` указывает, на каком шлюзе был выпущен ключ; при
  определении именно с него начинается опрос. С `--type` ничего не
  определяется, поэтому для китайского ключа передайте `--region china`.
- `accounts quota` и `accounts usage` охватывают все сохранённые аккаунты
  (ограничить можно через `--only`, повторяемый); аккаунты pay-as-you-go
  пропускаются, потому что monitor-эндпоинты есть только у coding plan.
  `accounts usage` группирует по часам при 8 днях или меньше и по дням при 9 и
  более (`--days` по умолчанию 14).
- `account status` проверяет ключ Coding Plan бесплатно через эндпоинт квоты.
  Ключу pay-as-you-go нужен один минимальный платный запрос (у Z.AI нет API
  проверки баланса для таких ключей), а `--watch` тарифицирует его при каждой
  повторной проверке.
- `account balance` и `account subscriptions` используют biz-эндпоинты, которые
  пока не верифицированы на живом аккаунте (см. [Дорожную карту](roadmap.md)).
- `usage quota` показывает каждое окно кредитов со временем сброса и pace, а
  также уведомление о пиковых часах (пн–пт 14:00–18:00 UTC+8). Других
  подкоманд у `usage` нет; для нескольких аккаунтов сразу используйте
  `accounts quota` и `accounts usage`.

## Инструменты для кода (GLM Coding Plan)

Настраивает Claude Code, Codex, OpenCode, Crush или Factory Droid на
использование вашего GLM Coding Plan. Полное руководство: [Инструменты для
кода](coding-tools.md).

```bash
go-z-ai coding auth <plan> <key> [--no-validate]   # validate + store a credential
go-z-ai coding auth revoke
go-z-ai coding auth reload <tool>     # re-push stored creds into a tool's config
go-z-ai coding load <tool> [--plan ...] [--key ...]   # write it into a tool's config
go-z-ai coding unload <tool>          # says so and succeeds if the tool isn't configured
go-z-ai coding status [--format json]
go-z-ai coding tools [--format json]  # list supported tools + install status
go-z-ai coding doctor                 # health check; exits non-zero on problems

go-z-ai coding mcp add <tool> [--server id]... [--plan ...] [--key ...]
go-z-ai coding mcp remove <tool> [--server id]...
go-z-ai coding mcp status [--format json]
```

ID инструментов: `claude-code`, `codex`, `opencode`, `crush`, `factory-droid`
(псевдонимы `claude`, `droid`, `factory`). Тарифы: `glm_coding_plan_global`,
`glm_coding_plan_china`.

`coding load` и `coding auth` принимают флаги тонкой настройки Claude Code:
`--haiku`, `--sonnet`, `--opus` (переопределить модель уровня),
`--no-model-mapping`, `--auto-compact-window`, `--max-thinking-tokens`,
`--max-output-tokens`.

`coding mcp` регистрирует четыре официальных MCP-сервера Z.AI; `--server`
(повторяемый) выбирает часть из них, а по умолчанию берётся каждый сервер,
который поддерживает инструмент. Codex принимает только локальный сервер Vision:
его клиент streamable-HTTP MCP отвергает размещённые (openai/codex#14793), так
что запрос такого сервера завершается ошибкой. Вызовы серверов расходуют квоту
тарифа.

| ID сервера | Что делает |
|---|---|
| `zai-mcp-server` | Vision: скриншоты, диаграммы, UI, изображения и видео. Запускается локально через `npx`, поэтому нужен Node.js |
| `web-search-prime` | Веб-поиск (размещённый) |
| `web-reader` | Загрузка и чтение веб-страниц (размещённый) |
| `zread` | Чтение и поиск по репозиториям GitHub (размещённый) |

Размещённые серверы аутентифицируются ключом тарифа в регионе тарифа.

## Файлы и пакетная обработка

```bash
go-z-ai files upload <file> [--purpose batch|user_data|agent|file-extract|code-interpreter|voice-clone-input]
go-z-ai files list [--purpose ...]
go-z-ai files delete <file-id>
go-z-ai files download <file-id> <output-path>

go-z-ai batch create <input-file-id> [--endpoint ...] [--auto-delete-input] [--metadata key=value]...
go-z-ai batch status <batch-id>
go-z-ai batch list [--after ...] [--limit N]
go-z-ai batch cancel <batch-id>
```

`files upload` по умолчанию использует `--purpose batch`. Пакетные задачи
обрабатывают множество запросов завершения чата или embeddings из JSONL-файла
асинхронно — сначала загрузите его, затем создайте пакетную задачу с
полученным file ID. `--auto-delete-input` удаляет входной файл по завершении
пакета; `--metadata` повторяем. `batch status` печатает ID выходного файла и
файла ошибок, которые вы получаете через `files download`.

## Генерация медиа

```bash
# Images — default model glm-image (cogview-4-250304 also supported)
go-z-ai image generate <prompt> [--model glm-image|cogview-4-250304] [--size ...] [--quality hd|standard] [--async]
go-z-ai image status <id>
# --quality: hd is the default (~20s); standard is faster (~5-10s).
# --async is supported by glm-image only.

# Video — always async (cogvideox-3 | viduq1-text | viduq1-image | viduq1-start-end | vidu2-image | vidu2-start-end | vidu2-reference)
go-z-ai video generate --prompt "..." [--model ...] [--image URL]... [--size ...] [--aspect-ratio ...] \
    [--duration N] [--audio] [--off-peak]
go-z-ai video status <id>
# --off-peak queues the task for off-peak processing at a lower price.
# Model-specific: --fps and --quality (cogvideox-3), --style (viduq1-text),
# --movement (Vidu), --aspect-ratio (Vidu text/reference). Default model cogvideox-3.

# Audio
go-z-ai audio transcribe <file> [--model ...] [--hotword word]... [--prompt ...]   # .wav/.mp3, <=25MB, <=30s
go-z-ai audio speech <text> <output-path> [--voice ...] [--speed N] [--audio-format wav|pcm]

# Voice cloning (pairs with audio speech --voice)
go-z-ai voice clone <voice-name> <sample-file-id> <preview-text>
go-z-ai voice list [--name ...] [--type OFFICIAL|PRIVATE]
go-z-ai voice delete <voice-id>
```

`audio transcribe` по умолчанию использует модель ASR из каталога
(`client.DefaultASRModel`), а `audio speech` — модель TTS
(`client.DefaultTTSModel`). `--voice` принимает системный голос (`tongtong`,
по умолчанию, `chuichui`, `xiaochen`, `jam`, `kazi`, `douji`, `luodo`) или ID
клонированного голоса; `--speed` — от 0,5 до 2. Файл-образец для `voice clone`
должен быть уже загружен через `files upload --purpose voice-clone-input`.

## Разбор документов и OCR

```bash
# Layout OCR (glm-ocr) — image/PDF into Markdown
go-z-ai ocr parse <file-or-url> [--start-page N] [--end-page N]
go-z-ai ocr handwriting <file> [--probability] [--language ...]

# Document parser (RAG/retrieval preprocessing) — a separate product from OCR
go-z-ai parser parse <file> <file-type>              # synchronous
go-z-ai parser create <file> <tool-type> <file-type> # async: lite|expert|prime
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
[Аккаунты и квоты § Региональные шлюзы](accounts-and-quota.md#региональные-шлюзы-apizai--openbigmodelcn),
почему так и что это значит для аутентификации. `--dimensions` применим только
к `embedding-3` (256, 512, 1024 или 2048). Rerank использует корень chat/PaaS
(`--base-url` либо значение `--region` по умолчанию); он не привязан к
китайскому хосту.

## Модерация контента

```bash
go-z-ai moderations check <text>
```

Направляется на `open.bigmodel.cn` — та же заметка, что и для Embeddings выше.

## Агенты

```bash
go-z-ai agents invoke <agent-id> <message> [--source-lang ...] [--target-lang ...] [--var key=value]...
go-z-ai agents async-result <agent-id> <async-id> [--conversation-id ID] [--var key=value]...
```

Вызывает специализированных агентов Z.AI (перевод, генерация слайдов/постеров,
шаблоны видеоэффектов). `--var` задаёт пользовательскую переменную агента и
повторяем; `--source-lang` и `--target-lang` — сокращения для переменных
`source_lang` и `target_lang`. `invoke` печатает ID диалога в stderr; передайте
его обратно в `async-result` через `--conversation-id`. Примечание: Agents API
возвращает HTTP 200, даже когда вызов падает на бизнес-уровне (например,
недостаточный баланс) — CLI сообщает о такой неудаче из тела ответа как об
ошибке команды.

## Инструменты (веб-поиск, reader, токенизатор)

```bash
go-z-ai tools web-search <query> [--engine ...] [--count N] [--recency ...] [--domain ...] [--content-size medium|high]
go-z-ai tools web-reader <url> [--no-images]
go-z-ai tools tokenizer <text> [--model ...]
```

Флаги `web-search`:

| Flag | Назначение |
|---|---|
| `--engine string` | `search-prime` (по умолчанию), `search_std`, `search_pro`, `search_pro_sogou`, `search_pro_quark` |
| `--count int` | Число результатов, 1–50 (по умолчанию 10) |
| `--recency string` | `oneDay`, `oneWeek`, `oneMonth`, `oneYear`, `noLimit` |
| `--domain string` | Ограничить результаты одним доменом |
| `--content-size string` | Размер содержимого результата: `medium` или `high` |

`tokenizer` считает token для одного пользовательского сообщения; `--model` по
умолчанию — модель чата по умолчанию из каталога.

## Эндпоинт, совместимый с Anthropic

```bash
go-z-ai anthropic messages <prompt> [--model ...] [--max-tokens 1024] \
    [--system ...] [--temperature ...] [--thinking-budget N] [--stream]
```

Вызывает Anthropic-совместимую поверхность Z.AI
(`/api/anthropic/v1/messages` в выбранном `--region`) — тот же эндпоинт, на
который GLM Coding Plan направляет Claude Code, — вместо OpenAI-стиля
`chat create`. `--model` по умолчанию — модель чата по умолчанию из каталога;
`--max-tokens` обязателен для Messages API и по умолчанию равен 1024. Печатает
текст сообщения (или стримит текстовые дельты с `--stream`; с `--format json` —
одно событие на строку); `--thinking-budget N` включает расширенное мышление и
выводит рассуждения в stderr. См.
[Руководство по библиотеке](library-guide.md#anthropic-совместимый-messages-api)
для Go API.

## Эндпоинт Responses (протокол Codex)

```bash
go-z-ai responses create <prompt> [--model ...] [--instructions ...] \
    [--effort low|high|max] [--max-output-tokens N] [--stream] [--show-reasoning]
```

Вызывает эндпоинт Z.AI с протоколом OpenAI Responses (`/api/v1/responses` в
выбранном `--region`) — поверхность, против которой настроен Codex,
документированная для GLM Coding Plan. `--model` по умолчанию — модель чата по
умолчанию из каталога, и `--effort` проверяется по ней. Печатает выходной
текст (вызовы инструментов и, с `--show-reasoning`, рассуждения идут в
stderr); `--stream` печатает текстовые дельты, а с `--format json` — одно
событие на строку. См. [Руководство по библиотеке](library-guide.md) для Go
API.

## Терминальный UI

```bash
go-z-ai tui
```

Запускает полноэкранный терминальный UI с вкладками Chat, Models, Usage,
Accounts, Coding, Media и Tools — та же функциональность, что и у CLI-команд
выше, в одном интерактивном сеансе. Нажмите `?` или `F1`, чтобы увидеть
клавиши.
