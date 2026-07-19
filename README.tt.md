# go-z-ai

Z.AI (Zhipu AI / BigModel) платформасы өчен Go **CLI**, **китапханә** һәм
**TUI** — һемин GLM модель өслеген бер инструментта, шулай ук `@z_ai/coding-helper`’ның
Go-порты: Claude Code, OpenCode, Crush, Factory Droid һәм Cursor’ны сезнең
GLM Coding Plan’га тоташтыра.

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/viewer/?uri=github.com/SamyRai/go-z-ai)
[![Latest release](https://img.shields.io/github/v/release/SamyRai/go-z-ai)](https://github.com/SamyRai/go-z-ai/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Тиз мисал

```bash
# 1. Конфигурацияләгез (теләсә нинди вариант туры — env-үзгәрүчән, .env файлы яки --config <файл>)
export ZAI_API_KEY=your_api_key_here
# яки: cp .env.example .env, аннары .env файлын үзгәрт

# 2. CLI куллану
go-z-ai chat create "Горутиналарны бер абзацта аңлат" --stream
```

```go
// …яки китапханәне импортлагез — CLI кирәк түгел.
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Горутиналарны бер абзацта аңлат"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Күбрәк әзер программалар — агымлы тапшыру, асинхрон рәсем тикшерү, Anthropic
`/v1/messages` эндпоинты — [`examples/`](examples/) каталогында урнашкан.

## Мөмкинлекләр

- **Чат** — агымлы тапшыру, структураләштерелгән чыгыш (JSON Schema), тирән уйлау,
  функция/инструмент чакыру, визуаль кертү (`glm-4.6v`/`glm-4.5v`) һәм
  **Anthropic-белән туры килүче `/v1/messages`** эндпоинты (шул ук Claude Code
  һәм Cursor GLM Coding Plan’га тоташканда кулланган).
- **Медиа** — рәсем генерациясе, видео генерациясе (һәрвакыт асинхрон),
  аудио язып алу, TTS һәм GLM-TTS тавыш клонлау.
- **Документларны аңлау** — layout OCR, кулъязма OCR һәм RAG-өчен алдан
  эшкәртү өчен документ парсеры.
- **Эзләү** — эмбеддинглар, ререйтинг, эчке веб-эзләү / веб-укучы /
  токенизатор инструментлары.
- **Модерацияләр** — China-платформа эндпоинты аша контент модерациясе.
- **Агентлар** — Z.AI’ның махсус агентлары (тәрҗемә, слайд/постер
  генерациясе, видео эффектлар).
- **Пакетлы эшләр һәм файллар** — чат тәмамлау өчен JSONL пакетлы эшләр,
  файл йөкләү/исемлек/йөкләп алу.
- **GLM Coding Plan** — квота/куллану мониторингы, күп аккаунт идарәсе
  һәм Claude Code, OpenCode, Crush, Factory Droid һәм Cursor’ны сезнең
  подпискаңызга тоташтыру өчен `go-z-ai coding`.
- **DX** — тулы экранлы терминал интерфейсы (`go-z-ai tui`), төбәк шлюзын
  алмаштыру (`api.z.ai` ↔ `open.bigmodel.cn`), экспоненциаль кичектерү һәм
  джиттер белән автоматик кабатлау, шулай ук һәр Z.AI хата коды тәкъдим ителгән
  типлаштырылган `APIError`.

## Урнаштыру

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Бу сезнең `$GOPATH/bin` эчендә `go-z-ai` дигән бинарник булдыра.

```bash
# Мөмкин булган кыска псевдоним: ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Go 1.26.4+ һәм [Z.AI API-аскычы](https://z.ai/manage-apikey/apikey-list) кирәк. Чыганактан
җыю, беренче тапкыр аутентификация һәм хаталарны бетерү:
**[Башлау →](docs/ru/getting-started.md)**

## CLI буларак

Бер генә `go-z-ai` бинарнигы тулы өслекне каплый. Һәр команда `--help`
кабул итә; тиз күзәтү:

```bash
go-z-ai chat create "..." --stream          # чат (агымлы тапшыру, инструментлар, визуаль кертү, структураләштерелгән чыгыш)
go-z-ai anthropic messages "..." --stream   # Anthropic-белән туры килүче /v1/messages
go-z-ai image|video|audio|voice ...         # медиа генерациясе, язып алу, TTS, клонлау
go-z-ai ocr|parser ...                      # OCR + документ парсингы
go-z-ai embeddings|rerank|moderations ...   # эзләү + контент модерациясе
go-z-ai models list                         # модель каталогы + бәяләр
go-z-ai accounts add|use|quota|usage ...    # күп аккаунт + GLM Coding Plan мониторингы
go-z-ai coding auth|load|doctor|mcp ...     # Claude Code / Cursor һ.б.-ны GLM Coding Plan’га тоташтыру
go-z-ai tui                                 # тулы экранлы терминал интерфейсы (өстәге барысы)
go-z-ai validate                            # аскычыгызның бер чын шакырту белән эшләвен раслагыз
```

Нәтиҗә китерүче һәр команда `--format text|json` кабул итә (JSON — stdout’ка,
прогресс хәбәрләре — stderr’га, шуңа күрә `jq`’ка piping итә аласыз).

→ Тулы команда исемлеге: **[CLI белешмәсе](docs/ru/cli-reference.md)**

## Go китапханәсе буларак

`pkg/client` — бердәнбер җәмәгать импортланган пакет; `internal/` астындагы
барлык нәрсә — гамәли тормыш детальләре. Кабатлау, таймаут, төбәк шлюзы
сайлау һәм хата сурәтләү үзәкләштерелгән — сервислар үзләренең
`http.Client`’ларын төзә алмый һәм чимал сораулар җибәрә алмый.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Әйберле: BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey, Region
})
```

Сервислар, барысы да `c.<Сервис>().<Метод>(ctx, …)` рәвешендә:

| Аксессор | Каплый |
|---|---|
| `c.Chat()` | Тәмамлау, агымлы тапшыру, асинхрон, `RunWithTools` |
| `c.Anthropic()` | Anthropic-протоколлы `/v1/messages` (Create, CreateStream) |
| `c.Models()` | List, Get, текст/визуаль/бушлай фильтрлар |
| `c.Images()` / `c.Videos()` | Рәсем (синхрон/асинхрон), видео (һәрвакыт асинхрон) |
| `c.Audio()` / `c.Voice()` | Язып алу, TTS, тавыш клонлау |
| `c.Layout()` / `c.FileParser()` | RAG өчен OCR + документтан-тексткә |
| `c.Files()` / `c.Batch()` | Йөкләү, пакетлы эшләр |
| `c.Agents()` | Z.AI махсус агентлары |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Эзләү + модерация |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Usage()` / `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan мониторингы |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Асинхрон бурычлар өчен уртак тикшерү |

→ Мисаллар белән тулы API: **[Китапханә кулланмасы](docs/ru/library-guide.md)**
→ Генерацияләнгән белешмә: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Конфигурация

Өч ысул белән танытмаларны тәкъдим итә аласыз, болар шушы өстенлек тәртибендә
чишелә (иң югарысы өстен):

| Өйрәнмә | Кайчан кулланырга |
|---|---|
| `--api-key <key>` флагы | Бер тапкыр чакырулар, скриптлар, CI |
| `--account <исем>` флагы | [Сакланган аккаунтлар](docs/ru/accounts-and-quota.md) арасында алыштыру |
| `ZAI_API_KEY` env-үзгәрүчән (яки `.env` файлы) | Гадәттәге җирле shell кулланышы |
| Аккаунтлар саклагычының актив аккаунты | `go-z-ai accounts use <исем>`’дан соң |

`.env` файлы — иң таралган очрак — аннотацияләнгән шаблонны күчерегез һәм
аны үзгәртегез:

```bash
cp .env.example .env
# яки теләсә нинди файлны күрсәтегзез: go-z-ai --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # чат эндпоинтын алмаштыру
# ZAI_REGION=china                                   # әгәр аскычыгыз open.bigmodel.cn’да чыгарылган булса
# ZAI_CHINA_API_KEY=...                              # аерым bigmodel.cn танытмасы
# ZAI_ENV=production
```

→ Тулы белешмә (күп аккаунт, төбәк шлюзлары, квота тәрәзәләре):
**[Аккаунтлар һәм квоталар](docs/ru/accounts-and-quota.md)**

## Документация

**[Тулы документация индексы →](docs/ru/README.md)**

| | |
|---|---|
| [Башлау](docs/ru/getting-started.md) | [CLI белешмәсе](docs/ru/cli-reference.md) |
| [Аккаунтлар һәм квоталар](docs/ru/accounts-and-quota.md) | [Код инструментлары](docs/ru/coding-tools.md) |
| [Китапханә кулланмасы](docs/ru/library-guide.md) | [Хаталарны эшкәртү](docs/ru/error-handling.md) |
| [Архитектура](docs/ru/architecture.md) | [Юл картасы һәм чикләр](docs/ru/roadmap.md) |
| [Өлеш кертү](CONTRIBUTING.md) | [Иминлек сәясәте](SECURITY.md) |
| [Үз-үзеңне тоту кодексы](CODE_OF_CONDUCT.md) | [Үзгәрешләр журналы](CHANGELOG.md) |

## Рәсми SDK’лар белән бәйләнеш

Z.AI / Zhipu **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
һәм **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4))
өчен рәсми SDK’лар чыгара. Рәсми Go SDK **юк** — `go-z-ai` бу бушлыкны
тулдыра һәм шундый ук API өстендә CLI, TUI, төбәк шлюзларын алмаштыруны
(`api.z.ai` ↔ `open.bigmodel.cn`) һәм күп аккаунтлы GLM Coding Plan
идарәсен өсти.

> ℹ️ Репозиторий тамырындагы `zai-claude-config.json` — бу placeholder’лар
> булган **шаблон** (`"your-zai-api-key-here"`), аны `go-z-ai coding load
> claude-code` куллана. Ул чын конфиг түгел һәм аңа бернинди дә
> аутентификация мәгълүматы кертелмәгән.

## Өлеш кертү

[CONTRIBUTING.md](CONTRIBUTING.md) карагыз — аерым алганда, сервис өстәсәгез
яки үзгәртсәгез, проектның тере тикшерү кагыйдәсенә (кулдан язылган
фикстуралар урынына чын API-шакымаларны язу) игътибар итегез.

## Лицензия

Apache License 2.0 — [LICENSE](LICENSE) карагыз.

## Ярдәм

- **Z.AI API документациясе**: [https://docs.z.ai](https://docs.z.ai)
- **Мәсьәләләр**: [GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **Иминлек**: [SECURITY.md](SECURITY.md) карагыз — зинһар, иминлек
  җитешсезлекләрен ачык мәсьәлә итеп төрмәгез.
