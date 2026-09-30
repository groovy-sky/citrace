package logging

import (
	"reflect"
	"testing"
)

func TestRedactArguments(t *testing.T) {
	input := []string{"--token", "secret", "--api-key=other", "visible"}
	want := []string{"--token", "[REDACTED]", "--api-key=[REDACTED]", "visible"}
	if got := RedactArguments(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	if input[1] != "secret" {
		t.Fatal("redaction modified child arguments")
	}
}
