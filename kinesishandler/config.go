package kinesis

import (
	"bytes"
	"os"
	"regexp"
	"strings"
)

// trimBOM trims the Byte-Order-Marks from the beginning of the file.
// this is for Windows compatibility only.
// see https://github.com/influxdata/telegraf/issues/1378
func trimBOM(f []byte) []byte {
	return bytes.TrimPrefix(f, []byte("\xef\xbb\xbf"))
}

// escapeEnv escapes a value for inserting into a TOML string.
func escapeEnv(value string) string {
	return strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
	).Replace(value)
}

// parseConfig will look for Environment variables and replace them.
func parseConfig(contents []byte) []byte {
	envVarRe := regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

	parameters := envVarRe.FindAllSubmatch(contents, -1)
	for _, parameter := range parameters {
		if len(parameter) != 3 {
			continue
		}

		var envVar []byte
		if parameter[1] != nil {
			envVar = parameter[1]
		} else if parameter[2] != nil {
			envVar = parameter[2]
		} else {
			continue
		}

		envVal, ok := os.LookupEnv(strings.TrimPrefix(string(envVar), "$"))
		if ok {
			envVal = escapeEnv(envVal)
			contents = bytes.Replace(contents, parameter[0], []byte(envVal), 1)
		}
	}

	return contents
}
