# go-z-ai

Eine Go-**CLI**, **Bibliothek** und ein **TUI** für die Z.AI-Plattform
(Zhipu AI / BigModel) — jede Modell-Schnittstelle von GLM in einem einzigen
Werkzeug, plus ein Go-Port von `@z_ai/coding-helper`, der Claude Code, Codex,
OpenCode, Crush und Factory Droid an Ihren GLM Coding Plan anbindet.

[English](README.md) | [简体中文](README.zh.md) | [Русский](README.ru.md) | **Deutsch** | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

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
go-z-ai chat create "Erkläre Goroutinen in einem Absatz" --stream
```

```go
// …oder die Bibliothek importieren — keine CLI erforderlich.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv() // liest ZAI_API_KEY
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

Weitere lauffähige Programme — Tool-Calling, strukturierte Ausgabe, Vision,
asynchrones Polling von Bildern, der Anthropic-Endpunkt `/v1/messages` —
finden Sie unter [`examples/`](examples/).

## Features

- **Chat** — Streaming (in der Bibliothek ein Go-Iterator), Steuerung des
  Reasonings (`reasoning_effort`, Thinking ein/aus), Function/Tool-Aufrufe
  (plus die integrierten Tool-Typen `web_search`, `retrieval` und `mcp`),
  JSON-Objekt-Ausgabe mit dem Schema im Prompt, multimodale Eingabe (Bild,
  Video, Datei) und ein **Anthropic-kompatibler `/v1/messages`-Endpunkt**
  (derselbe, den Claude Code anspricht, wenn es an einen GLM Coding Plan
  angebunden ist).
- **Modelle** — ein kuratierter Katalog (Kontextfenster, maximale Ausgabe,
  Preise, Fähigkeiten, akzeptierte Reasoning-Stufen) hinter `go-z-ai models`
  und `client.CatalogEntry`; die Standardmodelle sind
  [unten](#standardmodelle) aufgelistet.
- **Medien** — Bildgenerierung, Videogenerierung (immer asynchron, mit
  optionaler Einreihung außerhalb der Spitzenzeiten), Audiotranskription, TTS
  und GLM-TTS-Stimmklonung.
- **Dokumentenverständnis** — Layout-OCR, Handschrift-OCR und ein
  Dokumentenparser für die RAG-Vorverarbeitung.
- **Retrieval** — Embeddings, Reranking, integrierte Web-Such- /
  Web-Reader- / Tokenizer-Tools.
- **Moderationen** — Inhaltsmoderation über den Endpunkt der China-Plattform.
  Embeddings und Moderationen werden von `open.bigmodel.cn` bereitgestellt und
  können von der Berechtigung des Kontos abhängen; siehe
  [Roadmap & Einschränkungen](docs/en/roadmap.md).
- **Agenten** — die spezialisierten Agents von Z.AI (Übersetzung,
  Folien-/Postergenerierung, Videoeffekte).
- **Batch & Dateien** — JSONL-Batch-Jobs für Chat-Completions,
  Datei-Upload/-Auflistung/-Download/-Löschung.
- **GLM Coding Plan** — Erkennung des Kontotyps, creditbasierte Überwachung von
  Kontingent/Nutzung, Mehrfachkonten-Verwaltung und `go-z-ai coding`, um
  Claude Code, Codex, OpenCode, Crush und Factory Droid an Ihr Abonnement
  anzubinden und die vier offiziellen MCP-Server von Z.AI zu registrieren
  (Vision, Websuche, Web-Reader, Zread).
- **DX** — Terminal-Vollbild-UI (`go-z-ai tui`: Tabs für Chat, Modelle, Nutzung,
  Konten, Coding, Medien und Tools), ein einziger `--region`-Schalter
  (`api.z.ai` ↔ `open.bigmodel.cn`), der jeden Endpunkt auswählt, automatische
  Wiederholung mit Backoff + Jitter (unter Beachtung von `Retry-After`) sowie
  ein typisiertes `APIError`, in dem Z.AI-Fehlercodes abgebildet sind
  (unbekannte Codes fallen auf den HTTP-Status zurück).

### Standardmodelle

Aufrufer wählen Standardwerte aus den Konstanten in `pkg/client`, statt IDs
fest einzucodieren:

| Konstante | Modell | Verwendet für |
|---|---|---|
| `client.DefaultModel` | `glm-5.3` | Chat, Anthropic-Endpunkt, Tokenizer. 1M Kontext, Reasoning-Stufen `low`/`high`/`max` |
| `client.DefaultFastModel`, `client.DefaultVisionModel` | `glm-5.3-flash` | Schnelle, kostengünstige Stufe; nativ multimodal (Bild-, Video-, Dateieingabe) |
| `client.DefaultOCRModel` | `glm-ocr` | `ocr`, `Layout().Parse` |
| `client.DefaultASRModel` | `glm-asr-2512` | `audio transcribe` (Clips bis zu 30 s) |
| `client.DefaultTTSModel` | `glm-tts` | `audio speech` |
| `client.ModelGLMImage`, `client.ModelCogView4` | `glm-image` (Standard), `cogview-4-250304` | `image generate` |
| `client.VideoModels` | `cogvideox-3` (Standard), `viduq1-*`, `vidu2-*` | `video generate` |

Der Katalog umfasst außerdem `glm-5.3-flashx`, `glm-5.2`, die GLM-4.x-Familie
und die kostenlosen Stufen (`go-z-ai models free`). Er ist eine kuratierte
Momentaufnahme (zuletzt aktualisiert am 2026-10-02) — `go-z-ai models list`
zeigt Kontext, maximale Ausgabe, Preise, Fähigkeiten und Reasoning-Stufen, und
Live-Werte von `/models` haben Vorrang vor den Katalogwerten.

## Installation

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Dies erzeugt einen Binary namens `go-z-ai` in Ihrem `$GOPATH/bin`.

```bash
# Optionaler Kurz-Alias: ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Setzt Go 1.26.4+ und einen [Z.AI API-Key](https://z.ai/manage-apikey/apikey-list) voraus.
Build aus dem Quellcode, Erstanmeldung und Fehlerbehebung:
**[Erste Schritte →](docs/en/getting-started.md)**

## Als CLI

Ein einzelner `go-z-ai`-Binary deckt die gesamte Oberfläche ab. Jeder Befehl
unterstützt `--help`; hier der Schnelldurchlauf:

```bash
go-z-ai chat create "..." --stream          # Chat (Streaming, Reasoning-Stufe, Tools, Bild-/Video-/Dateieingabe, JSON-Ausgabe)
go-z-ai anthropic messages "..." --stream   # Anthropic-kompatibel /v1/messages
go-z-ai responses create "..." --stream     # OpenAI-Responses-Protokoll (/api/v1, wird von Codex verwendet)
go-z-ai image|video|audio|voice ...         # Medien-Generierung, Transkription, TTS, Klonen
go-z-ai ocr|parser ...                      # OCR + Dokumenten-Parsing
go-z-ai embeddings|rerank|moderations ...   # Retrieval + Inhaltsmoderation
go-z-ai models list                         # Modellkatalog: Kontext, Preise, Fähigkeiten, Reasoning-Stufen
go-z-ai account detect|status               # Key-Typ (Coding Plan / Pay-as-you-go), Region, Zustand
go-z-ai accounts add|use|quota|usage ...    # Mehrfachkonten + GLM-Coding-Plan-Überwachung
go-z-ai usage quota                         # Kontingentfenster des GLM Coding Plan für den aktuellen Key
go-z-ai coding auth|load|doctor|mcp ...     # Claude Code / Codex / OpenCode / Crush / Factory Droid an GLM Coding Plan anbinden
go-z-ai tui                                 # Terminal-Vollbild-UI (alles oben Genannte)
go-z-ai validate                            # prüfen, ob Ihr Key funktioniert (eine kostenlose Anfrage)
```

Die meisten Befehle, die Ergebnisse ausgeben, akzeptieren `--format text|json`
(JSON geht an stdout, Fortschrittsmeldungen an stderr, sodass Sie in `jq`
weiterleiten können). Das Root-Flag `--region global|china` (Umgebungsvariable
`ZAI_REGION`) wählt jeden Endpunkt aus; `--base-url` überschreibt nur die
Chat-/PaaS-Wurzel.

### Coding-Tools

`go-z-ai coding` ist ein Go-Port von Z.AIs `@z_ai/coding-helper`. Unterstützte
Tools: `claude-code`, `codex`, `opencode`, `crush` und `factory-droid`.

```bash
go-z-ai coding auth glm_coding_plan_global <key>   # Plan-Key prüfen + speichern (oder glm_coding_plan_china)
go-z-ai coding load claude-code                    # Konfiguration des Tools schreiben (auch: codex, opencode, crush, factory-droid)
go-z-ai coding mcp add claude-code                 # die offiziellen MCP-Server registrieren (--server zur Auswahl)
go-z-ai coding doctor                              # Zustandsprüfung; beendet sich bei Problemen mit einem Wert ungleich null
```

- Bei Claude Code wird die Haiku-Stufe auf `client.DefaultFastModel` und die
  Sonnet-/Opus-Stufen werden auf `client.DefaultModel` abgebildet, mit dem
  `[1m]`-Suffix von Claude Code für 1M Kontext. `--haiku`, `--sonnet`, `--opus`,
  `--no-model-mapping`, `--auto-compact-window`, `--max-thinking-tokens` und
  `--max-output-tokens` passen dies bei `coding load` und `coding auth` an.
- `coding mcp add|remove <tool>` registriert die vier offiziellen MCP-Server
  von Z.AI: `zai-mcp-server` (Vision; läuft lokal über `npx`, benötigt
  Node.js), `web-search-prime`, `web-reader` und `zread` (gehostet,
  authentifiziert mit dem Plan-Key). Standard ist jeder Server, den das Tool
  unterstützt — Codex nimmt nur Vision (openai/codex#14793); `coding mcp status`
  zeigt, was jedes Tool hat. MCP-Aufrufe verbrauchen Plan-Kontingent.
- Für Codex schreibt `coding load codex` einen ZAI-Provider, der das
  Responses-Protokoll (`/api/v1`) spricht, nach `~/.codex/config.toml` und die
  Metadaten des Modells nach `~/.codex/models.json`.

→ Vollständige Befehlsliste: **[CLI-Referenz](docs/en/cli-reference.md)**

## Als Go-Bibliothek

`pkg/client` ist die Bibliothek (nur Standardbibliothek); `pkg/observe` ist ein
optionaler OpenTelemetry-Hook-Adapter. Alles unter `internal/` ist
Implementierungsdetail. Retry, Timeout, Auswahl des regionalen Gateways und
Fehler-Mapping sind zentralisiert — Services bauen niemals ihren eigenen
`http.Client` und senden keine rohen Requests.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

// Aus der Umgebung: ZAI_API_KEY, ZAI_API_BASE_URL, ZAI_REGION,
// ZAI_CHINA_API_KEY, ZAI_MONITOR_TIMEZONE.
c, err := client.NewClientFromEnv()

// Oder explizit:
c, err = client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // Optional: Region, BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey,
    // UserAgent, Hooks
})
```

Services, alle nach dem Muster `c.<Service>().<Method>(ctx, …)`:

| Accessor | Deckt ab |
|---|---|
| `c.Chat()` | `Create`, `Stream` (Iterator), `CreateAsync`, `RunWithTools` |
| `c.Anthropic()` | Anthropic-Protokoll `/v1/messages` (`Create`, `Stream`) |
| `c.Responses()` | OpenAI-Responses-Protokoll unter `/api/v1` — die Codex-Schnittstelle (`Create`, `Stream`) |
| `c.Models()` | List, Get, Filter nach Text/Vision/kostenlos |
| `c.Images()` / `c.Videos()` | Bilder (`Generate`, `GenerateAsync`), Video (immer async) |
| `c.Audio()` / `c.Voice()` | Transkription, TTS, Stimmklonung |
| `c.Layout()` / `c.FileParser()` | OCR + Dokument-zu-Text für RAG |
| `c.Files()` / `c.Batch()` | Upload, Batch-Jobs |
| `c.Agents()` | spezialisierte Agents von Z.AI |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Retrieval + Moderation |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Quota()` / `c.Account()` / `c.Detection()` | Kontingent und Nutzung des GLM Coding Plan, Kontoinformationen, Erkennung des Kontotyps |
| `c.GetAsyncResult()` / `c.WaitForResult()` | gemeinsames Polling für asynchrone Aufgaben |

Gut zu wissen:

- Standardwerte stammen aus dem Katalog: `client.DefaultModel`,
  `client.DefaultFastModel`, `client.DefaultVisionModel` sowie die
  OCR-/ASR-/TTS-Konstanten. `ChatRequest.ReasoningEffort` wird gegen die vom
  Modell akzeptierten Stufen validiert (`client.EffortLow`, `EffortHigh`,
  `EffortMax` für GLM-5.3); `client.CatalogEntry(model)` und `Pricing.Cost(usage)`
  liefern Kontextgrößen und Preise.
- `Chat().Stream`, `Anthropic().Stream` und `Responses().Stream` liefern
  `iter.Seq2`-Iteratoren. Das Verlassen der Schleife schließt den Stream, und
  ein In-Band-Fehler-Chunk beendet ihn mit einem `*APIError`.
- Z.AI kennt kein Antwortformat `json_schema`. Für strukturierte Ausgabe setzen
  Sie `ResponseFormat: client.JSONObjectFormat()` und legen
  `client.JSONSchemaPrompt(schema)` in den System-Prompt.
- `Config.Region` (`client.RegionGlobal` oder `client.RegionChina`) bestimmt
  jede URL; `Config.BaseURL` ist standardmäßig `Region.PaaSBaseURL()`.

→ Vollständige API mit Beispielen: **[Bibliotheksanleitung](docs/en/library-guide.md)**
→ Generierte Referenz: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Konfiguration

Anmeldedaten werden in dieser Prioritätsreihenfolge aufgelöst
(höchste gewinnt):

| Methode | Wann verwenden |
|---|---|
| `--api-key <key>` Flag | Einmalige Aufrufe, Skripte, CI |
| `--account <name>` Flag | Wechseln zwischen [gespeicherten Konten](docs/en/accounts-and-quota.md) |
| `ZAI_API_KEY` Umgebungsvariable (oder `.env`-Datei) | Tägliche lokale Shell-Nutzung |
| Aktives Konto des Account-Stores | Nach `go-z-ai accounts use <name>` |

Die `.env`-Datei ist der Normalfall — kopieren Sie die kommentierte Vorlage und
bearbeiten Sie sie:

```bash
cp .env.example .env
# oder auf eine beliebige Datei zeigen: go-z-ai --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # Chat-Endpunkt überschreiben
# ZAI_REGION=china                                   # falls Ihr Key auf open.bigmodel.cn ausgestellt wurde; wählt jeden Endpunkt aus
# ZAI_CHINA_API_KEY=...                              # separate bigmodel.cn-Anmeldedaten (Embeddings/Moderationen)
# ZAI_MONITOR_TIMEZONE=UTC                           # Zeitzone der Kontingent-/Nutzungs-API (Standard UTC+8)
```

Das `client.NewClientFromEnv()` der Bibliothek liest dieselben Variablen aus der
Prozessumgebung (es lädt keine `.env`-Datei).

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
([PyPI `zai-sdk`](https://pypi.org/project/zai-sdk/)), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
und **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4)).
Es gibt **kein offizielles Go-SDK** — `go-z-ai` schließt diese Lücke und schichtet
eine CLI, eine TUI, das Umschalten regionaler Gateways
(`api.z.ai` ↔ `open.bigmodel.cn`) und die Mehrfachkonten-Verwaltung für den
GLM Coding Plan über derselben API-Oberfläche auf.

> ℹ️ `zai-claude-config.json` im Repo-Root ist eine **Vorlage** mit
> Platzhaltern (`"your-zai-api-key-here"`), die veranschaulicht, welche
> Einstellungen `go-z-ai coding load claude-code` schreibt. Das Programm liest
> sie nicht, es handelt sich nicht um eine echte Konfiguration, und es werden
> keine Anmeldedaten mitgeliefert; führen Sie `go-z-ai coding load claude-code`
> aus, um die aktuelle Modellzuordnung zu erhalten.
>
> ⚠️ **Nutzungsrichtlinie.** Der Coding-Endpunkt von Z.AI ist auf
> „offiziell unterstützte Tools" beschränkt und untersagt SDK-basierten
> Zugriff; siehe
> [Coding Tools — Compliance](docs/en/coding-tools.md#compliance--usage-policy-).
> `go-z-ai` sendet bei jeder Anfrage einen identifizierenden
> `User-Agent`-Header, und sein `coding`-Unterbefehl verbindet offiziell
> unterstützte Tools. Die direkte Nutzung von `pkg/client` gegen den
> Coding-Endpunkt aus einer eigenen Integration erfolgt auf eigenes Risiko,
> bis eine ausdrückliche Aufnahme erfolgt.

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
