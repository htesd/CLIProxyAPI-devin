package signaturetest

import "github.com/MoeForge/moego/base64"

// GPTReasoning builds a synthetic Fernet-shaped GPT reasoning encrypted_content.
// The payload is generated filler, not captured or replayable state.
func GPTReasoning() string {
	payload := make([]byte, 1+8+16+16+32)
	payload[0] = 0x80
	for i := 9; i < len(payload); i++ {
		payload[i] = byte(i)
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

// DevinOpenAIReasoning builds the raw signature Devin streams for OpenAI
// models: a JSON array holding one native OpenAI reasoning item.
func DevinOpenAIReasoning() string {
	return `[{"id":"rs_test","summary":[],"type":"reasoning","content":[],"encrypted_content":"` + GPTReasoning() + `","status":""}]`
}
