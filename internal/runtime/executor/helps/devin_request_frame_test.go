package helps

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildDevinRequestFrameKeepsSmallRequestsPlain(t *testing.T) {
	proto := []byte("small request")
	frame, compressed, err := BuildDevinRequestFrame(proto)
	if err != nil {
		t.Fatalf("BuildDevinRequestFrame: %v", err)
	}
	if compressed {
		t.Fatal("small request was compressed")
	}
	if !bytes.Equal(frame, WrapConnectEnvelope(proto)) {
		t.Fatal("small request frame differs from the plain Connect envelope")
	}
}

func TestBuildDevinRequestFrameCompressesLargeRequests(t *testing.T) {
	proto := []byte(strings.Repeat("assistant tool output line with some text\n", (DevinRequestCompressThreshold/40)+1024))
	frame, compressed, err := BuildDevinRequestFrame(proto)
	if err != nil {
		t.Fatalf("BuildDevinRequestFrame: %v", err)
	}
	if !compressed {
		t.Fatal("large request was not compressed")
	}
	if frame[0] != ConnectFlagCompressed {
		t.Fatalf("frame flag = %#x, want %#x", frame[0], ConnectFlagCompressed)
	}
	if len(frame) >= len(proto) {
		t.Fatalf("compressed frame %d bytes is not smaller than %d", len(frame), len(proto))
	}
	flag, payload, err := ReadConnectFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("ReadConnectFrame: %v", err)
	}
	if flag != ConnectFlagCompressed || !bytes.Equal(payload, proto) {
		t.Fatal("compressed frame does not round-trip to the original request")
	}
}
