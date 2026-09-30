package provider

import "testing"

func TestDetectGitHubAndAllowlistMetadata(t *testing.T) {
	environment := map[string]string{
		"GITHUB_ACTIONS":    "true",
		"GITHUB_REPOSITORY": "company/service",
		"GITHUB_TOKEN":      "must-not-leak",
		"UNRELATED":         "must-not-leak",
	}
	provider := Detect(func(name string) string { return environment[name] })
	if provider.Name() != "github-actions" {
		t.Fatalf("got provider %q", provider.Name())
	}
	metadata := provider.Metadata()
	if metadata["repository"] != "company/service" || len(metadata) != 1 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestDetectLocal(t *testing.T) {
	provider := Detect(func(string) string { return "" })
	if provider.Name() != "local" {
		t.Fatalf("got provider %q", provider.Name())
	}
}
