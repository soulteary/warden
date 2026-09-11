package locales

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// referenceLocale is the fallback language: i18n-kit falls back to it for any key the
// active locale is missing, so it defines the complete key set.
const referenceLocale = "en.json"

// loadedLocales must stay in sync with loadEmbeddedTranslations in internal/i18n/i18n.go.
// A file present in the embedded FS but absent here would never be served.
var loadedLocales = []string{"en.json", "zh.json", "fr.json", "it.json", "ja.json", "de.json", "ko.json"}

// intentionallyIdentical lists key/locale pairs whose translation is legitimately
// byte-identical to English (loan words, protocol names, and the like). Everything else
// that matches English exactly is treated as an untranslated string, which is how six
// user-facing http.* messages silently shipped in English in every language.
var intentionallyIdentical = map[string]map[string]bool{
	// "some.key": {"de": true},
}

func readLocale(t *testing.T, name string) map[string]string {
	t.Helper()
	data, err := FS.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return m
}

func langOf(file string) string { return strings.TrimSuffix(file, ".json") }

// TestEmbeddedLocalesAreAllLoaded fails when a translation file is embedded but never
// registered with the bundle (or vice versa).
func TestEmbeddedLocalesAreAllLoaded(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read embedded dir: %v", err)
	}
	var embedded []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			embedded = append(embedded, e.Name())
		}
	}
	loaded := append([]string(nil), loadedLocales...)
	sort.Strings(embedded)
	sort.Strings(loaded)
	if strings.Join(embedded, ",") != strings.Join(loaded, ",") {
		t.Fatalf("embedded locale files %v do not match the list loaded by internal/i18n %v", embedded, loaded)
	}
}

// TestLocaleKeySetsMatchReference is the completeness gate: every locale must define
// exactly the keys en.json defines. Missing keys silently fall back to English at runtime,
// which is how five locales drifted 18 keys behind without anything failing.
func TestLocaleKeySetsMatchReference(t *testing.T) {
	reference := readLocale(t, referenceLocale)

	for _, file := range loadedLocales {
		if file == referenceLocale {
			continue
		}
		t.Run(langOf(file), func(t *testing.T) {
			locale := readLocale(t, file)

			var missing, extra []string
			for key := range reference {
				if _, ok := locale[key]; !ok {
					missing = append(missing, key)
				}
			}
			for key := range locale {
				if _, ok := reference[key]; !ok {
					extra = append(extra, key)
				}
			}
			sort.Strings(missing)
			sort.Strings(extra)

			if len(missing) > 0 {
				t.Errorf("%s is missing %d key(s) present in %s:\n  %s",
					file, len(missing), referenceLocale, strings.Join(missing, "\n  "))
			}
			if len(extra) > 0 {
				t.Errorf("%s defines %d key(s) absent from %s (dead translations):\n  %s",
					file, len(extra), referenceLocale, strings.Join(extra, "\n  "))
			}
		})
	}
}

// formatVerb matches printf verbs such as %s, %d and %%.
var formatVerb = regexp.MustCompile(`%[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)

// TestLocaleFormatVerbsMatchReference guards the translated strings that reach
// fmt.Sprintf. A dropped, added or reordered verb corrupts the rendered message.
func TestLocaleFormatVerbsMatchReference(t *testing.T) {
	reference := readLocale(t, referenceLocale)

	for _, file := range loadedLocales {
		if file == referenceLocale {
			continue
		}
		t.Run(langOf(file), func(t *testing.T) {
			locale := readLocale(t, file)
			for key, refValue := range reference {
				value, ok := locale[key]
				if !ok {
					continue // reported by TestLocaleKeySetsMatchReference
				}
				want := formatVerb.FindAllString(refValue, -1)
				got := formatVerb.FindAllString(value, -1)
				if strings.Join(want, "") != strings.Join(got, "") {
					t.Errorf("%s[%q]: format verbs %v do not match %s verbs %v\n  en: %s\n  %s: %s",
						file, key, got, referenceLocale, want, refValue, langOf(file), value)
				}
			}
		})
	}
}

// TestLocaleValuesAreTranslated catches strings copied verbatim from English. Add a
// deliberate exception to intentionallyIdentical rather than deleting this check.
func TestLocaleValuesAreTranslated(t *testing.T) {
	reference := readLocale(t, referenceLocale)

	for _, file := range loadedLocales {
		if file == referenceLocale {
			continue
		}
		lang := langOf(file)
		t.Run(lang, func(t *testing.T) {
			locale := readLocale(t, file)
			var identical []string
			for key, refValue := range reference {
				value, ok := locale[key]
				if !ok || value != refValue {
					continue
				}
				// Very short values are frequently identical across languages by nature
				// (identifiers, units); only flag real sentences.
				if len([]rune(refValue)) < 8 {
					continue
				}
				if intentionallyIdentical[key][lang] {
					continue
				}
				identical = append(identical, key+" = "+refValue)
			}
			sort.Strings(identical)
			if len(identical) > 0 {
				t.Errorf("%s has %d value(s) identical to %s (untranslated?):\n  %s",
					file, len(identical), referenceLocale, strings.Join(identical, "\n  "))
			}
		})
	}
}
