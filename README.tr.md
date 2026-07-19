# go-z-ai

Z.AI (Zhipu AI / BigModel) platformu için bir Go **CLI**'sı, **kitaplığı** ve
**TUI**'sı — tüm GLM model yüzeylerini tek bir araçta, ayrıca Claude Code,
OpenCode, Crush, Factory Droid ve Cursor'ı GLM Coding Plan'ınıza bağlayan
`@z_ai/coding-helper`'ın bir Go port'u.

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/viewer/?uri=github.com/SamyRai/go-z-ai)
[![Latest release](https://img.shields.io/github/v/release/SamyRai/go-z-ai)](https://github.com/SamyRai/go-z-ai/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Hızlı örnek

```bash
# 1. Yapılandır (bunlardan herhangi biri çalışır — ortam değişkeni, .env dosyası veya --config <dosya>)
export ZAI_API_KEY=your_api_key_here
# veya: cp .env.example .env, sonra .env dosyasını düzenle

# 2. CLI'yı kullan
go-z-ai chat create "Goroutine'leri tek bir paragrafta açıkla" --stream
```

```go
// …veya kitaplığı içe aktarın — CLI gerekmez.
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Goroutine'leri tek bir paragrafta açıkla"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Daha fazla çalıştırılabilir program — akış (streaming), asenkron görsel sorgulama,
Anthropic `/v1/messages` uç noktası — [`examples/`](examples/) dizininde.

## Özellikler

- **Sohbet** — akış (streaming), yapılandırılmış çıktı (JSON Schema), derin
  düşünme, fonksiyon/araç çağrısı, görsel giriş (`glm-4.6v`/`glm-4.5v`) ve bir
  **Anthropic uyumlu `/v1/messages`** uç noktası (GLM Coding Plan'a
  bağlandığında Claude Code ve Cursor'ın eriştiği aynı uç nokta).
- **Medya** — görsel üretimi, video üretimi (her zaman asenkron), ses
  transkripsiyonu, TTS ve GLM-TTS ses klonlama.
- **Belge anlama** — düzen OCR'ı, el yazısı OCR'ı ve RAG ön işleme için bir
  belge ayrıştırıcı.
- **Erişim (Retrieval)** — gömme (embedding), yeniden sıralama, yerleşik web
  arama / web okuyucu / tokenizer araçları.
- **Moderasyon** — Çin platformu uç noktası üzerinden içerik moderasyonu.
- **Ajanlar** — Z.AI'nin uzmanlaşmış ajanları (çeviri, slayt/poster üretimi,
  video efektleri).
- **Toplu işler & dosyalar** — sohbet tamamlamaları için JSONL toplu işler,
  dosya yükleme/listeleme/indirme.
- **GLM Coding Plan** — kota/kullanım izleme, çoklu hesap yönetimi ve
  aboneliğinize Claude Code, OpenCode, Crush, Factory Droid ve Cursor'ı bağlamak
  için `go-z-ai coding`.
- **DX** — tam ekran terminal UI'ı (`go-z-ai tui`), bölgesel ağ geçidi
  değiştirme (`api.z.ai` ↔ `open.bigmodel.cn`), backoff + jitter ile otomatik
  yeniden deneme ve her Z.AI hata kodunun eşlendiği tipli bir `APIError`.

## Kurulum

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Bu, `$GOPATH/bin` altında `go-z-ai` adlı bir ikili (binary) oluşturur.

```bash
# İsteğe bağlı kısa alias: ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

Go 1.26.4+ ve bir [Z.AI API anahtarı](https://z.ai/manage-apikey/apikey-list) gerektirir.
Kaynaktan derleme, ilk çalıştırmada kimlik doğrulama ve sorun giderme:
**[Başlarken →](docs/en/getting-started.md)**

## CLI olarak

Tüm yüzeyi kapsayan tek bir `go-z-ai` ikilisi. Her komut `--help`
destekler; hızlı tur:

```bash
go-z-ai chat create "..." --stream          # sohbet (akış, araçlar, görsel giriş, yapılandırılmış çıktı)
go-z-ai anthropic messages "..." --stream   # Anthropic uyumlu /v1/messages
go-z-ai image|video|audio|voice ...         # medya üretimi, transkripsiyon, TTS, klonlama
go-z-ai ocr|parser ...                      # OCR + belge ayrıştırma
go-z-ai embeddings|rerank|moderations ...   # erişim + içerik moderasyonu
go-z-ai models list                         # model kataloğu + fiyatlandırma
go-z-ai accounts add|use|quota|usage ...    # çoklu hesap + GLM Coding Plan izleme
go-z-ai coding auth|load|doctor|mcp ...     # Claude Code / Cursor / vb. öğeleri GLM Coding Plan'a bağla
go-z-ai tui                                 # tam ekran terminal UI'ı (yukarıdakilerin tamamı)
go-z-ai validate                            # anahtarınızın çalıştığını tek bir gerçek çağrıyla doğrula
```

Sonuç üreten her komut `--format text|json` alır (JSON stdout'a, ilerleme
konuşmaları stderr'e gider, böylece `jq`'ya yönlendirebilirsiniz).

→ Tam komut listesi: **[CLI Referansı](docs/en/cli-reference.md)**

## Go kitaplığı olarak

`pkg/client` tek genel olarak içe aktarılabilir pakettir; `internal/` altındaki
her şey uygulama detayıdır. Yeniden deneme, zaman aşımı, bölgesel ağ geçidi
seçimi ve hata eşleme merkezîdir — servisler kendi `http.Client`'larını
oluşturmaz veya ham istekler göndermez.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // İsteğe bağlı: BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey, Region
})
```

Servisler, tümü `c.<Service>().<Method>(ctx, …)` desenini izler:

| Erişim | Kapsar |
|---|---|
| `c.Chat()` | Tamamlamalar, akış, asenkron, `RunWithTools` |
| `c.Anthropic()` | Anthropic protokollü `/v1/messages` (Create, CreateStream) |
| `c.Models()` | List, Get, metin/görsel/ücretsiz filtreler |
| `c.Images()` / `c.Videos()` | Görsel (senkron/asenkron), video (her zaman asenkron) |
| `c.Audio()` / `c.Voice()` | Transkripsiyon, TTS, ses klonlama |
| `c.Layout()` / `c.FileParser()` | RAG için OCR + belgeden-metine |
| `c.Files()` / `c.Batch()` | Yükleme, toplu işler |
| `c.Agents()` | Z.AI uzmanlaşmış ajanları |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Erişim + moderasyon |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Usage()` / `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan izleme |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Asenkron görevler için paylaşılan sorgulama |

→ Örneklerle tam API: **[Kitaplık Kılavuzu](docs/en/library-guide.md)**
→ Oluşturulmuş referans: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Yapılandırma

Kimlik bilgilerini sağlamanın üç yolu, şu öncelik sırasıyla çözümlenir (en
yüksek olan kazanır):

| Yöntem | Ne zaman kullanılır |
|---|---|
| `--api-key <key>` bayrağı | Tek seferlik çağrılar, betikler, CI |
| `--account <name>` bayrağı | [Kayıtlı hesaplar](docs/en/accounts-and-quota.md) arasında geçiş |
| `ZAI_API_KEY` ortam değişkeni (veya `.env` dosyası) | Günlük yerel kabuk kullanımı |
| Hesap deposunun aktif hesabı | `go-z-ai accounts use <name>`'dan sonra |

`.env` dosyası yaygın olanıdır — açıklamalı şablonu kopyalayın ve düzenleyin:

```bash
cp .env.example .env
# veya herhangi bir dosyayı gösterin: go-z-ai --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # sohbet uç noktasını geçersiz kıl
# ZAI_REGION=china                                   # anahtarınız open.bigmodel.cn üzerinde yayımlandıysa
# ZAI_CHINA_API_KEY=...                              # ayrı bigmodel.cn kimlik bilgisi
# ZAI_ENV=production
```

→ Tam referans (çoklu hesap, bölgesel ağ geçitleri, kota pencereleri):
**[Hesaplar & Kotalar](docs/en/accounts-and-quota.md)**

## Belgeler

**[Tam belge dizini →](docs/en/README.md)**

| | |
|---|---|
| [Başlarken](docs/en/getting-started.md) | [CLI Referansı](docs/en/cli-reference.md) |
| [Hesaplar & Kotalar](docs/en/accounts-and-quota.md) | [Kodlama Araçları](docs/en/coding-tools.md) |
| [Kitaplık Kılavuzu](docs/en/library-guide.md) | [Hata Yönetimi](docs/en/error-handling.md) |
| [Mimari](docs/en/architecture.md) | [Yol Haritası & Sınırlamalar](docs/en/roadmap.md) |
| [Katkıda Bulunma](CONTRIBUTING.md) | [Güvenlik Politikası](SECURITY.md) |
| [Davranış Kuralları](CODE_OF_CONDUCT.md) | [Değişiklik Günlüğü](CHANGELOG.md) |

## Resmî SDK'lar ile ilişkisi

Z.AI / Zhipu, **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
ve **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4))
için resmî SDK'lar yayımlar. Resmî bir Go SDK'sı **yoktur** — `go-z-ai` bu
boşluğu doldurur ve aynı API yüzeyi üzerine bir CLI, bir TUI, bölgesel ağ
geçidi değiştirme (`api.z.ai` ↔ `open.bigmodel.cn`) ve GLM Coding Plan çoklu
hesap yönetimi ekler.

> ℹ️ Repo kökündeki `zai-claude-config.json`, `go-z-ai coding load
> claude-code` tarafından kullanılan, yer tutucu değerler içeren
> (`"your-zai-api-key-here"`) bir **şablondur**. Gerçek bir yapılandırma
> değildir ve hiçbir kimlik bilgisi içermez.

## Katkıda Bulunma

[CONTRIBUTING.md](CONTRIBUTING.md)'ye bakın — özellikle bir servis ekliyor veya
değiştiriyorsanız projenin canlı doğrulama kuralına (el ile yazılmış
fixture'lar yerine kaydedilmiş API kasetleri) dikkat edin.

## Lisans

Apache License 2.0 — bkz. [LICENSE](LICENSE).

## Destek

- **Z.AI API belgeleri**: [https://docs.z.ai](https://docs.z.ai)
- **Sorunlar**: [GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **Güvenlik**: bkz. [SECURITY.md](SECURITY.md) — lütfen güvenlik açıklarını
  herkese açık issue olarak açmayın.
