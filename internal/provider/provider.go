package provider

import "strings"

type Metadata map[string]string

type Provider interface {
	Name() string
	Detect() bool
	Metadata() Metadata
}

type environmentProvider struct {
	name      string
	detect    func() bool
	getenv    func(string) string
	allowlist map[string]string
}

func (provider environmentProvider) Name() string { return provider.name }
func (provider environmentProvider) Detect() bool { return provider.detect() }
func (provider environmentProvider) Metadata() Metadata {
	metadata := Metadata{}
	for environmentName, recordName := range provider.allowlist {
		if isSensitive(environmentName) {
			continue
		}
		if value := provider.getenv(environmentName); value != "" {
			metadata[recordName] = value
		}
	}
	return metadata
}

func Detect(getenv func(string) string) Provider {
	providers := []Provider{
		github(getenv),
		azure(getenv),
		generic(getenv),
	}
	for _, candidate := range providers {
		if candidate.Detect() {
			return candidate
		}
	}
	return environmentProvider{name: "local", detect: func() bool { return true }, getenv: getenv}
}

func isSensitive(name string) bool {
	upperName := strings.ToUpper(name)
	for _, term := range []string{"TOKEN", "SECRET", "PASSWORD", "PASSWD", "KEY", "CREDENTIAL", "AUTH", "COOKIE", "SESSION"} {
		if strings.Contains(upperName, term) {
			return true
		}
	}
	return false
}

func generic(getenv func(string) string) Provider {
	return environmentProvider{
		name:   "generic-ci",
		getenv: getenv,
		detect: func() bool { return strings.EqualFold(getenv("CI"), "true") },
	}
}
