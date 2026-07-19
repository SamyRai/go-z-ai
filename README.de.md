# Z.AI API Client

Eine Go-**CLI**, **Bibliothek** und ein **TUI** für die Z.AI-Plattform
(Zhipu AI / BigModel) — jede Modell-Schnittstelle von GLM in einem einzigen
Werkzeug, plus ein Go-Port von `@z_ai/coding-helper`, der Claude Code,
OpenCode, Crush, Factory Droid und Cursor an Ihren GLM Coding Plan anbindet.

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | **Deutsch** | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/viewer/?uri=github.com/SamyRai/go-z-ai)
[![Latest release](https://img.shields.io/github/v/release/SamyRai/go-z-ai)](https://github.com/SamyRai/go-z-ai/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Schnelles Beispiel

```bash
# 1. Konfigurieren (eine dieser Varianten funktioniert — Umgebungsvariable, .env-Datei oder --config <Datei>)
export ZAI_API_KEY=your_api_key_here
# oder: cp .env.example .env, dann .env bearbeiten

# 2. CLI verwenden
zai-client chat create "Erkläre Goroutinen in einem Absatz" --stream
```

```go
// …oder die Bibliothek importieren — keine CLI erforderlich.
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

## Features

- **Chat** — Streaming, strukturierte Ausgabe (JSON Schema), Deep Thinking,
  Function/Tool-Aufrufe, Vision (`glm-4.6v`/`glm-4.5v`) und ein
  **Anthropic-kompatibler `/v1/messages`-Endpunkt** (derselbe, den Claude Code
  und Cursor ansprechen, wenn sie an einen GLM Coding Plan angebunden sind).
- **Medien** — Bildgenerierung, Videogenerierung (immer asynchron),
  Audiokranskription, TTS und GLM-TTS-Stimmklonung.
- **Dokumentenverständnis** — Layout-OCR, Handschrift-OCR und ein
  Dokumentenparser für die RAG-Vorverarbeitung.
- **Retrieval** — Embeddings, Reranking, integrierte Web-Such- /
  Web-Reader- / Tokenizer-Tools.
- **Moderationen** — Inhaltsmoderation über den Endpunkt der China-Plattform.
- **Agenten** — die spezialisierten Agents von Z.AI (Übersetzung,
  Folien-/Postergenerierung, Videoeffekte).
- **Batch & Dateien** — JSONL-Batch-Jobs für Chat-Completions,
  Datei-Upload/-Auflistung/-Download.
- **GLM Coding Plan** — Überwachung von Kontingent/Nutzung, Mehrfachkonten-Verwaltung
  und `zai-client coding`, um Claude Code, OpenCode, Crush, Factory Droid und
  Cursor an Ihr Abonnement anzubinden.
- **DX** — Terminal-Vollbild-UI (`zai-client tui`), Umschalten zwischen regionalen
  Gateways (`api.z.ai` ↔ `open.bigmodel.cn`), automatische Wiederholung mit
  Backoff + Jitter sowie ein typisiertes `APIError`, in dem jeder Z.AI-Fehlercode
  abgebildet ist.

## Installation

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Dies erzeugt einen Binary namens `go-z-ai` in Ihrem `$GOPATH/bin`. Die
Beispiele unten verwenden den kürzeren Namen **`zai-client`** — verlinken
oder benennen Sie ihn um:

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
# oder: mv "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
```

Setzt Go 1.26.4+ und einen [Z.AI API-Key](https://z.ai/manage-apikey/apikey-list) voraus.
Build aus dem Quellcode, Erstanmeldung und Fehlerbehebung:
**[Erste Schritte →](docs/en/getting-started.md)**

## Als CLI

Ein einzelner `zai-client`-Binary deckt die gesamte Oberfläche ab. Jeder Befehl
unterstützt `--help`; hier der Schnelldurchlauf:

```bash
zai-client chat create "..." --stream          # Chat (Streaming, Tools, Vision, strukturierte Ausgabe)
zai-client anthropic messages "..." --stream   # Anthropic-kompatibel /v1/messages
zai-client image|video|audio|voice ...         # Medien-Generierung, Transkription, TTS, Klonen
zai-client ocr|parser ...                      # OCR + Dokumenten-Parsing
zai-client embeddings|rerank|moderations ...   # Retrieval + Inhaltsmoderation
zai-client models list                         # Modellkatalog + Preise
zai-client accounts add|use|quota|usage ...    # Mehrfachkonten + GLM-Coding-Plan-Überwachung
zai-client coding auth|load|doctor|mcp ...     # Claude Code / Cursor / usw. an GLM Coding Plan anbinden
zai-client tui                                 # Terminal-Vollbild-UI (alles oben Genannte)
zai-client validate                            # mit einem echten Aufruf prüfen, ob Ihr Key funktioniert
```

Jeder Befehl, der Ergebnisse liefert, akzeptiert `--format text|json` (JSON geht
an stdout, Fortschrittsmeldungen an stderr, sodass Sie in `jq` weiterleiten
können).

→ Vollständige Befehlsliste: **[CLI-Referenz](docs/en/cli-reference.md)**

## Als Go-Bibliothek

`pkg/client` ist das einzige öffentlich importierbare Package; alles unter
`internal/` ist Implementierungsdetail. Retry, Timeout, Auswahl des regionalen
Gateways und Fehler-Mapping sind zentralisiert — Services bauen niemals ihren
eigenen `http.Client` und stellen keine rohen Requests ab.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Optional: BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey, Region
})
```

Services, alle nach dem Muster `c.<Service>().<Method>(ctx, …)`:

| Accessor | Deckt ab |
|---|---|
| `c.Chat()` | Completions, Streaming, Async, `RunWithTools` |
| `c.Anthropic()` | Anthropic-Protokoll `/v1/messages` (Create, CreateStream) |
| `c.Models()` | List, Get, Filter nach Text/Vision/kostenlos |
| `c.Images()` / `c.Videos()` | Bilder (sync/async), Video (immer async) |
| `c.Audio()` / `c.Voice()` | Transkription, TTS, Stimmklonung |
| `c.Layout()` / `c.FileParser()` | OCR + Dokument-zu-Text für RAG |
| `c.Files()` / `c.Batch()` | Upload, Batch-Jobs |
| `c.Agents()` | spezialisierte Agents von Z.AI |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Retrieval + Moderation |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Usage()` / `c.Quota()` / `c.Account()` / `c.Detection()` | GLM-Coding-Plan-Überwachung |
| `c.GetAsyncResult()` / `c.WaitForResult()` | gemeinsames Polling für asynchrone Aufgaben |

→ Vollständige API mit Beispielen: **[Bibliotheksanleitung](docs/en/library-guide.md)**
→ Generierte Referenz: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Konfiguration

Drei Wege, Anmeldedaten anzugeben, aufgelöst in dieser Reihenfolge
(höchste gewinnt):

| Methode | Wann verwenden |
|---|---|
| `--api-key <key>` Flag | Einmalige Aufrufe, Skripte, CI |
| `--account <name>` Flag | Wechseln zwischen [gespeicherten Konten](docs/en/accounts-and-quota.md) |
| `ZAI_API_KEY` Umgebungsvariable (oder `.env`-Datei) | Tägliche lokale Shell-Nutzung |
| Aktives Konto des Account-Stores | Nach `zai-client accounts use <name>` |

Die `.env`-Datei ist der Normalfall — kopieren Sie die kommentierte Vorlage und
bearbeiten Sie sie:

```bash
cp .env.example .env
# oder auf eine beliebige Datei zeigen: zai-client --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # Chat-Endpunkt überschreiben
# ZAI_REGION=china                                   # falls Ihr Key auf open.bigmodel.cn ausgestellt wurde
# ZAI_CHINA_API_KEY=...                              # separate bigmodel.cn-Anmeldedaten
# ZAI_ENV=production
```

→ Vollständige Referenz (Mehrfachkonten, regionale Gateways, Kontingent-Zeiträume):
**[Konten & Kontingent](docs/en/accounts-and-quota.md)**

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

## Verhältnis zu den offiziellen SDKs

Z.AI / Zhipu bieten offizielle SDKs für **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
und **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4)).
Es gibt **kein offizielles Go-SDK** — `go-z-ai` schließt diese Lücke und schichtet
eine CLI, eine TUI, das Umschalten regionaler Gateways
(`api.z.ai` ↔ `open.bigmodel.cn`) und die Mehrfachkonten-Verwaltung für den
GLM Coding Plan über derselben API-Oberfläche auf.

> ℹ️ `zai-claude-config.json` im Repo-Root ist eine **Vorlage** mit
> Platzhaltern (`"your-zai-api-key-here"`), die von
> `zai-client coding load claude-code` verwendet wird. Es handelt sich nicht um
> eine echte Konfiguration, und es werden keine Anmeldedaten mitgeliefert.

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
