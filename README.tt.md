# Z.AI API клиенты

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/view.html?uri=github.com/SamyRai/go-z-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Z.AI (Zhipu AI / BigModel) платформасы өчен Go командалык юлы (CLI) һәм
клиент китапханәсе: чат тәмамлау, модельләр, рәсемнәр, видео, аудио,
эмбеддинглар, модерация, ререйтинг, агентлар, пакетлы эшләр, файлларны
эшкәртү, GLM Coding Plan аккаунтлары һәм квоталары белән идарә итү, шулай
ук `@z_ai/coding-helper`’ның Go-порты — Claude Code, OpenCode, Crush, Factory
Droid һәм Cursor’ны сезнең GLM Coding Plan’га тоташтыру өчен.

## Урнаштыру

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Бу `$GOPATH/bin` эчендә `go-z-ai` дигән бинарник булдыра. Астагы мисаллар
кыскарак исемне куллана — **`zai-client`** — символик сылтама яки исемен
үзгәртегез:

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
# яки: mv "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
```

Go 1.26.4+ һәм [Z.AI API-аскычы](https://z.ai/manage-apikey) кирәк. Чыганактан
җыю, беренче тапкыр аутентификация һәм хаталарны бетерү:
**[Башлау →](docs/ru/getting-started.md)**

## Тиз мисал

```bash
export ZAI_API_KEY=your_api_key_here
zai-client chat create "Горутиналарны бер абзацта аңлат" --stream
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Горутиналарны бер абзацта аңлат"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Күбрәк әзер мисаллар — агымлы тапшыру, рәсемнәрне асинхрон тикшерү, Anthropic
`/v1/messages` эндпоинты — [`examples/`](examples/) каталогында.

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

## Нәрсә каплана

Чат (агымлы тапшыру, структуralаңдырылган чыгыш, тирән уйлау, функция
чакыру, визуаль кертү), Anthropic-белән туры килүче `/v1/messages` эндпоинты,
модельләр, рәсемнәр, видео, аудио (язып алу + TTS + тавыш клонлау), OCR һәм
документларны эшкәртү, эмбеддинглар, модерация, ререйтинг, агентлар,
файллар, пакетлы эшләр, GLM Coding Plan куллануы/квотасы/күп аккаунт
идарәсе, шулай ук тулы экранлы терминал интерфейсы (`zai-client tui`).
Тулы команда исемлеген [CLI белешмәсендә](docs/ru/cli-reference.md), Go API’ны
[китапханә кулланмасында](docs/ru/library-guide.md) карагыз.

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
> булган **шаблон** (`"your-zai-api-key-here"`), аны `zai-client coding load
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
  җитешсезлекләрен ачық мәсьәлә итеп төрмәгез.
