package sitegen

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"strings"
)

//go:embed locales/*.json
var localeFS embed.FS

// LocaleBundle is a flat key→translation map for one locale.
type LocaleBundle map[string]string

// enBundle is loaded once at init and used as the fallback for any key
// missing in a non-English locale.
var enBundle LocaleBundle

func init() {
	var err error
	enBundle, err = loadBundle("en")
	if err != nil {
		panic(fmt.Sprintf("sitegen: failed to load en locale: %v", err))
	}
}

// loadBundle reads and parses a locale JSON file.
func loadBundle(lang string) (LocaleBundle, error) {
	path := fmt.Sprintf("locales/%s.json", lang)
	data, err := localeFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load locale %s: %w", lang, err)
	}
	var b LocaleBundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse locale %s: %w", lang, err)
	}
	return b, nil
}

// LoadLocale loads a locale bundle, falling back to English for missing keys
// at lookup time (not load time). Returns English if the locale file is
// missing entirely.
func LoadLocale(lang string) LocaleBundle {
	b, err := loadBundle(lang)
	if err != nil {
		log.Printf("i18n: locale %q not found, using en", lang)
		return enBundle
	}
	return b
}

// translateFunc returns a translation function bound to a specific locale
// bundle. Missing keys fall back to English; missing in English returns the
// key itself (developer error). Interpolation replaces {0}, {1}, … with the
// positional args.
func translateFunc(bundle LocaleBundle) func(string, ...any) string {
	return func(key string, args ...any) string {
		s, ok := bundle[key]
		if !ok {
			// Fall back to English.
			if s2, ok2 := enBundle[key]; ok2 {
				s = s2
			} else {
				log.Printf("i18n: key %q missing in en (developer error)", key)
				return key
			}
		}
		if len(args) > 0 {
			s = interpolate(s, args...)
		}
		return s
	}
}

// interpolate replaces {0}, {1}, … tokens with positional arguments.
func interpolate(s string, args ...any) string {
	pairs := make([]string, 0, len(args)*2)
	for i, a := range args {
		pairs = append(pairs, fmt.Sprintf("{%d}", i), fmt.Sprint(a))
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// AvailableLocaleCodes returns the list of locale codes that have a JSON file.
func AvailableLocaleCodes() []string {
	entries, err := fs.ReadDir(localeFS, "locales")
	if err != nil {
		return []string{"en"}
	}
	var codes []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		code := strings.TrimSuffix(e.Name(), ".json")
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return []string{"en"}
	}
	return codes
}
