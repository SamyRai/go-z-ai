# Начало работы

Здесь вы за пару минут пройдёте путь от нуля до своей первой команды
`go-z-ai`.

## 1. Установка

**Предварительные требования:** Go 1.26.4+, API-ключ Z.AI
([создайте его здесь](https://z.ai/manage-apikey/apikey-list)).

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Бинарник устанавливается как `go-z-ai`. Необязательный короткий псевдоним:

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Либо соберите из исходников сразу с нужным именем:

```bash
git clone https://github.com/SamyRai/go-z-ai.git
cd go-z-ai
go build -o go-z-ai .
```

Каким бы путём вы ни пошли, убедитесь, что `go-z-ai` разрешается и лежит в
вашем `PATH`:

```bash
go-z-ai --version
```

Далее предполагается, что бинарник называется `go-z-ai`.

## 2. Аутентификация

Выберите то, что подходит под ваш стиль работы. Методы разрешаются в следующем
порядке приоритета (высший побеждает):

| Метод | Когда использовать |
|---|---|
| Флаг `--api-key` | Разовые вызовы, скрипты, CI |
| Флаг `--account <name>` | Зарегистрировано несколько аккаунтов (см. [Аккаунты и квоты](accounts-and-quota.md)) |
| Переменная окружения `ZAI_API_KEY` (или файл `.env` в текущей директории) | Повседневная работа в локальной оболочке — типичный случай |
| Активный аккаунт из хранилища аккаунтов | Вы выполнили `accounts use <name>` и хотите, чтобы он применялся по умолчанию |

Для единственного ключа самый быстрый путь:

```bash
export ZAI_API_KEY=your_api_key_here
go-z-ai validate
```

`validate` выполняет один бесплатный запрос (получает список моделей) и
подтверждает, что ключ работает, прежде чем двигаться дальше.

Читаются следующие переменные окружения (у каждой есть соответствующий флаг):

| Переменная | Флаг | Назначение |
|---|---|---|
| `ZAI_API_KEY` | `--api-key` | Ваш API-ключ Z.AI |
| `ZAI_REGION` | `--region` | Региональный шлюз: `global` (api.z.ai, по умолчанию) или `china` (open.bigmodel.cn) |
| `ZAI_API_BASE_URL` | `--base-url` | Переопределяет только корень chat/PaaS API (по умолчанию — корень региона) |
| `ZAI_CHINA_API_KEY` | `--china-api-key` | Отдельный ключ open.bigmodel.cn для Embeddings/Moderations (если не задан, берётся `ZAI_API_KEY`) |

Также читается `ZAI_MONITOR_TIMEZONE` (`--monitor-timezone`); он важен только
для вывода квоты/использования.

Если ваш ключ был выпущен на китайской платформе Z.AI (`open.bigmodel.cn`),
укажите `--region china` (или `ZAI_REGION=china`). Регион выбирает хост для
каждого эндпоинта — чата, квоты / использования, аккаунта, агентов и
определения типа аккаунта, — поэтому без него эти вызовы идут на `api.z.ai`,
где ключ, выпущенный в Китае, может не пройти аутентификацию. `accounts add`
определяет регион за вас при регистрации ключа (с `--type` определение
пропускается, так что добавьте `--region china` самостоятельно). Embeddings и
Moderations всегда используют `open.bigmodel.cn`. Полная картина — в разделе
[Аккаунты и квоты § Региональные шлюзы](accounts-and-quota.md#региональные-шлюзы-apizai--openbigmodelcn).

## 3. Ваши первые команды

```bash
# See what models you have access to (context, prices, capabilities from the catalog)
go-z-ai models list

# Send a chat completion (uses the catalog's default chat model)
go-z-ai chat create "Explain goroutines in one paragraph"

# Stream the response token-by-token
go-z-ai chat create "Write a haiku about Go" --stream

# Detect whether the key is Coding Plan or pay-as-you-go, and on which gateway (free)
go-z-ai account detect

# Check that the key can spend (free for Coding Plan keys; one minimal billed
# request for pay-as-you-go keys), then your quota (Coding Plan accounts only)
go-z-ai account status
go-z-ai usage quota
```

Добавьте `--format json` к большинству команд для машиночитаемого вывода;
сообщения о ходе выполнения идут в stderr, поэтому stdout остаётся чистым.

Дальше:

- **Полный справочник по командам:** [Справочник по CLI](cli-reference.md)
- **Несколько аккаунтов / мониторинг квоты:** [Аккаунты и квоты](accounts-and-quota.md)
- **Подключение Claude Code / OpenCode / Crush / Factory Droid к вашему GLM Coding Plan:** [Инструменты для кода](coding-tools.md)
- **Использование проекта как библиотеки Go вместо CLI:** [Руководство по библиотеке](library-guide.md)
- **Полноэкранный терминальный интерфейс** (вкладки чата, моделей, использования, аккаунтов, кодинга, медиа и инструментов в одном месте): `go-z-ai tui`

## Устранение неполадок

**"API key is required"** — ни один из четырёх методов выше не дал ключа.
Перепроверьте `echo $ZAI_API_KEY` или явно укажите `--api-key`, чтобы
убедиться.

**"invalid API key" / HTTP 401** — ключ найден, но Z.AI его отклонил.
Перевыпустите его на [z.ai/manage-apikey](https://z.ai/manage-apikey/apikey-list).

**"Unknown Model" (error 1211) на `embeddings`/`moderations`/`rerank`/`voice`** —
почти всегда это ограничение по тарифу аккаунта, а не баг: в каталоге тарифа
вашего аккаунта нет этой модели. Выполните `go-z-ai models list`, чтобы
посмотреть, что реально доступно вашему ключу. Полное объяснение — в
[Аккаунты и квоты](accounts-and-quota.md).

**Что-то другое** — [откройте issue](https://github.com/SamyRai/go-z-ai/issues)
с точной командой и выводом ошибки (удалите ключ из текста).
