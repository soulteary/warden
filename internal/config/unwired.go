package config

import (
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// unwiredKeys are the configuration-file keys that are parsed but have no
// effect on the running service. The service uses its built-in value for each
// of them (see docs/enUS/CONFIGURATION.md, "YAML schema and application wiring").
var unwiredKeys = map[string][]string{
	"server": {"read_timeout", "write_timeout", "shutdown_timeout", "idle_timeout", "max_header_bytes"},
	"redis":  {"db"},
	"cache":  {"ttl", "update_interval"},
	"http":   {"max_retries", "retry_delay"},
}

// UnwiredKeysInFile returns the keys present in the configuration file that
// the service parses but does not apply, as sorted "section.key" names. A file
// that cannot be read or parsed yields nil: loading the configuration reports
// that error.
func UnwiredKeysInFile(path string) []string {
	if path == "" {
		return nil
	}
	validated, err := validateConfigPath(path)
	if err != nil {
		return nil
	}
	// #nosec G304 -- the operator-selected configuration file, validated above
	data, err := os.ReadFile(validated)
	if err != nil {
		return nil
	}
	var document map[string]map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil
	}
	var found []string
	for section, keys := range unwiredKeys {
		for _, key := range keys {
			if _, present := document[section][key]; present {
				found = append(found, section+"."+key)
			}
		}
	}
	sort.Strings(found)
	return found
}
