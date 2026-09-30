package cli

import (
	"io"
	"reflect"
	"testing"
)

func TestParsePreservesCommandArguments(t *testing.T) {
	config, err := Parse([]string{"--execution-name", "build", "--", "bash", "-c", "echo hello"}, func(string) string { return "" }, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if config.Command != "bash" || !reflect.DeepEqual(config.Arguments, []string{"-c", "echo hello"}) {
		t.Fatalf("unexpected command: %q %#v", config.Command, config.Arguments)
	}
	if config.ExecutionName != "build" {
		t.Fatalf("unexpected execution name: %q", config.ExecutionName)
	}
}

func TestParseRequiresSeparator(t *testing.T) {
	if _, err := Parse([]string{"bash", "-c", "true"}, func(string) string { return "" }, io.Discard); err == nil {
		t.Fatal("expected missing separator error")
	}
}

func TestResolveLogDirectoryPrecedence(t *testing.T) {
	environment := map[string]string{
		"RUNNER_TEMP": "/runner",
		"TMPDIR":      "/tmpdir",
	}
	getenv := func(name string) string { return environment[name] }
	if got := ResolveLogDirectory(getenv); got != "/runner/citrace" {
		t.Fatalf("got %q", got)
	}
	environment["CITRACE_LOG_DIR"] = "/explicit"
	if got := ResolveLogDirectory(getenv); got != "/explicit" {
		t.Fatalf("got %q", got)
	}
}

func TestExplicitQuietFalseOverridesEnvironment(t *testing.T) {
	getenv := func(name string) string {
		if name == "CITRACE_QUIET" {
			return "true"
		}
		return ""
	}
	config, err := Parse([]string{"--quiet=false", "--", "true"}, getenv, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if config.Quiet {
		t.Fatal("explicit quiet=false did not override environment")
	}
}
