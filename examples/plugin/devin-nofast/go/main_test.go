package main

import (
	"testing"

	"github.com/MoeForge/moego/ojson"
)

func TestStripFast(t *testing.T) {
	cases := []struct {
		name, in, path, want string
		gone                 bool
	}{
		{"service_tier removed", `{"service_tier":"priority","generation_config":{"thinking_level":"high"}}`, "service_tier", "", true},
		{"level fast removed", `{"generation_config":{"thinking_level":"fast"}}`, "generation_config.thinking_level", "", true},
		{"level priority removed", `{"generation_config":{"thinking_level":"PRIORITY"}}`, "generation_config.thinking_level", "", true},
		{"level high-fast trimmed", `{"generation_config":{"thinking_level":"high-fast"}}`, "generation_config.thinking_level", "high", false},
		{"level medium-priority trimmed", `{"generation_config":{"thinking_level":"medium-priority"}}`, "generation_config.thinking_level", "medium", false},
		{"plain high kept", `{"generation_config":{"thinking_level":"high"},"x":1}`, "generation_config.thinking_level", "high", false},
	}
	for _, c := range cases {
		out := stripFast([]byte(c.in))
		res := ojson.GetBytes(out, c.path)
		if c.gone && res.Exists() {
			t.Errorf("%s: %s should be removed, got %s", c.name, c.path, out)
		}
		if !c.gone && res.String() != c.want {
			t.Errorf("%s: want %q, got %q (%s)", c.name, c.want, res.String(), out)
		}
	}
	plain := []byte(`{"generation_config":{"thinking_level":"high"},"contents":[{"role":"user"}]}`)
	if string(stripFast(plain)) != string(plain) {
		t.Errorf("body without fast selection must be returned unchanged")
	}
}
