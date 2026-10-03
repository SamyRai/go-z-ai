# Инструменты для кода (GLM Coding Plan)

`go-z-ai coding` настраивает сторонние ассистенты для кода на использование
вашего GLM Coding Plan вместо их стандартного провайдера. Это порт на Go
официальной утилиты `@z_ai/coding-helper` ("chelper") от Z.AI
([docs](https://docs.z.ai/devpack/extension/coding-tool-helper)),
использующий тот же файл учётных данных, поэтому оба инструмента можно
применять взаимозаменяемо.

## Поддерживаемые инструменты

| Инструмент | ID (псевдонимы) | Конфиг, в который он пишет |
|---|---|---|
| Claude Code | `claude-code` (`claude`) | `~/.claude/settings.json` (+ `~/.claude.json`: флаг онбординга и MCP-серверы) |
| Codex | `codex` | `~/.codex/config.toml` (+ `~/.codex/models.json`: метаданные модели) |
| OpenCode | `opencode` | `~/.config/opencode/opencode.json` |
| Crush | `crush` | `~/.config/crush/crush.json` |
| Factory Droid | `factory-droid` (`droid`, `factory`) | `~/.factory/settings.json` (MCP-серверы в `~/.factory/mcp.json`) |

Пути указаны относительно вашего домашнего каталога. Выполните
`go-z-ai coding tools`, чтобы увидеть статус установки (лежит ли бинарник
инструмента в `PATH`: `claude`, `codex`, `opencode`, `crush`, `droid`) и точные
разрешённые пути на вашей машине.

Cursor не поддерживается (ранние релизы писали файл настроек Cursor). Z.AI
документирует Cursor как настройку только через GUI: протокол OpenAI, ваш ключ
и coding-эндпоинт тарифа в качестве базового URL
([docs](https://docs.z.ai/devpack/tool/cursor)).

### Что получает конфиг каждого инструмента

`coding load <tool>` сливает тариф в собственный конфиг инструмента и не
трогает посторонние настройки.

- **Claude Code**: в `~/.claude/settings.json` — блок `env` с ключом,
  Anthropic-эндпоинтом тарифа, сопоставлением уровней и остальными
  переменными, перечисленными в разделе
  [Claude Code: сопоставление моделей](#claude-code-сопоставление-моделей).
  В `~/.claude.json` он задаёт `hasCompletedOnboarding: true`, если оно ещё не
  задано. Также он удаляет устаревший `ANTHROPIC_API_KEY` из блока `env`.
  Ручной эквивалент см. в [документации Claude Code](https://docs.z.ai/devpack/tool/claude).
- **Codex**: в `~/.codex/config.toml` — `model_provider = "ZAI"`,
  `model = <client.DefaultModel>`, `model_reasoning_effort` (самый сильный
  уровень, который принимает модель, `max` для GLM-5.3),
  `model_catalog_json = "~/.codex/models.json"` и таблица
  `[model_providers.ZAI]` с Responses-эндпоинтом тарифа как `base_url`, ключом
  как `experimental_bearer_token` и `wire_api = "responses"`. В
  `~/.codex/models.json` он пишет метаданные модели из каталога (уровни
  рассуждений, окно контекста, входные модальности), заменяя более старую
  запись для той же модели и сохраняя остальные. Это формат документации Z.AI
  и официального хелпера. ([docs](https://docs.z.ai/devpack/tool/codex))
- **OpenCode**: запись провайдера, названная по встроенному провайдеру тарифа
  OpenCode, — `zai-coding-plan` (тариф Global) или `zhipuai-coding-plan`
  (тариф China), — содержащая `options.apiKey`; запись другого тарифа
  удаляется. Он задаёт `$schema`, а `model` и `small_model` — как
  `<provider>/<client.DefaultModel>` и `<provider>/<client.DefaultFastModel>`
  только когда они не заданы или уже указывают на провайдера тарифа, так что
  ваш собственный выбор сохраняется.
  ([docs](https://docs.z.ai/devpack/tool/opencode))
- **Crush**: `providers.zai` с `id: "zai"`, `name: "ZAI Provider"`,
  coding-эндпоинтом тарифа как `base_url` и `api_key`. Модель не пишется;
  выберите её в UI Crush. ([docs](https://docs.z.ai/devpack/tool/crush))
- **Factory Droid**: две записи `customModels` для `client.DefaultModel`, одна
  по протоколу Anthropic (`provider: "anthropic"`, Anthropic-эндпоинт тарифа) и
  одна по протоколу OpenAI (`provider: "generic-chat-completion-api"`,
  coding-эндпоинт). `maxOutputTokens` берётся из каталога моделей. Отображаемые
  имена выглядят как `GLM-5.3 [GLM Coding Plan Global] - Anthropic`; любая
  существующая запись, чьё отображаемое имя содержит `GLM Coding Plan`,
  заменяется (это покрывает и записи, написанные официальным хелпером), прочие
  записи сохраняются. ([docs](https://docs.z.ai/devpack/tool/droid))

Как записываются файлы:

- API-ключ попадает в конфиг каждого инструмента открытым текстом, как этого
  ожидают сами инструменты. Файлы пишутся атомарно с правами `0600`.
- Перед первой перезаписью файла конфига рядом с ним сохраняется копия как
  `<file>.zai.bak`. Существующая резервная копия никогда не заменяется.
  Конфиги-симлинки пишутся сквозь ссылку, так что менеджеры dotfiles продолжают
  работать.
- Конфиги читаются и перезаписываются как JSON (с отступами, ключи
  отсортированы) либо как TOML для файла `.toml` (Codex). Перезапись TOML
  отбрасывает комментарии, как и у официального хелпера; резервная копия
  `.zai.bak` хранит оригинал. Конфиг, который не удаётся разобрать, приводит к
  сбою команды с ошибкой разбора; он не перезаписывается.
- `coding load` для неустановленного инструмента всё равно пишет его конфиг.

## Тарифы

| Идентификатор тарифа | Шлюз | OpenAI-совместимый эндпоинт | Anthropic-эндпоинт | Эндпоинт Responses (Codex) |
|---|---|---|---|---|
| `glm_coding_plan_global` | `https://api.z.ai` | `https://api.z.ai/api/coding/paas/v4` | `https://api.z.ai/api/anthropic` | `https://api.z.ai/api/v1` |
| `glm_coding_plan_china` | `https://open.bigmodel.cn` | `https://open.bigmodel.cn/api/coding/paas/v4` | `https://open.bigmodel.cn/api/anthropic` | `https://open.bigmodel.cn/api/v1` |

Выберите тот, где живёт ваша подписка GLM Coding Plan. Тариф определяет регион
всего, что пишет `coding` (эндпоинты инструментов, URL размещённых
MCP-серверов, режим сервера Vision). Все URL берутся из `client.Region` в
`pkg/client/region.go`. Глобальные флаги `--region`, `--api-key` и `--account`
к `coding` не применяются; используйте вместо них тариф и `--key`/`--plan`.

## Быстрый старт

```bash
# 1. Store and validate your GLM Coding Plan key (one-time)
go-z-ai coding auth glm_coding_plan_global YOUR_KEY

# 2. Load it into a tool
go-z-ai coding load claude-code
# tool IDs: claude-code, codex, opencode, crush, factory-droid
# aliases also work: claude, droid, factory

# 3. Check everything's wired up
go-z-ai coding status
go-z-ai coding doctor
```

Учётные данные хранятся в `~/.chelper/config.yaml` (побайтово совместим с
официальным Node-хелпером; ключи `lang`, `plan`, `api_key`) — `coding auth`
записывает их один раз, а `coding load` читает оттуда для каждого инструмента,
если не передавать `--key`/`--plan` для переопределения.

Чтобы перестать использовать Z.AI для инструмента без потери сохранённых
учётных данных:

```bash
go-z-ai coding unload claude-code
```

Это удаляет только Z.AI-специфичные поля, добавленные `load` (управляемые
переменные `env` у Claude Code, провайдер `ZAI` у Codex и — пока он остаётся
активным провайдером — его верхнеуровневые настройки модели, провайдер тарифа
и `model`/`small_model` тарифа у OpenCode, `providers.zai` у Crush, записи
`customModels` тарифа у Droid); остальную часть существующего конфига он не
трогает. `models.json` у Codex остаётся, как его оставляет официальный хелпер.
MCP-серверы он не удаляет (используйте `coding mcp remove`), а
`hasCompletedOnboarding` у Claude Code остаётся.

Если в конфиге инструмента нет тарифа Z.AI, `unload` сообщает
`<Tool> is not configured for a Z.AI plan; nothing to remove.` и завершается с
кодом 0, так что его безопасно запускать многократно в скриптах. Инструмент
считается настроенным, только когда его конфиг указывает на эндпоинт тарифа
(`ANTHROPIC_BASE_URL` у Claude Code, `model_providers.ZAI.base_url` у Codex,
`providers.zai.base_url` у Crush) либо содержит запись тарифа (OpenCode,
Droid); ключ, заданный для другого провайдера, не затрагивается.

Вкладка coding в TUI (`go-z-ai tui`) предлагает те же действия: `a` — auth,
`l` — load, `u` — unload, `m` — зарегистрировать MCP-серверы, которые
поддерживает инструмент, `r` — пересканировать. Она загружает с описанной ниже
настройкой Claude Code по умолчанию.

## Claude Code: сопоставление моделей

`load` пишет переменные, которые Z.AI документирует для Claude Code, включая
сопоставление уровней моделей Claude Code с моделями GLM через
`ANTHROPIC_DEFAULT_*_MODEL`
([docs](https://docs.z.ai/devpack/tool/claude)):

| Env-переменная | Значение |
|---|---|
| `ANTHROPIC_AUTH_TOKEN` | ваш ключ (отправляется как `Authorization: Bearer`, чего ожидает Anthropic-эндпоинт Z.AI; `ANTHROPIC_API_KEY` не используется) |
| `ANTHROPIC_BASE_URL` | Anthropic-эндпоинт тарифа |
| `API_TIMEOUT_MS` | `3000000` |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | `1` |
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | `client.DefaultFastModel` + `[1m]` (`glm-5.3-flash[1m]`) |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | `client.DefaultModel` + `[1m]` (`glm-5.3[1m]`) |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | `client.DefaultModel` + `[1m]` (`glm-5.3[1m]`) |
| `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | размер контекста основной модели, 1048576 (задайте своё значение флагом ниже) |
| `MAX_THINKING_TOKENS`, `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | только когда вы передаёте соответствующий флаг |

ID моделей берутся из `pkg/client/models_catalog.go`, так что обновление
каталога меняет то, что пишет `load`. Суффикс `[1m]` — соглашение Claude Code,
включающее для модели контекст в 1M token; он относится только к этим
переменным и никогда не попадает в API-запрос. `load` добавляет его к каждой
модели, у которой контекст в каталоге составляет 1M token или больше, что
покрывает оба значения по умолчанию, как и делает официальный хелпер.
Значения, переданные через `--haiku`/`--sonnet`/`--opus`, пишутся дословно,
так что для модели с контекстом 1M добавьте `[1m]` сами.

Страницы Z.AI не вполне согласованы в вопросе сопоставления:
`docs.z.ai/devpack/tool/claude` разделяет его (flash для haiku, флагман для
sonnet и opus, как выше), тогда как `docs.z.ai/devpack/latest-model` показывает
flash-модель для всех трёх уровней. Официальный хелпер (0.1.1) тоже пишет
flash-модель для всех трёх. Переопределите любой уровень или полностью
откажитесь от сопоставления:

```bash
# Use the flash model for every tier
go-z-ai coding load claude-code \
  --sonnet 'glm-5.3-flash[1m]' --opus 'glm-5.3-flash[1m]'

# Leave Claude Code's model selection alone
go-z-ai coding load claude-code --no-model-mapping
```

Флаги настройки, все опциональны:

| Флаг | Эффект |
|---|---|
| `--haiku`, `--sonnet`, `--opus` | Переопределить ID модели этого уровня |
| `--no-model-mapping` | Не писать три переменные `ANTHROPIC_DEFAULT_*_MODEL` |
| `--auto-compact-window int` | Задать `CLAUDE_CODE_AUTO_COMPACT_WINDOW`; по умолчанию — контекст основной модели; уменьшите (например, до 128000), если вы привязаны к модели с меньшим контекстом; `0` не пишет переменную |
| `--max-thinking-tokens int` | Задать `MAX_THINKING_TOKENS` (бюджет расширенного мышления); `0`/опущено = не задавать |
| `--max-output-tokens int` | Задать `CLAUDE_CODE_MAX_OUTPUT_TOKENS`; `0`/опущено = не задавать |

Эти флаги есть только у `coding load` и `coding auth`, и они действуют в
момент записи конфига инструмента: `coding load <tool>` и
`coding auth reload <tool>`. `coding auth <plan> <key>` лишь сохраняет
учётные данные и игнорирует их. На инструменты, кроме Claude Code, они не
влияют. Каждый `load` перед записью удаляет переменные, которыми управляет, так
что настройка предыдущего запуска не переносится: передавайте флаги заново
каждый раз.

## Управление ключами

```bash
go-z-ai coding auth glm_coding_plan_global YOUR_KEY   # validate and store
go-z-ai coding auth revoke              # clear the stored key, keep the plan choice
go-z-ai coding auth reload <tool>       # re-push stored creds into a tool
go-z-ai coding load <tool> --key OTHER_KEY --plan glm_coding_plan_china  # one-off override
```

`--plan` и `--key` есть у `coding load` и `coding mcp add`; каждый по
отдельности откатывается к сохранённому значению. Без сохранённых учётных
данных и без переопределений команда завершается с ошибкой
`no credentials configured`.

По умолчанию `coding auth` валидирует новый ключ через API перед сохранением:
реальный `GET /models` на coding-эндпоинте тарифа (таймаут 30 секунд). 401
сообщается как отклонение (`Z.AI rejected the key (401)`); сетевая ошибка
сообщается как сбой валидации. Пропустите валидацию через `--no-validate`,
если хотите сохранить ключ офлайн (например, при автоматизации настройки
машины, которую вы ещё не проверяли на доступность сети).

## MCP-серверы

В мастере официального `@z_ai/coding-helper` есть шаг "manage MCP services".
`coding mcp` делает то же для четырёх официальных MCP-серверов Z.AI для GLM
Coding Plan, регистрируя их в любом используемом вами инструменте:

| ID сервера | Тип | Что делает |
|---|---|---|
| `zai-mcp-server` | Локальный, `npx -y @z_ai/mcp-server` | Vision: OCR скриншотов, диагностика скриншотов с ошибками, UI-в-артефакт и проверки различий UI, понимание диаграмм и графиков, анализ изображений и видео ([docs](https://docs.z.ai/devpack/mcp/vision-mcp-server)) |
| `web-search-prime` | Размещённый, streamable HTTP | Веб-поиск ([docs](https://docs.z.ai/devpack/mcp/search-mcp-server)) |
| `web-reader` | Размещённый, streamable HTTP | Загрузка и чтение веб-страниц ([docs](https://docs.z.ai/devpack/mcp/reader-mcp-server)) |
| `zread` | Размещённый, streamable HTTP | Чтение и поиск по репозиториям GitHub ([docs](https://docs.z.ai/devpack/mcp/zread-mcp-server)) |

```bash
go-z-ai coding mcp add claude-code              # all four, using the stored credentials
go-z-ai coding mcp add opencode --server web-search-prime --server zread
go-z-ai coding mcp add crush --plan glm_coding_plan_china --key OTHER_KEY
go-z-ai coding mcp remove claude-code --server zread
go-z-ai coding mcp status                       # which tools have which servers
```

`--server` принимает ID сервера, повторяемый или через запятую; по умолчанию
берётся каждый сервер, который поддерживает инструмент, а неизвестный ID — это
ошибка со списком допустимых. `add` пишет запись заново, если она уже есть, что
обновляет ключ. `remove` удаляет только названные официальные записи и не
трогает остальные MCP-серверы в файле. `mcp status` читает MCP-файл каждого
инструмента и перечисляет найденные там ID официальных серверов.

**Codex принимает только сервер Vision.** MCP-клиент Codex на streamable-HTTP
считает фатальным пустой ответ, который размещённые серверы Z.AI отправляют на
`notifications/initialized`
([openai/codex#14793](https://github.com/openai/codex/issues/14793)), поэтому
официальный хелпер скрывает их для Codex, и `coding mcp add codex` делает то
же: по умолчанию он регистрирует Vision, а размещённый сервер отвергает с этой
причиной, ничего не записывая.

Размещённые серверы находятся по адресу `https://<gateway>/api/mcp/<name>/mcp`,
где `<name>` — одно из `web_search_prime`, `web_reader`, `zread`;
аутентификация — `Authorization: Bearer <your key>`. Сервер Vision запускается
локально, и ему передаётся ключ как `Z_AI_API_KEY`, а также `Z_AI_MODE`: `ZAI`
для тарифа Global, `ZHIPU` для тарифа China. Этот переключатель определяет,
какую платформу вызывает сервер Vision, так что китайский ключ при `ZAI` попал
бы на `api.z.ai`.

Вызовы этих серверов расходуют квоту вашего тарифа. На тарифе с кредитами Web
Search, Web Reader и Zread стоят 1,2 кредита за вызов
([docs](https://docs.z.ai/devpack/overview)); см.
[Аккаунты и квоты](accounts-and-quota.md).

**Для сервера Vision требуется Node.js.** Он запускается через `npx`.
Собственная документация Z.AI рекомендует Node.js 22+, хотя npm-пакет
объявляет требование 18+. `coding mcp add` выдаёт предупреждение (но не
блокирует работу), если `npx` не найден в `PATH`, а вы выбрали сервер Vision,
а `coding doctor` и `coding mcp status` печатают ту же заметку, поскольку
конфиг валиден с того момента, как появляется Node.js.

**Файл конфига MCP часто не совпадает с файлом учётных данных GLM.** В двух из
пяти инструментов MCP-серверы хранятся в отдельном файле от настроек
провайдера/API, и у каждого инструмента своя форма записи:

| Инструмент | Конфиг учётных данных | Конфиг MCP | Форма записи |
|---|---|---|---|
| Claude Code | `~/.claude/settings.json` | `~/.claude.json`, ключ `mcpServers` | `{type: "stdio", command, args, env}` / `{type: "http", url, headers}` |
| Codex | `~/.codex/config.toml` | тот же файл, таблица `mcp_servers` | `{type: "local", command, args, env}`; размещённые серверы не поддерживаются |
| OpenCode | `opencode.json` | тот же файл, ключ `mcp` | `{type: "local", command: [...], environment}` / `{type: "remote", url, headers}` |
| Crush | `crush.json` | тот же файл, ключ `mcp` | `{type: "stdio", ...}` / `{type: "http", ...}` |
| Factory Droid | `~/.factory/settings.json` | `~/.factory/mcp.json`, ключ `mcpServers` | как у Claude Code, плюс `disabled: false` |

## Статус и вывод JSON

```bash
go-z-ai coding status          # stored plan, whether a key is stored, every tool
go-z-ai coding tools           # IDs, commands, install status, config paths
go-z-ai coding mcp status      # official MCP servers per tool
go-z-ai coding status --format json
```

`status`, `tools` и `mcp status` принимают `--format json` (по умолчанию
`text`). `tools` и `mcp status` печатают массив с одним объектом на
инструмент; `status` печатает
`{"credentials": {"plan": ..., "key_stored": true}, "tools": [...]}` — ни одна
часть ключа не выводится ни здесь, ни в `doctor`. У каждого объекта инструмента есть `id`, `name`,
`command`, `installed`, `config_path`, `configured` и, если присутствуют,
`plan`, `model_map` (сопоставление уровней Claude Code), `mcp_servers` и
`error` (конфиг, который не удалось прочитать, например некорректный JSON;
один нечитаемый инструмент не мешает сообщить об остальных).

```text
Stored credentials
==================
  Plan: GLM Coding Plan (Global)
  Key:  stored

Coding tools
============
  Claude Code    installed       Z.AI · GLM Coding Plan (Global) · MCP: zai-mcp-server, web-search-prime, web-reader, zread
                 models: haiku=glm-5.3-flash[1m] sonnet=glm-5.3[1m] opus=glm-5.3[1m]
  Codex          installed       Z.AI · GLM Coding Plan (Global) · MCP: zai-mcp-server
  OpenCode       not installed   native config
  Crush          not installed   native config
  Factory Droid  not installed   native config
```

`native config` означает, что в конфиге инструмента нет тарифа Z.AI.

## Doctor

```bash
go-z-ai coding doctor
```

Проверяет, что тариф и ключ сохранены, что конфиг каждого поддерживаемого
инструмента можно прочитать и что хотя бы один поддерживаемый инструмент
установлен в `PATH`; также отмечает, доступен ли `npx` для MCP-сервера Vision.
Каждая находка — одна строка (`✓` — ок, `⚠` — проблема, `ℹ` — заметка).
Хороший первый шаг, когда что-то не работает.

`doctor` завершается со статусом 1 и сообщением `N problem(s) found`, когда
находит любую проблему: нет сохранённых учётных данных, конфиг инструмента
невозможно разобрать или в `PATH` нет ни одного поддерживаемого инструмента.
Отсутствие `npx` — лишь заметка и не меняет статус выхода, так что `doctor`
пригоден как проверка в CI или при подготовке окружения.

## Соответствие политике и правила использования ⚠️

Coding-эндпоинт Z.AI (`/api/coding/paas/v4`) ограничен
[политикой использования](https://docs.z.ai/devpack/usage-policy) до
«официально поддерживаемых инструментов», а «доступ на основе SDK» прямо
запрещён. Три нарушения приводят к блокировке аккаунта. Неопознанные сторонние
клиенты на сервере неотличимы от запрещённого доступа
([pi#4187](https://github.com/earendil-works/pi/issues/4187)).
[Обзор тарифа](https://docs.z.ai/devpack/overview) говорит то же: тариф
предназначен для официально поддерживаемых инструментов.

`go-z-ai` смягчает это двумя способами:

1. **Подкоманда `coding` настраивает инструменты, которые документирует
   Z.AI** (у Claude Code, Codex, OpenCode, Crush и Factory Droid есть по
   странице в `docs.z.ai/devpack/tool/`), в их родные форматы конфигов — это
   то же самое, что делает собственный `@z_ai/coding-helper` от Z.AI. Сама
   настройка — поддерживаемый путь; `go-z-ai` лишь автоматизирует её из
   Go-бинарника. Какие инструменты входят в поддерживаемый список политики —
   решает Z.AI; актуальный список смотрите на странице политики.
2. **Каждый запрос, который `go-z-ai` делает сам, несёт идентифицирующий
   заголовок `User-Agent: go-z-ai/<version>`** (переопределяется через
   `Config.UserAgent` для downstream-приложений, прокси и MCP-серверов,
   которым нужен собственный идентификатор). Это минимальная гигиена,
   отличающая go-z-ai от анонимного/запрещённого доступа.

Чего это **не** делает:

- Это **не** делает go-z-ai «официально поддерживаемым инструментом» — такой
  статус может присвоить только Z.AI. До тех пор использование подкоманды
  `coding` для настройки поддерживаемого инструмента — соответствующий
  политике путь; использование `pkg/client` напрямую против
  `/api/coding/paas/v4` из собственной интеграции — на собственный риск
  пользователя.
- Это **не** маскирует и не обходит обнаружение. Заголовок честно
  идентифицирует клиента; цель — быть добропорядочным гражданином, а не
  скрываться.

Если вы создаёте downstream-инструмент, прокси или MCP-сервер поверх
`pkg/client`, задайте отличительный `Config.UserAgent` (например,
`"my-tool/1.0 (go-z-ai)"`), чтобы Z.AI мог отдельно идентифицировать ваш
трафик, а вы унаследовали добропорядочное значение по умолчанию от go-z-ai, а
не ослабили его.
