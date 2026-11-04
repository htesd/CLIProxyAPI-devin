package signature

import (
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/signature/signaturetest"
)

func TestDevinOpenAIReasoningRoundTrip(t *testing.T) {
	raw := signaturetest.DevinOpenAIReasoning()
	wrapped, ok := WrapDevinOpenAIReasoning(raw)
	if !ok {
		t.Fatal("expected Devin GPT signature to be wrapped")
	}
	if !strings.HasPrefix(wrapped, DevinOpenAIReasoningPrefix) || !IsDevinOpenAIReasoningEnvelope(wrapped) {
		t.Fatalf("wrapped signature lacks prefix: %q", wrapped)
	}
	got, ok := UnwrapDevinOpenAIReasoning(wrapped)
	if !ok {
		t.Fatal("expected envelope to unwrap")
	}
	if string(got) != raw {
		t.Fatalf("round trip changed bytes:\n got %s\nwant %s", got, raw)
	}
}

func TestWrapDevinOpenAIReasoningRejectsOtherShapes(t *testing.T) {
	cases := map[string]string{
		"bare gpt signature":      signaturetest.GPTReasoning(),
		"sealed envelope":         "sealed.v1.abc",
		"array without reasoning": `[{"type":"message","encrypted_content":"` + signaturetest.GPTReasoning() + `"}]`,
		"reasoning without gpt":   `[{"type":"reasoning","encrypted_content":"not-a-signature"}]`,
		"truncated stream":        signaturetest.DevinOpenAIReasoning()[:40],
		"empty":                   "",
	}
	for name, raw := range cases {
		if _, ok := WrapDevinOpenAIReasoning(raw); ok {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestUnwrapDevinOpenAIReasoningRejectsTamperedEnvelope(t *testing.T) {
	if _, ok := UnwrapDevinOpenAIReasoning(DevinOpenAIReasoningPrefix + "!!!"); ok {
		t.Fatal("expected invalid base64 to be rejected")
	}
	if _, ok := UnwrapDevinOpenAIReasoning(DevinOpenAIReasoningPrefix + "W10"); ok {
		t.Fatal("expected empty array to be rejected")
	}
	if _, ok := UnwrapDevinOpenAIReasoning(signaturetest.GPTReasoning()); ok {
		t.Fatal("expected bare signature without prefix to be rejected")
	}
}
