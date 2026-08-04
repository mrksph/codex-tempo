package projectresolver

import "testing"

func TestNormalizeRemote(t *testing.T) {
	cases := map[string]string{
		"git@github.com:OpenAI/codex.git":     "github.com/openai/codex",
		"https://github.com/OpenAI/codex.git": "github.com/openai/codex",
	}
	for input, want := range cases {
		if got := normalizeRemote(input); got != want {
			t.Errorf("%q => %q, want %q", input, got, want)
		}
	}
}
