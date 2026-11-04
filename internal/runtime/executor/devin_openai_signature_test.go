package executor

import (
	"testing"

	internalsignature "github.com/router-for-me/CLIProxyAPI/v8/internal/signature"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/signature/signaturetest"
)

func TestParseSignatureBytesRestoresDevinGPTSignature(t *testing.T) {
	raw := signaturetest.DevinOpenAIReasoning()
	wrapped, ok := internalsignature.WrapDevinOpenAIReasoning(raw)
	if !ok {
		t.Fatal("wrap failed")
	}
	got, sigType := parseSignatureBytes(wrapped)
	if sigType != "openai" {
		t.Fatalf("signature type = %q, want openai", sigType)
	}
	if string(got) != raw {
		t.Fatalf("restored bytes differ:\n got %s\nwant %s", got, raw)
	}
}
