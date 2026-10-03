# go-z-ai

Z.AI (Zhipu AI / BigModel) платформасы өчен Go **CLI**, **китапханә** һәм
**TUI** — бөтен GLM модель өслеген бер инструментта, шулай ук `@z_ai/coding-helper`’ның
Go-порты: Claude Code, Codex, OpenCode, Crush һәм Factory Droid’ны сезнең
GLM Coding Plan’га тоташтыра.

[English](README.md) | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | **Татарча** | [Türkçe](README.tr.md)

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
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv() // ZAI_API_KEY’ны укый
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

Күбрәк әзер программалар — инструментларны чакыру, структураләштерелгән чыгыш,
визуаль кертү, асинхрон рәсем тикшерү, Anthropic `/v1/messages` эндпоинты —
[`examples/`](examples/) каталогында урнашкан.

## Мөмкинлекләр

- **Чат** — агымлы тапшыру (китапханәдә — Go итераторы), фикер йөртүне идарә
  итү (`reasoning_effort`, thinking кабызу/сүндерү), функция/инструмент чакыру
  (өстәвенә эчке `web_search`, `retrieval` һәм `mcp` инструмент төрләре),
  схемасы промптта бирелгән JSON-объект чыгышы, мультимодаль кертү (рәсем,
  видео, файл) һәм **Anthropic-белән туры килүче `/v1/messages`** эндпоинты
  (шул ук Claude Code GLM Coding Plan’га тоташканда кулланган).
- **Модельләр** — `go-z-ai models` һәм `client.CatalogEntry` артындагы сайланма
  каталог (контекст тәрәзәсе, максималь чыгыш, бәяләр, мөмкинлекләр, кабул
  ителә торган фикер йөртү дәрәҗәләре); килешү буенча модельләр
  [түбәндә](#килешү-буенча-модельләр) күрсәтелгән.
- **Медиа** — рәсем генерациясе, видео генерациясе (һәрвакыт асинхрон, теләк
  буенча йөкләнеш аз сәгатьләрдә чиратка кую белән), аудио язып алу, TTS һәм
  GLM-TTS тавыш клонлау.
- **Документларны аңлау** — layout OCR, кулъязма OCR һәм RAG-өчен алдан
  эшкәртү өчен документ парсеры.
- **Эзләү** — эмбеддинглар, ререйтинг, эчке веб-эзләү / веб-укучы /
  токенизатор инструментлары.
- **Модерацияләр** — China-платформа эндпоинты аша контент модерациясе.
  Эмбеддинглар һәм модерацияләр `open.bigmodel.cn` аша хезмәт күрсәтелә һәм
  аккаунт хокукларына бәйле рәвештә чикләнергә мөмкин; карагыз
  [Юл картасы һәм чикләр](docs/ru/roadmap.md).
- **Агентлар** — Z.AI’ның махсус агентлары (тәрҗемә, слайд/постер
  генерациясе, видео эффектлар).
- **Пакетлы эшләр һәм файллар** — чат тәмамлау өчен JSONL пакетлы эшләр,
  файл йөкләү/исемлек/йөкләп алу/бетерү.
- **GLM Coding Plan** — аккаунт төрен билгеләү, кредитларга нигезләнгән
  квота/куллану мониторингы, күп аккаунт идарәсе һәм Claude Code, Codex,
  OpenCode, Crush һәм Factory Droid’ны сезнең подпискаңызга тоташтыру һәм
  Z.AI’ның дүрт рәсми MCP серверын (Vision, веб-эзләү, веб-укучы, Zread)
  теркәү өчен `go-z-ai coding`.
- **DX** — тулы экранлы терминал интерфейсы (`go-z-ai tui`: чат, модельләр,
  куллану, аккаунтлар, coding, медиа һәм инструментлар вкладкалары), һәр
  эндпоинтны сайлаучы бер `--region` күчергече (`api.z.ai` ↔ `open.bigmodel.cn`),
  экспоненциаль кичектерү һәм джиттер белән автоматик кабатлау
  (`Retry-After`’ны исәпкә алып), шулай ук Z.AI хата кодлары тәкъдим ителгән
  типлаштырылган `APIError` (билгесез кодлар HTTP статусына кайта).

### Килешү буенча модельләр

Чакыручылар ID-ларны кодка кадаклап куймыйча, килешү буенча кыйммәтләрне
`pkg/client` константаларыннан сайлый:

| Константа | Модель | Нәрсә өчен |
|---|---|---|
| `client.DefaultModel` | `glm-5.3` | Чат, Anthropic эндпоинты, токенизатор. 1M контекст, фикер йөртү дәрәҗәләре `low`/`high`/`max` |
| `client.DefaultFastModel`, `client.DefaultVisionModel` | `glm-5.3-flash` | Тиз, арзан дәрәҗә; нигездә мультимодаль (рәсем, видео, файл кертү) |
| `client.DefaultOCRModel` | `glm-ocr` | `ocr`, `Layout().Parse` |
| `client.DefaultASRModel` | `glm-asr-2512` | `audio transcribe` (30 секундка кадәр клиплар) |
| `client.DefaultTTSModel` | `glm-tts` | `audio speech` |
| `client.ModelGLMImage`, `client.ModelCogView4` | `glm-image` (килешү буенча), `cogview-4-250304` | `image generate` |
| `client.VideoModels` | `cogvideox-3` (килешү буенча), `viduq1-*`, `vidu2-*` | `video generate` |

Каталог шулай ук `glm-5.3-flashx`, `glm-5.2`, GLM-4.x гаиләсен һәм бушлай
дәрәҗәләрне (`go-z-ai models free`) үз эченә ала. Бу — сайланма снимок (соңгы
яңартылу 2026-10-02) — `go-z-ai models list` контекстны, максималь чыгышны,
бәяләрне, мөмкинлекләрне һәм фикер йөртү дәрәҗәләрен күрсәтә, ә `/models`’тан
алынган җанлы кыйммәтләр каталог кыйммәтләреннән өстенрәк.

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
go-z-ai chat create "..." --stream          # чат (агымлы тапшыру, фикер йөртү дәрәҗәсе, инструментлар, рәсем/видео/файл кертү, JSON чыгышы)
go-z-ai anthropic messages "..." --stream   # Anthropic-белән туры килүче /v1/messages
go-z-ai responses create "..." --stream     # OpenAI Responses протоколы (/api/v1, аны Codex куллана)
go-z-ai image|video|audio|voice ...         # медиа генерациясе, язып алу, TTS, клонлау
go-z-ai ocr|parser ...                      # OCR + документ парсингы
go-z-ai embeddings|rerank|moderations ...   # эзләү + контент модерациясе
go-z-ai models list                         # модель каталогы: контекст, бәяләр, мөмкинлекләр, фикер йөртү дәрәҗәләре
go-z-ai account detect|status               # аскыч төре (coding plan / куллану буенча түләү), төбәк, торыш
go-z-ai accounts add|use|quota|usage ...    # күп аккаунт + GLM Coding Plan мониторингы
go-z-ai usage quota                         # хәзерге аскыч өчен GLM Coding Plan квота тәрәзәләре
go-z-ai coding auth|load|doctor|mcp ...     # Claude Code / Codex / OpenCode / Crush / Factory Droid’ны GLM Coding Plan’га тоташтыру
go-z-ai tui                                 # тулы экранлы терминал интерфейсы (өстәге барысы)
go-z-ai validate                            # аскычыгызның эшләвен раслагыз (бушлай сорау)
```

Нәтиҗә чыгаручы күпчелек команда `--format text|json` кабул итә (JSON —
stdout’ка, прогресс хәбәрләре — stderr’га, шуңа күрә `jq`’ка piping итә
аласыз). Тамыр `--region global|china` флагы (env `ZAI_REGION`) һәр эндпоинтны
сайлый; `--base-url` исә чат/PaaS тамырын гына алмаштыра.

### Код инструментлары

`go-z-ai coding` — Z.AI’ның `@z_ai/coding-helper` инструментының Go-порты.
Хупланган инструментлар: `claude-code`, `codex`, `opencode`, `crush` һәм
`factory-droid`.

```bash
go-z-ai coding auth glm_coding_plan_global <key>   # план аскычын тикшерү + саклау (яки glm_coding_plan_china)
go-z-ai coding load claude-code                    # инструментның конфигурациясен язу (шулай ук: codex, opencode, crush, factory-droid)
go-z-ai coding mcp add claude-code                 # рәсми MCP серверларын теркәү (сайлау өчен --server)
go-z-ai coding doctor                              # сәламәтлек тикшерүе; проблемалар булса нуль булмаган код белән чыга
```

- Claude Code өчен haiku дәрәҗәсе `client.DefaultFastModel`’га, ә sonnet/opus
  дәрәҗәләре `client.DefaultModel`’га чагыла; Claude Code’ның `[1m]` 1M-контекст
  суффиксы белән. `--haiku`, `--sonnet`, `--opus`, `--no-model-mapping`,
  `--auto-compact-window`, `--max-thinking-tokens` һәм `--max-output-tokens`
  моны `coding load` һәм `coding auth` өстендә көйли.
- `coding mcp add|remove <tool>` Z.AI’ның дүрт рәсми MCP серверын теркәй:
  `zai-mcp-server` (Vision; `npx` аша җирле эшли, Node.js кирәк),
  `web-search-prime`, `web-reader` һәм `zread` (хостингта, план аскычы белән
  аутентификацияләнә). Килешү буенча — инструмент хуплаган һәр сервер; Codex
  фәкать Vision’ны кабул итә (openai/codex#14793); `coding mcp status` һәр
  инструментта нәрсә барлыгын күрсәтә. MCP чакырулары план квотасын сарыф итә.
- Codex өчен `coding load codex` Responses протоколы (`/api/v1`) белән сөйләшүче
  ZAI провайдерын `~/.codex/config.toml` файлына, ә модель метаданнарын
  `~/.codex/models.json` файлына яза.

→ Тулы команда исемлеге: **[CLI белешмәсе](docs/ru/cli-reference.md)**

## Go китапханәсе буларак

`pkg/client` — китапханәнең үзе (бары тик стандарт китапханә); `pkg/observe` —
теләк буенча OpenTelemetry hook адаптеры. `internal/` астындагы
барлык нәрсә — тормышка ашыру детальләре. Кабатлау, таймаут, төбәк шлюзы
сайлау һәм хата сурәтләү үзәкләштерелгән — сервислар үзләренең
`http.Client`’ларын төзә алмый һәм чимал сораулар җибәрә алмый.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

// Мохиттән: ZAI_API_KEY, ZAI_API_BASE_URL, ZAI_REGION,
// ZAI_CHINA_API_KEY, ZAI_MONITOR_TIMEZONE.
c, err := client.NewClientFromEnv()

// Яки ачыктан-ачык:
c, err = client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Теләк буенча: Region, BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey,
    // UserAgent, Hooks
})
```

Сервислар, барысы да `c.<Сервис>().<Метод>(ctx, …)` рәвешендә:

| Аксессор | Каплый |
|---|---|
| `c.Chat()` | `Create`, `Stream` (итератор), `CreateAsync`, `RunWithTools` |
| `c.Anthropic()` | Anthropic-протоколлы `/v1/messages` (`Create`, `Stream`) |
| `c.Responses()` | `/api/v1` буенча OpenAI Responses протоколы — Codex өслеге (`Create`, `Stream`) |
| `c.Models()` | List, Get, текст/визуаль/бушлай фильтрлар |
| `c.Images()` / `c.Videos()` | Рәсем (`Generate`, `GenerateAsync`), видео (һәрвакыт асинхрон) |
| `c.Audio()` / `c.Voice()` | Язып алу, TTS, тавыш клонлау |
| `c.Layout()` / `c.FileParser()` | RAG өчен OCR + документтан-тексткә |
| `c.Files()` / `c.Batch()` | Йөкләү, пакетлы эшләр |
| `c.Agents()` | Z.AI махсус агентлары |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Эзләү + модерация |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan квотасы һәм куллануы, аккаунт мәгълүматы, аккаунт төрен билгеләү |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Асинхрон бурычлар өчен уртак тикшерү |

Белергә кирәк нәрсәләр:

- Килешү буенча кыйммәтләр каталогтан алына: `client.DefaultModel`,
  `client.DefaultFastModel`, `client.DefaultVisionModel` һәм OCR/ASR/TTS
  константалары. `ChatRequest.ReasoningEffort` модель кабул итә торган
  дәрәҗәләр буенча тикшерелә (GLM-5.3 өчен `client.EffortLow`, `EffortHigh`,
  `EffortMax`); `client.CatalogEntry(model)` һәм `Pricing.Cost(usage)` контекст
  зурлыкларын һәм бәяләрне бирә.
- `Chat().Stream`, `Anthropic().Stream` һәм `Responses().Stream` `iter.Seq2`
  итераторларын кайтара. Циклдан чыгу агымны ябып куя, ә агым эчендәге хата
  чанкы аны `*APIError` белән тәмамлый.
- Z.AI’да `json_schema` җавап форматы юк. Структураләштерелгән чыгыш өчен
  `ResponseFormat: client.JSONObjectFormat()` куегыз һәм
  `client.JSONSchemaPrompt(schema)`’ны система промптына салыгыз.
- `Config.Region` (`client.RegionGlobal` яки `client.RegionChina`) һәр URL’ны
  билгели; `Config.BaseURL` килешү буенча `Region.PaaSBaseURL()` була.

→ Мисаллар белән тулы API: **[Китапханә кулланмасы](docs/ru/library-guide.md)**
→ Генерацияләнгән белешмә: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Конфигурация

Танытмалар шушы өстенлек тәртибендә чишелә (иң югарысы өстен):

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
# ZAI_REGION=china                                   # әгәр аскычыгыз open.bigmodel.cn’да чыгарылган булса; һәр эндпоинтны сайлый
# ZAI_CHINA_API_KEY=...                              # аерым bigmodel.cn танытмасы (эмбеддинглар/модерацияләр)
# ZAI_MONITOR_TIMEZONE=UTC                           # квота/куллану API вакыт зонасы (килешү буенча UTC+8)
```

Китапханәнең `client.NewClientFromEnv()` функциясе шул ук үзгәрүчәннәрне
процесс мохитеннән укый (ул `.env` файлын йөкләми).

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
([PyPI `zai-sdk`](https://pypi.org/project/zai-sdk/)), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
һәм **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4))
өчен рәсми SDK’лар чыгара. Рәсми Go SDK **юк** — `go-z-ai` бу бушлыкны
тулдыра һәм шундый ук API өстендә CLI, TUI, төбәк шлюзларын алмаштыруны
(`api.z.ai` ↔ `open.bigmodel.cn`) һәм күп аккаунтлы GLM Coding Plan
идарәсен өсти.

> ℹ️ Репозиторий тамырындагы `zai-claude-config.json` — бу placeholder’лар
> булган **шаблон** (`"your-zai-api-key-here"`), ул `go-z-ai coding load claude-code`
> яза торган көйләүләрне күрсәтә. Программа аны укымый, ул чын конфиг түгел
> һәм аңа бернинди дә аутентификация мәгълүматы кертелмәгән; хәзерге модель
> чагылышын белү өчен `go-z-ai coding load claude-code` башкарыгыз.
>
> ⚠️ **Куллану сәясәте.** Z.AI кодлау очрак ноктасы «рәсми яктан хупланган
> кораллар» белән чикләнгән һәм SDK нигезендә керүне тыя; карагыз
> [Coding Tools — Compliance](docs/en/coding-tools.md#compliance--usage-policy-).
> `go-z-ai` һәр сораштыруда идентификацияләүче `User-Agent` башлыгын җибәрә,
> ә аның `coding` субкомандасы рәсми яктан хупланган коралларны тоташтыра.
> Рәсми рәвештә исемлеккә керткәнче кадәр, `pkg/client`ны кодлау очрак
> ноктасына үз интеграцияңездан турыдан-туры куллану — үз куркынычыгыз астында.

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
