# Z.AI API Client

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/view.html?uri=github.com/SamyRai/go-z-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Ein Go-CLI und eine Client-Bibliothek für die Z.AI-Plattform (Zhipu AI /
BigModel): Chat-Completions, Modelle, Bilder, Video, Audio, Embeddings,
Moderation, Reranking, Agenten, Batch-Jobs, Datei-Parsing, Kontoverwaltung und
Kontingente für GLM Coding Plan sowie ein Go-Port von `@z_ai/coding-helper` —
um Claude Code, OpenCode, Crush, Factory Droid und Cursor an Ihren GLM Coding
Plan anzubinden.

## Installation

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Dies erzeugt einen Binary namens `go-z-ai` in Ihrem `$GOPATH/bin`. Die
Beispiele unten verwenden den kürzeren Namen **`zai-client`** — verlinken
oder umbenennen Sie ihn:

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
# oder: mv "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
```

Setzt Go 1.26.4+ und einen [Z.AI API-Key](https://z.ai/manage-apikey) voraus.
Build aus dem Quellcode, Erstanmeldung und Fehlerbehebung:
**[Erste Schritte →](docs/en/getting-started.md)**

## Schnelles Beispiel

```bash
export ZAI_API_KEY=your_api_key_here
zai-client chat create "Erkläre Goroutinen in einem Absatz" --stream
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Erkläre Goroutinen in einem Absatz"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Weitere lauffähige Programme — Streaming, asynchrones Polling von Bildern,
der Anthropic-Endpunkt `/v1/messages` — finden Sie unter [`examples/`](examples/).

## Dokumentation

**[Vollständige Dokumentationsübersicht →](docs/en/README.md)**

| | |
|---|---|
| [Erste Schritte](docs/en/getting-started.md) | [CLI-Referenz](docs/en/cli-reference.md) |
| [Konten & Kontingent](docs/en/accounts-and-quota.md) | [Coding-Tools](docs/en/coding-tools.md) |
| [Bibliotheksanleitung](docs/en/library-guide.md) | [Fehlerbehandlung](docs/en/error-handling.md) |
| [Architektur](docs/en/architecture.md) | [Roadmap & Einschränkungen](docs/en/roadmap.md) |
| [Mitwirken](CONTRIBUTING.md) | [Sicherheitsrichtlinie](SECURITY.md) |
| [Verhaltenskodex](CODE_OF_CONDUCT.md) | [Changelog](CHANGELOG.md) |

## Abdeckung

Chat (Streaming, strukturierte Ausgabe, tiefer Denkprozess, Function Calling,
visuelle Eingabe), der Anthropic-kompatible Endpunkt `/v1/messages`, Modelle,
Bilder, Video, Audio (Transkription + TTS + Stimmklon), OCR und
Dokumenten-Parsing, Embeddings, Moderation, Reranking, Agenten, Dateien,
Batch-Jobs, Verwaltung von Nutzung/Kontingent/Mehrfachkonten für GLM Coding
Plan sowie eine Terminal-Vollbild-UI (`zai-client tui`). Die vollständige
Befehlsliste findet sich in der [CLI-Referenz](docs/en/cli-reference.md), die
Go-API in der [Bibliotheksanleitung](docs/en/library-guide.md).

## Verhältnis zu den offiziellen SDKs

Z.AI / Zhipu bieten offizielle SDKs für **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
und **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4)).
Es gibt **kein offizielles Go-SDK** — `go-z-ai` schließt diese Lücke und
schichtet eine CLI, eine TUI, das Umschalten regionaler Gateways
(`api.z.ai` ↔ `open.bigmodel.cn`) und das Verwalten mehrerer GLM-Coding-Plan-Konten
über derselben API-Oberfläche auf.

> ℹ️ `zai-claude-config.json` im Repo-Root ist eine **Vorlage** mit
> Platzhaltern (`"your-zai-api-key-here"`, der von
> `zai-client coding load claude-code` verwendet wird. Es handelt sich nicht
> um eine echte Konfiguration, und es werden keine Anmeldedaten mitgeliefert.

## Mitwirken

Siehe [CONTRIBUTING.md](CONTRIBUTING.md) — insbesondere die Konvention des
Projekts zur Live-Verifikation (aufgezeichnete API-Cassetten statt
handgestrickter Fixtures), falls Sie einen Service hinzufügen oder ändern.

## Lizenz

Apache License 2.0 — siehe [LICENSE](LICENSE).

## Support

- **Z.AI API-Dokumentation**: [https://docs.z.ai](https://docs.z.ai)
- **Issues**: [GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **Sicherheit**: siehe [SECURITY.md](SECURITY.md) — bitte melden Sie
  Schwachstellen nicht als öffentliche Issues.
