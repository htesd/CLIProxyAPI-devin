package responses

import (
	"context"
	"strings"
	"testing"

	"github.com/MoeForge/moego/ojson"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/signature"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/signature/signaturetest"
)

func devinThoughtSignatureEvent(t *testing.T, raw string) []byte {
	t.Helper()
	doc := ojson.NewDocString(`{"index":0,"delta":{"signature":"","signature_type":"openai","type":"thought_signature"},"event_type":"step.delta"}`)
	if err := doc.Set("delta.signature", raw); err != nil {
		t.Fatalf("set signature: %v", err)
	}
	return []byte("event: step.delta\ndata: " + doc.String() + "\n\n")
}

func TestConvertInteractionsResponseToOpenAIResponsesWrapsDevinGPTSignature(t *testing.T) {
	var param any
	raw := signaturetest.DevinOpenAIReasoning()
	var out [][]byte
	for _, chunk := range [][]byte{
		[]byte("event: step.start\ndata: {\"index\":0,\"step\":{\"type\":\"thought\"},\"event_type\":\"step.start\"}\n\n"),
		[]byte("event: step.delta\ndata: {\"index\":0,\"delta\":{\"content\":{\"text\":\"thinking\",\"type\":\"text\"},\"type\":\"thought_summary\"},\"event_type\":\"step.delta\"}\n\n"),
		devinThoughtSignatureEvent(t, raw[:30]),
		devinThoughtSignatureEvent(t, raw),
		[]byte("event: step.stop\ndata: {\"index\":0,\"event_type\":\"step.stop\"}\n\n"),
		[]byte("event: interaction.completed\ndata: {\"interaction\":{\"id\":\"interaction_1\",\"status\":\"completed\",\"object\":\"interaction\",\"model\":\"gpt-6-astra\"},\"event_type\":\"interaction.completed\"}\n\n"),
	} {
		out = append(out, ConvertInteractionsResponseToOpenAIResponses(context.Background(), "gpt-6-astra", []byte(`{"model":"gpt-6-astra"}`), nil, chunk, &param)...)
	}

	donePayload := findResponsesEventPayload(out, "response.output_item.done")
	got := ojson.GetBytesNoCopy(donePayload, "item.encrypted_content").String()
	if !strings.HasPrefix(got, signature.DevinOpenAIReasoningPrefix) {
		t.Fatalf("done encrypted_content = %q, want Devin GPT envelope. Payload: %s", got, string(donePayload))
	}
	unwrapped, ok := signature.UnwrapDevinOpenAIReasoning(got)
	if !ok || string(unwrapped) != raw {
		t.Fatalf("envelope does not restore the raw Devin signature: ok=%v got=%s", ok, string(unwrapped))
	}
	completedPayload := findResponsesEventPayload(out, "response.completed")
	if c := ojson.GetBytesNoCopy(completedPayload, "response.output.0.encrypted_content").String(); c != got {
		t.Fatalf("completed encrypted_content = %q, want %q", c, got)
	}
}

func TestConvertOpenAIResponsesRequestKeepsDevinGPTReasoning(t *testing.T) {
	wrapped, ok := signature.WrapDevinOpenAIReasoning(signaturetest.DevinOpenAIReasoning())
	if !ok {
		t.Fatal("wrap failed")
	}
	doc := ojson.NewDocString(`{"model":"gpt-6-astra","input":[
		{"type":"message","role":"user","content":[{"type":"input_text","text":"go"}]},
		{"type":"reasoning","summary":[{"type":"summary_text","text":"17*23=391"}],"encrypted_content":""},
		{"type":"function_call","name":"ping","call_id":"c1","arguments":"{}"}
	]}`)
	if err := doc.Set("input.1.encrypted_content", wrapped); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, _ := ConvertOpenAIResponsesRequestToInteractions("gpt-6-astra", doc.Bytes(), false)
	if got := ojson.GetBytesNoCopy(out, "input.1.type").String(); got != "thought" {
		t.Fatalf("input.1.type = %q, want thought. Output: %s", got, string(out))
	}
	if got := ojson.GetBytesNoCopy(out, "input.1.signature").String(); got != wrapped {
		t.Fatalf("signature = %q, want envelope. Output: %s", got, string(out))
	}
	if got := ojson.GetBytesNoCopy(out, "input.1.content.0.text").String(); got != "17*23=391" {
		t.Fatalf("thought text = %q", got)
	}
}
