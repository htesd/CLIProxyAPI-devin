package executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MoeForge/moego/base64"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
)

type capturedDevinRequest struct {
	calls   int
	headers http.Header
	body    []byte
}

func runDevinRequestSizeCase(t *testing.T, userText string) (*capturedDevinRequest, error) {
	t.Helper()
	captured := &capturedDevinRequest{}
	mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		captured.calls++
		captured.headers = req.Header.Clone()
		body, _ := io.ReadAll(req.Body)
		captured.body = body
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/connect+proto"}},
			Body:       io.NopCloser(bytes.NewReader(devinBillingResponse(87, true))),
		}, nil
	})
	payload, errMarshal := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "user", "content": userText}},
	})
	if errMarshal != nil {
		t.Fatalf("marshal payload: %v", errMarshal)
	}
	exec := NewDevinExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{ID: "devin-size", Provider: "devin", Attributes: map[string]string{"api_key": "test-key"}}
	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", mockRT)
	req := cliproxyexecutor.Request{Model: "devin/gpt-6-1-sol", Payload: payload}
	_, err := exec.Execute(ctx, auth, req, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAI})
	return captured, err
}

func TestDevinExecutorKeepsOrdinaryRequestsUncompressed(t *testing.T) {
	captured, err := runDevinRequestSizeCase(t, "hello")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got := captured.headers.Get(helps.DevinConnectContentEncodingHeader); got != "" {
		t.Fatalf("Connect-Content-Encoding = %q, want empty", got)
	}
	if captured.body[0] != helps.ConnectFlagData {
		t.Fatalf("frame flag = %#x, want plain", captured.body[0])
	}
}

func TestDevinExecutorCompressesLargeRequests(t *testing.T) {
	text := strings.Repeat("tool output: compiled 42 files without errors\n", 6<<20/46)
	captured, err := runDevinRequestSizeCase(t, text)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got := captured.headers.Get(helps.DevinConnectContentEncodingHeader); got != "gzip" {
		t.Fatalf("Connect-Content-Encoding = %q, want gzip", got)
	}
	if captured.body[0] != helps.ConnectFlagCompressed {
		t.Fatalf("frame flag = %#x, want compressed", captured.body[0])
	}
	_, proto, errRead := helps.ReadConnectFrame(bytes.NewReader(captured.body))
	if errRead != nil {
		t.Fatalf("ReadConnectFrame: %v", errRead)
	}
	if !bytes.Contains(proto, []byte("compiled 42 files")) || len(captured.body) >= len(proto) {
		t.Fatalf("compressed body %d bytes does not carry the %d-byte request", len(captured.body), len(proto))
	}
}

func TestDevinExecutorRejectsRequestsAboveUpstreamLimitLocally(t *testing.T) {
	random := make([]byte, 12<<20)
	if _, errRand := rand.Read(random); errRand != nil {
		t.Fatalf("rand: %v", errRand)
	}
	captured, err := runDevinRequestSizeCase(t, base64.StdEncoding.EncodeToString(random))
	if captured.calls != 0 {
		t.Fatalf("upstream called %d times for an oversized request", captured.calls)
	}
	var se statusErr
	if !errors.As(err, &se) || se.code != http.StatusBadRequest {
		t.Fatalf("err = %v, want local 400 statusErr", err)
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if errJSON := json.Unmarshal([]byte(se.msg), &body); errJSON != nil {
		t.Fatalf("error body is not JSON: %v (%q)", errJSON, se.msg)
	}
	if body.Error.Code != "context_length_exceeded" || body.Error.Type != "invalid_request_error" {
		t.Fatalf("error = %+v, want invalid_request_error/context_length_exceeded", body.Error)
	}
	if !strings.Contains(body.Error.Message, "request too large for Devin") {
		t.Fatalf("message = %q", body.Error.Message)
	}
}
