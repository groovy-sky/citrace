package logging

import "strings"

func RedactArguments(arguments []string) []string {
	redacted := append([]string(nil), arguments...)
	for index := 0; index < len(redacted); index++ {
		argument := redacted[index]
		name, _, hasValue := strings.Cut(argument, "=")
		if !secretFlag(name) {
			continue
		}
		if hasValue {
			redacted[index] = name + "=[REDACTED]"
		} else if index+1 < len(redacted) {
			redacted[index+1] = "[REDACTED]"
			index++
		}
	}
	return redacted
}

func secretFlag(value string) bool {
	switch strings.ToLower(value) {
	case "--token", "--password", "--api-key", "--authorization":
		return true
	default:
		return false
	}
}
