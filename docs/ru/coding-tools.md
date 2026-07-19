# Инструменты для кода (GLM Coding Plan)

`go-z-ai coding` настраивает сторонние ассистенты для кода на использование
вашего GLM Coding Plan вместо их стандартного провайдера. Это порт на Go
официальной утилиты `@z_ai/coding-helper` ("chelper") от Z.AI, использующий тот
же файл учётных данных, поэтому оба инструмента можно применять
взаимозаменяемо.

## Поддерживаемые инструменты

| Инструмент | Конфиг, в который он пишет |
|---|---|
| Claude Code | `~/.claude/settings.json` (+ флаг онбординга в `~/.claude.json`) |
| OpenCode | `~/.config/opencode/opencode.json` |
| Crush | `~/.config/crush/crush.json` |
| Factory Droid | `~/.factory/settings.json` |
| Cursor | Зависит от ОС — `~/Library/Application Support/Cursor/User/settings.json` на macOS, `~/.cursor/settings.json` (или `~/.config/Cursor/User/settings.json`) в остальных системах |

Выполните `go-z-ai coding tools`, чтобы увидеть статус установки и точные
резолвленные пути на вашей машине.

## Тарифы

| Идентификатор тарифа | Эндпоинт |
|---|---|
| `glm_coding_plan_global` | `https://api.z.ai` |
| `glm_coding_plan_china` | `https://open.bigmodel.cn` |

Выберите тот, где живёт ваша подписка GLM Coding Plan.

## Быстрый старт

```bash
# 1. Сохранить и провалидировать ключ GLM Coding Plan (один раз)
go-z-ai coding auth glm_coding_plan_global YOUR_KEY

# 2. Загрузить его в инструмент
go-z-ai coding load claude-code
# ID инструментов: claude-code, opencode, crush, factory-droid, cursor
# также работают алиасы: claude, droid, factory

# 3. Проверить, что всё настроено
go-z-ai coding status
go-z-ai coding doctor
```

Учётные данные хранятся в `~/.chelper/config.yaml` (побайтово совместим с
официальным Node-хелпером) — `coding auth` записывает их один раз, а `coding
load` читает оттуда для каждого инструмента, если не передавать
`--key`/`--plan` для переопределения.

Чтобы перестать использовать Z.AI для инструмента без потери сохранённых
учётных данных:

```bash
go-z-ai coding unload claude-code
```

Это удаляет только добавленные Z.AI-специфичные поля; остальную часть
существующего конфига он не трогает.

## Claude Code: сопоставление моделей

Официальный хелпер задаёт только `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`,
`API_TIMEOUT_MS` и `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`. Этот клиент по
умолчанию идёт дальше и также сопоставляет уровни моделей Claude Code с
конкретными моделями GLM через `ANTHROPIC_DEFAULT_*_MODEL`, в соответствии с
[задокументированной рекомендацией Z.AI](https://docs.z.ai/scenario-example/develop-tools/claude):

| Уровень Claude | Модель GLM по умолчанию |
|---|---|
| haiku | `glm-4.5-air` |
| sonnet | `glm-4.7` |
| opus | `glm-4.7` |

Переопределите любой уровень или полностью отключите сопоставление:

```bash
go-z-ai coding auth glm_coding_plan_global YOUR_KEY \
  --sonnet glm-5.2 --opus glm-5.2

go-z-ai coding auth glm_coding_plan_global YOUR_KEY --no-model-mapping
```

Также настраивается, всё опционально (0/опущено = не задавать env-переменную):

| Флаг | env-переменная | Зачем нужно |
|---|---|---|
| `--auto-compact-window int` | `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | По умолчанию 1 000 000 (размер контекста GLM-5.2); уменьшите (например, до 128000), если вы привязаны к модели с контекстом 128K |
| `--max-thinking-tokens int` | `MAX_THINKING_TOKENS` | Бюджет расширенного мышления |
| `--max-output-tokens int` | `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | Лимит на вывод |

Эти флаги сохраняются на команде `coding`, поэтому одинаково применяются к
`auth`, `load` и `reload`.

## Управление ключами

```bash
go-z-ai coding auth revoke              # очистить сохранённый ключ, сохранив выбор тарифа
go-z-ai coding auth reload <tool>       # повторно загрузить сохранённые учётные данные в инструмент
go-z-ai coding load <tool> --key OTHER_KEY --plan glm_coding_plan_china  # разовое переопределение
```

По умолчанию `coding auth` валидирует новый ключ через API перед сохранением
(реальный вызов `/models`). Пропустите это через `--no-validate`, если хотите
сохранить ключ офлайн (например, при автоматизации настройки машины, которую вы
ещё не проверяли на доступность сети).

## Vision MCP server

Мастер официального `@z_ai/coding-helper` имеет шаг "manage MCP services",
который этот клиент до сих пор не воспроизводил: Z.AI поставляет собственный
[Vision MCP Server](https://docs.z.ai/devpack/mcp/vision-mcp-server)
(`@z_ai/mcp-server`) — OCR скриншотов, диагностика скриншотов с ошибками,
понимание диаграмм/графиков, а также общий анализ изображений/видео через
GLM-4.6V, запускаемый по требованию через `npx`. `coding mcp` регистрирует его
в любом используемом вами инструменте:

```bash
go-z-ai coding mcp add claude-code     # использует сохранённый API-ключ
go-z-ai coding mcp add crush --key OTHER_KEY
go-z-ai coding mcp status              # в каких инструментах он настроен
go-z-ai coding mcp remove claude-code
```

**Требуется Node.js.** Сам сервер запускается через `npx -y @z_ai/mcp-server` —
собственная документация Z.AI сейчас рекомендует Node.js 22+, хотя npm-пакет
объявляет требование только 18+. `coding mcp add`/`doctor` выдают
предупреждение (но не блокируют работу), если `npx` не найден в `PATH`,
поскольку конфиг валиден с того момента, как появляется Node.js.

**Файл конфига MCP часто не совпадает с файлом учётных данных GLM.** В двух из
пяти инструментов MCP-серверы хранятся в отдельном файле от настроек
провайдера/API:

| Инструмент | Конфиг учётных данных | Конфиг MCP |
|---|---|---|
| Claude Code | `~/.claude/settings.json` | `~/.claude.json` |
| OpenCode | `opencode.json` | тот же файл, ключ `mcp` |
| Crush | `crush.json` | тот же файл, ключ `mcp` |
| Factory Droid | `~/.factory/settings.json` | `~/.factory/mcp.json` |
| Cursor | Зависящий от ОС `settings.json` | соседний `mcp.json` в той же директории |

Cursor не указан явно как поддерживаемый клиент Vision MCP в собственной
документации Z.AI — здесь используется общая форма MCP, которую Cursor
документирует для любого сервера; это должно работать, но не подтверждено
Z.AI для конкретно этого сервера.

## Doctor

```bash
go-z-ai coding doctor
```

Проверяет: сохранены ли учётные данные, выглядят ли они корректно, какие из
поддерживаемых инструментов установлены в `PATH` и какие из них уже имеют
конфигурацию Z.AI (включая Vision MCP server, если он зарегистрирован). Хороший
первый шаг, когда что-то не работает.
