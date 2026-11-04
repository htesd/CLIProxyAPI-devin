package signature

import (
	"strings"

	"github.com/MoeForge/moego/base64"
	"github.com/MoeForge/moego/ojson"
)

// DevinOpenAIReasoningPrefix marks a Devin GPT reasoning signature carried in a
// Responses reasoning item's encrypted_content.
//
// For OpenAI models Devin streams its thought signature (signature_type
// "openai") as a JSON array of native OpenAI reasoning items, each holding a
// Fernet "gAAAA..." encrypted_content. That array is neither a bare GPT
// signature nor a sealed.v1 envelope, so it has to be wrapped to survive a
// Responses round trip and be handed back to Devin byte-for-byte.
const DevinOpenAIReasoningPrefix = "devin.openai.v1."

// WrapDevinOpenAIReasoning returns the envelope for a raw Devin GPT signature.
// It reports false when raw is not a JSON array containing at least one
// reasoning item with a GPT encrypted_content.
func WrapDevinOpenAIReasoning(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !isDevinOpenAIReasoningArray(trimmed) {
		return "", false
	}
	return DevinOpenAIReasoningPrefix + base64.RawURLEncoding.EncodeToString([]byte(trimmed)), true
}

// IsDevinOpenAIReasoningEnvelope reports whether sig carries the Devin GPT
// reasoning envelope prefix.
func IsDevinOpenAIReasoningEnvelope(sig string) bool {
	return strings.HasPrefix(strings.TrimSpace(sig), DevinOpenAIReasoningPrefix)
}

// UnwrapDevinOpenAIReasoning returns the original Devin signature bytes from an
// envelope produced by WrapDevinOpenAIReasoning.
func UnwrapDevinOpenAIReasoning(sig string) ([]byte, bool) {
	trimmed := strings.TrimSpace(sig)
	if !strings.HasPrefix(trimmed, DevinOpenAIReasoningPrefix) {
		return nil, false
	}
	decoded, errDecode := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(trimmed, DevinOpenAIReasoningPrefix))
	if errDecode != nil {
		return nil, false
	}
	if !isDevinOpenAIReasoningArray(string(decoded)) {
		return nil, false
	}
	return decoded, true
}

func isDevinOpenAIReasoningArray(raw string) bool {
	if !strings.HasPrefix(raw, "[") || !ojson.Valid(raw) {
		return false
	}
	root := ojson.Parse(raw)
	if !root.IsArray() {
		return false
	}
	found := false
	root.ForEach(func(_, item ojson.Result) bool {
		if item.Get("type").String() != "reasoning" {
			return true
		}
		if IsValidGPTReasoningSignature(item.Get("encrypted_content").String()) {
			found = true
			return false
		}
		return true
	})
	return found
}
