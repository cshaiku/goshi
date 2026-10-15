package toolbridge

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestToolCompletion(t *testing.T) {
	for _, tt := range []struct{ words, want []string }{
		{[]string{"gr"}, []string{"graph"}},
		{[]string{"graph", "res"}, []string{"resume"}},
		{[]string{"graph", "run", "--app"}, []string{"--approve-digest"}},
		{[]string{"graph", "context", "--stage", "v"}, []string{"verify"}},
		{[]string{"graph", "schema", "--kind", "w"}, []string{"workflow"}},
		{[]string{"graph", "verify", "--allow"}, []string{}},
	} {
		if got := ToolCompletions(tt.words); !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%v: %v != %v", tt.words, got, tt.want)
		}
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		var out, errout bytes.Buffer
		handled, code := Handle([]string{"tools", "completion", shell}, "bram", "test", &out, &errout)
		if !handled || code != 0 || !strings.Contains(out.String(), "tools complete --") {
			t.Fatalf("%s: %s %s", shell, &out, &errout)
		}
	}
}
