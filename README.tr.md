# Z.AI API İstemcisi

**English** | [简体中文](README.zh.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/view.html?uri=github.com/SamyRai/go-z-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Z.AI (Zhipu AI / BigModel) platforması için bir Go CLI ve istemci
kitaplığı: sohbet tamamlama, modeller, görseller, video, ses, gömme
(embedding), moderasyon, yeniden sıralama, ajanlar, toplu işler, dosya
ayrıştırma, GLM Coding Plan hesap/kota yönetimi ve `@z_ai/coding-helper`'ın
Go port'u — Claude Code, OpenCode, Crush, Factory Droid ve Cursor'ı GLM
Coding Plan'ınıza bağlamak için.

## Kurulum

```bash
go install github.com/SamyRai/go-z-ai@latest
```

Go 1.26.4+ ve bir [Z.AI API anahtarı](https://z.ai/manage-apikey) gerektirir.
Kaynaktan derleme, ilk kimlik doğrulama ve sorun giderme:
**[Başlarken →](docs/en/getting-started.md)**

## Hızlı örnek

```bash
export ZAI_API_KEY=your_api_key_here
zai-client chat create "Goroutine'leri tek bir paragrafta açıkla" --stream
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "Goroutine'leri tek bir paragrafta açıkla"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

Daha fazla çalıştırılabilir örnek — akış (streaming), görsellerin asenkron
sorgulanması, Anthropic `/v1/messages` uç noktası — [`examples/`](examples/)
dizininde.

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

## Kapsam

Sohbet (akış, yapılandırılmış çıktı, derin düşünme, fonksiyon çağrısı, görsel
girdi), Anthropic uyumlu `/v1/messages` uç noktası, modeller, görseller,
video, ses (transkripsiyon + TTS + ses klonlama), OCR ve belge ayrıştırma,
gömme (embedding), moderasyon, yeniden sıralama, ajanlar, dosyalar, toplu
işler, GLM Coding Plan kullanımı/kotası/çoklu hesap yönetimi ve tam ekran
terminal UI'ı (`zai-client tui`). Komutların tamamı için [CLI
Referansı](docs/en/cli-reference.md), Go API'si için [Kitaplık
Kılavuzu](docs/en/library-guide.md)'na bakın.

## Resmî SDK'lar ile ilişkisi

Z.AI / Zhipu, **Python**
([zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python), PyPI
`zai-sdk`), **Node** ([MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4))
ve **Java** ([MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4))
için resmî SDK'lar yayımlar. Resmî bir Go SDK'sı **yoktur** — `go-z-ai` bu
boşluğu doldurur ve aynı API yüzeyi üzerine bir CLI, bir TUI, bölgesel ağ
geçidi değiştirme (`api.z.ai` ↔ `open.bigmodel.cn`) ve çok hesapli GLM
Coding Plan yönetimi ekler.

> ℹ️ Repo kökündeki `zai-claude-config.json`, `zai-client coding load
> claude-code` tarafından kullanılan, yer tutucu değerler içeren
> (`"your-zai-api-key-here"`) bir **şablondur**. Gerçek bir yapılandırma
> değildir ve hiçbir kimlik bilgisi içermez.

## Katkıda bulunma

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
