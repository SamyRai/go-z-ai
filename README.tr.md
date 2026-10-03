# go-z-ai

Z.AI (Zhipu AI / BigModel) platformu için bir Go **CLI**'sı, **kitaplığı** ve
**TUI**'sı — tüm GLM model yüzeylerini tek bir araçta, ayrıca Claude Code, Codex,
OpenCode, Crush ve Factory Droid'i GLM Coding Plan'ınıza bağlayan
`@z_ai/coding-helper`'ın bir Go port'u.

[English](README.md) | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | **Türkçe**

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
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv() // ZAI_API_KEY'i okur
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

Daha fazla çalıştırılabilir program — araç çağırma, yapılandırılmış çıktı,
görsel girdi, asenkron görsel sorgulama, Anthropic `/v1/messages` uç noktası —
[`examples/`](examples/) dizininde.

## Özellikler

- **Sohbet** — akış (kitaplıkta bir Go yineleyicisi), akıl yürütme kontrolü
  (`reasoning_effort`, thinking açık/kapalı), fonksiyon/araç çağrısı (ayrıca
  yerleşik `web_search`, `retrieval` ve `mcp` araç türleri), şemanın istemde
  verildiği JSON nesnesi çıktısı, çok modlu girdi (görsel, video, dosya) ve bir
  **Anthropic uyumlu `/v1/messages`** uç noktası (GLM Coding Plan'a
  bağlandığında Claude Code'un eriştiği aynı uç nokta).
- **Modeller** — `go-z-ai models` ve `client.CatalogEntry` arkasındaki derlenmiş
  bir katalog (bağlam penceresi, azami çıktı, fiyatlar, yetenekler, kabul edilen
  akıl yürütme düzeyleri); varsayılanlar [aşağıda](#varsayılan-modeller)
  listelenmiştir.
- **Medya** — görsel üretimi, video üretimi (her zaman asenkron, isteğe bağlı
  yoğun olmayan saatlerde kuyruklama ile), ses transkripsiyonu, TTS ve GLM-TTS
  ses klonlama.
- **Belge anlama** — düzen OCR'ı, el yazısı OCR'ı ve RAG ön işleme için bir
  belge ayrıştırıcı.
- **Erişim (Retrieval)** — gömme (embedding), yeniden sıralama, yerleşik web
  arama / web okuyucu / tokenizer araçları.
- **Moderasyon** — Çin platformu uç noktası üzerinden içerik moderasyonu.
  Gömme ve moderasyon `open.bigmodel.cn` tarafından sunulur ve hesap
  yetkilendirmesine bağlı olarak kısıtlanabilir; bkz.
  [Yol Haritası & Sınırlamalar](docs/en/roadmap.md).
- **Ajanlar** — Z.AI'nin uzmanlaşmış ajanları (çeviri, slayt/poster üretimi,
  video efektleri).
- **Toplu işler & dosyalar** — sohbet tamamlamaları için JSONL toplu işler,
  dosya yükleme/listeleme/indirme/silme.
- **GLM Coding Plan** — hesap türü tespiti, kredi tabanlı kota/kullanım izleme,
  çoklu hesap yönetimi ve aboneliğinize Claude Code, Codex, OpenCode, Crush ve
  Factory Droid'i bağlamak ve Z.AI'nin dört resmî MCP sunucusunu (Vision, web
  arama, web okuyucu, Zread) kaydetmek için `go-z-ai coding`.
- **DX** — tam ekran terminal UI'ı (`go-z-ai tui`: sohbet, modeller, kullanım,
  hesaplar, coding, medya ve araçlar sekmeleri), her uç noktayı seçen tek bir
  `--region` anahtarı (`api.z.ai` ↔ `open.bigmodel.cn`), backoff + jitter ile
  otomatik yeniden deneme (`Retry-After` dikkate alınır) ve Z.AI hata
  kodlarının eşlendiği tipli bir `APIError` (bilinmeyen kodlar HTTP durumuna
  düşer).

### Varsayılan modeller

Çağıranlar, kimlikleri koda gömmek yerine varsayılanları `pkg/client`
sabitlerinden seçer:

| Sabit | Model | Kullanım alanı |
|---|---|---|
| `client.DefaultModel` | `glm-5.3` | Sohbet, Anthropic uç noktası, tokenizer. 1M bağlam, akıl yürütme düzeyleri `low`/`high`/`max` |
| `client.DefaultFastModel`, `client.DefaultVisionModel` | `glm-5.3-flash` | Hızlı, düşük maliyetli katman; doğal olarak çok modlu (görsel, video, dosya girdisi) |
| `client.DefaultOCRModel` | `glm-ocr` | `ocr`, `Layout().Parse` |
| `client.DefaultASRModel` | `glm-asr-2512` | `audio transcribe` (30 sn'ye kadar klipler) |
| `client.DefaultTTSModel` | `glm-tts` | `audio speech` |
| `client.ModelGLMImage`, `client.ModelCogView4` | `glm-image` (varsayılan), `cogview-4-250304` | `image generate` |
| `client.VideoModels` | `cogvideox-3` (varsayılan), `viduq1-*`, `vidu2-*` | `video generate` |

Katalog ayrıca `glm-5.3-flashx`, `glm-5.2`, GLM-4.x ailesini ve ücretsiz
katmanları (`go-z-ai models free`) kapsar. Derlenmiş bir anlık görüntüdür (son
güncelleme 2026-10-02) — `go-z-ai models list` bağlamı, azami çıktıyı,
fiyatları, yetenekleri ve akıl yürütme düzeylerini gösterir; canlı `/models`
değerleri katalog değerlerinden önceliklidir.

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
go-z-ai chat create "..." --stream          # sohbet (akış, akıl yürütme düzeyi, araçlar, görsel/video/dosya girdisi, JSON çıktısı)
go-z-ai anthropic messages "..." --stream   # Anthropic uyumlu /v1/messages
go-z-ai responses create "..." --stream     # OpenAI Responses protokolü (/api/v1, Codex'in kullandığı)
go-z-ai image|video|audio|voice ...         # medya üretimi, transkripsiyon, TTS, klonlama
go-z-ai ocr|parser ...                      # OCR + belge ayrıştırma
go-z-ai embeddings|rerank|moderations ...   # erişim + içerik moderasyonu
go-z-ai models list                         # model kataloğu: bağlam, fiyatlar, yetenekler, akıl yürütme düzeyleri
go-z-ai account detect|status               # anahtar türü (coding plan / kullandıkça öde), bölge, sağlık durumu
go-z-ai accounts add|use|quota|usage ...    # çoklu hesap + GLM Coding Plan izleme
go-z-ai usage quota                         # geçerli anahtar için GLM Coding Plan kota pencereleri
go-z-ai coding auth|load|doctor|mcp ...     # Claude Code / Codex / OpenCode / Crush / Factory Droid'i GLM Coding Plan'a bağla
go-z-ai tui                                 # tam ekran terminal UI'ı (yukarıdakilerin tamamı)
go-z-ai validate                            # anahtarınızın çalıştığını doğrula (ücretsiz bir istek)
```

Sonuç yazdıran komutların çoğu `--format text|json` alır (JSON stdout'a,
ilerleme konuşmaları stderr'e gider, böylece `jq`'ya yönlendirebilirsiniz).
Kök `--region global|china` bayrağı (ortam değişkeni `ZAI_REGION`) her uç
noktayı seçer; `--base-url` yalnızca sohbet/PaaS kökünü geçersiz kılar.

### Kodlama araçları

`go-z-ai coding`, Z.AI'nin `@z_ai/coding-helper` aracının bir Go port'udur.
Desteklenen araçlar: `claude-code`, `codex`, `opencode`, `crush` ve
`factory-droid`.

```bash
go-z-ai coding auth glm_coding_plan_global <key>   # plan anahtarını doğrula + sakla (veya glm_coding_plan_china)
go-z-ai coding load claude-code                    # aracın yapılandırmasını yaz (ayrıca: codex, opencode, crush, factory-droid)
go-z-ai coding mcp add claude-code                 # resmî MCP sunucularını kaydet (seçmek için --server)
go-z-ai coding doctor                              # sağlık kontrolü; sorun varsa sıfırdan farklı kodla çıkar
```

- Claude Code için haiku katmanı `client.DefaultFastModel`'e, sonnet/opus
  katmanları ise `client.DefaultModel`'e eşlenir; Claude Code'un `[1m]`
  1M bağlam soneki kullanılır. `--haiku`, `--sonnet`, `--opus`,
  `--no-model-mapping`, `--auto-compact-window`, `--max-thinking-tokens` ve
  `--max-output-tokens` bunu `coding load` ve `coding auth` üzerinde ayarlar.
- `coding mcp add|remove <tool>`, Z.AI'nin dört resmî MCP sunucusunu kaydeder:
  `zai-mcp-server` (Vision; `npx` ile yerelde çalışır, Node.js gerekir),
  `web-search-prime`, `web-reader` ve `zread` (barındırılan, plan anahtarıyla
  kimlik doğrulaması yapar). Varsayılan, aracın desteklediği her sunucudur —
  Codex yalnızca Vision'ı alır (openai/codex#14793); `coding mcp status` her
  aracın neye sahip olduğunu gösterir. MCP çağrıları plan kotasından düşer.
- Codex için `coding load codex`, Responses protokolünü (`/api/v1`) konuşan bir
  ZAI sağlayıcısını `~/.codex/config.toml` dosyasına, modelin meta verilerini
  ise `~/.codex/models.json` dosyasına yazar.

→ Tam komut listesi: **[CLI Referansı](docs/en/cli-reference.md)**

## Go kitaplığı olarak

`pkg/client` kitaplığın kendisidir (yalnızca standart kütüphane); `pkg/observe`
isteğe bağlı bir OpenTelemetry hook adaptörüdür. `internal/` altındaki her şey
uygulama detayıdır. Yeniden deneme, zaman aşımı, bölgesel ağ geçidi seçimi ve
hata eşleme merkezîdir — servisler kendi `http.Client`'larını oluşturmaz veya
ham istekler göndermez.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

// Ortamdan: ZAI_API_KEY, ZAI_API_BASE_URL, ZAI_REGION,
// ZAI_CHINA_API_KEY, ZAI_MONITOR_TIMEZONE.
c, err := client.NewClientFromEnv()

// Veya açıkça:
c, err = client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // İsteğe bağlı: Region, BaseURL, Timeout, MaxRetries, RetryDelay, ChinaAPIKey,
    // UserAgent, Hooks
})
```

Servisler, tümü `c.<Service>().<Method>(ctx, …)` desenini izler:

| Erişim | Kapsar |
|---|---|
| `c.Chat()` | `Create`, `Stream` (yineleyici), `CreateAsync`, `RunWithTools` |
| `c.Anthropic()` | Anthropic protokollü `/v1/messages` (`Create`, `Stream`) |
| `c.Responses()` | `/api/v1` üzerinde OpenAI Responses protokolü — Codex yüzeyi (`Create`, `Stream`) |
| `c.Models()` | List, Get, metin/görsel/ücretsiz filtreler |
| `c.Images()` / `c.Videos()` | Görsel (`Generate`, `GenerateAsync`), video (her zaman asenkron) |
| `c.Audio()` / `c.Voice()` | Transkripsiyon, TTS, ses klonlama |
| `c.Layout()` / `c.FileParser()` | RAG için OCR + belgeden-metine |
| `c.Files()` / `c.Batch()` | Yükleme, toplu işler |
| `c.Agents()` | Z.AI uzmanlaşmış ajanları |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | Erişim + moderasyon |
| `c.Tools()` | WebSearch, WebReader, Tokenize |
| `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan kotası ve kullanımı, hesap bilgisi, hesap türü tespiti |
| `c.GetAsyncResult()` / `c.WaitForResult()` | Asenkron görevler için paylaşılan sorgulama |

Bilinmesi gerekenler:

- Varsayılanlar katalogdan gelir: `client.DefaultModel`,
  `client.DefaultFastModel`, `client.DefaultVisionModel` ve OCR/ASR/TTS
  sabitleri. `ChatRequest.ReasoningEffort`, modelin kabul ettiği düzeylere
  göre doğrulanır (GLM-5.3 için `client.EffortLow`, `EffortHigh`, `EffortMax`);
  `client.CatalogEntry(model)` ve `Pricing.Cost(usage)` bağlam boyutlarını ve
  fiyatları sunar.
- `Chat().Stream`, `Anthropic().Stream` ve `Responses().Stream`, `iter.Seq2`
  yineleyicileri döndürür. Döngüden çıkmak akışı kapatır, akış içi bir hata
  parçası (chunk) ise akışı bir `*APIError` ile sonlandırır.
- Z.AI'de `json_schema` yanıt biçimi yoktur. Yapılandırılmış çıktı için
  `ResponseFormat: client.JSONObjectFormat()` ayarlayın ve
  `client.JSONSchemaPrompt(schema)` değerini sistem istemine koyun.
- `Config.Region` (`client.RegionGlobal` veya `client.RegionChina`) her URL'nin
  sahibidir; `Config.BaseURL` varsayılan olarak `Region.PaaSBaseURL()` olur.

→ Örneklerle tam API: **[Kitaplık Kılavuzu](docs/en/library-guide.md)**
→ Oluşturulmuş referans: [pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## Yapılandırma

Kimlik bilgileri şu öncelik sırasıyla çözümlenir (en yüksek olan kazanır):

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
# ZAI_REGION=china                                   # anahtarınız open.bigmodel.cn üzerinde yayımlandıysa; her uç noktayı seçer
# ZAI_CHINA_API_KEY=...                              # ayrı bigmodel.cn kimlik bilgisi (gömme/moderasyon)
# ZAI_MONITOR_TIMEZONE=UTC                           # kota/kullanım API'sinin saat dilimi (varsayılan UTC+8)
```

Kitaplığın `client.NewClientFromEnv()` işlevi aynı değişkenleri süreç
ortamından okur (`.env` dosyasını yüklemez).

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

> ℹ️ Repo kökündeki `zai-claude-config.json`, yer tutucu değerler içeren
> (`"your-zai-api-key-here"`) bir **şablondur**; `go-z-ai coding load claude-code`
> komutunun yazdığı ayarları örnekler. Program bu dosyayı okumaz, gerçek bir
> yapılandırma değildir ve hiçbir kimlik bilgisi içermez; güncel model eşlemesi
> için `go-z-ai coding load claude-code` komutunu çalıştırın.
>
> ⚠️ **Kullanım politikası.** Z.AI'nin coding uç noktası "resmi olarak
> desteklenen araçlar"la sınırlıdır ve SDK tabanlı erişimi yasaklar; bkz.
> [Coding Tools — Compliance](docs/en/coding-tools.md#compliance--usage-policy-).
> `go-z-ai` her istekte tanımlayıcı bir `User-Agent` başlığı gönderir ve
> `coding` alt komutu resmi olarak desteklenen araçları bağlar. Resmi listeye
> alınana kadar, `pkg/client`'ı coding uç noktasına doğrudan kendi
> entegrasyonunuzdan kullanmak kendi sorumluluğunuzdadır.

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
