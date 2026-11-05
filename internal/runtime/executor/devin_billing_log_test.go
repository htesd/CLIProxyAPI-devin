package executor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/encoding/protowire"
)

// devinBillingResponse builds a Connect response whose usage frame reports
// api_provider (field 6) and used_provider_plan (field 13).
func devinBillingResponse(apiProvider uint64, usedProviderPlan bool) []byte {
	var usage []byte
	usage = protowire.AppendTag(usage, 2, protowire.VarintType)
	usage = protowire.AppendVarint(usage, 10)
	usage = protowire.AppendTag(usage, 3, protowire.VarintType)
	usage = protowire.AppendVarint(usage, 20)
	usage = protowire.AppendTag(usage, 6, protowire.VarintType)
	usage = protowire.AppendVarint(usage, apiProvider)
	if usedProviderPlan {
		usage = protowire.AppendTag(usage, 13, protowire.VarintType)
		usage = protowire.AppendVarint(usage, 1)
	}
	usage = protowire.AppendTag(usage, 9, protowire.BytesType)
	usage = protowire.AppendString(usage, "gpt-6-1-sol-medium")

	var frame []byte
	frame = protowire.AppendTag(frame, 7, protowire.BytesType)
	frame = protowire.AppendBytes(frame, usage)

	var buf bytes.Buffer
	buf.Write(helps.WrapConnectEnvelope(frame))
	buf.Write(helps.WrapConnectEnvelopeWithFlag(helps.ConnectFlagEndStream, []byte(`{}`)))
	return buf.Bytes()
}

func runDevinBillingCase(t *testing.T, body []byte, stream bool) []string {
	t.Helper()
	mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/connect+proto"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil
	})
	hook := new(logtest.Hook)
	log.StandardLogger().AddHook(hook)
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(make(log.LevelHooks)) })

	exec := NewDevinExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		ID:         "devin-billing",
		Provider:   "devin",
		FileName:   "/data/auths/devin-gpt.json",
		Attributes: map[string]string{"api_key": "test-key"},
	}
	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", mockRT)
	req := cliproxyexecutor.Request{
		Model:   "devin/gpt-6-1-sol",
		Payload: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
	}
	opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAI}

	if stream {
		result, err := exec.ExecuteStream(ctx, auth, req, opts)
		if err != nil {
			t.Fatalf("ExecuteStream failed: %v", err)
		}
		for range result.Chunks {
		}
	} else if _, err := exec.Execute(ctx, auth, req, opts); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var lines []string
	for _, entry := range hook.AllEntries() {
		if strings.HasPrefix(entry.Message, "devin billing:") {
			lines = append(lines, entry.Message)
		}
	}
	return lines
}

func TestDevinBillingLogReportsProviderPlan(t *testing.T) {
	for _, stream := range []bool{false, true} {
		lines := runDevinBillingCase(t, devinBillingResponse(87, true), stream)
		if len(lines) != 1 {
			t.Fatalf("stream=%t: billing log lines = %d (%v), want 1", stream, len(lines), lines)
		}
		want := "auth=devin-gpt.json model=gpt-6-1-sol-medium source=provider_plan api_provider=OPENAI_SIWC provider_plan=true"
		if !strings.Contains(lines[0], want) {
			t.Fatalf("stream=%t: billing log = %q, want it to contain %q", stream, lines[0], want)
		}
	}
}

func TestDevinBillingLogReportsDevinQuotaFallback(t *testing.T) {
	for _, stream := range []bool{false, true} {
		lines := runDevinBillingCase(t, devinBillingResponse(66, false), stream)
		if len(lines) != 1 || !strings.Contains(lines[0], "source=devin_quota api_provider=66 provider_plan=false") {
			t.Fatalf("stream=%t: billing log = %v, want devin_quota fallback", stream, lines)
		}
	}
}
