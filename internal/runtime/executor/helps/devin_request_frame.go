package helps

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

const (
	// DevinRequestCompressThreshold is the request size above which the
	// GetChatMessage frame is gzip-compressed. Smaller requests keep the plain
	// Connect frame so ordinary traffic is unchanged on the wire.
	DevinRequestCompressThreshold = 4 << 20
	// DevinRequestBodyLimit is the largest request body Devin's edge accepts;
	// larger bodies are rejected by its nginx with a bare HTML 413.
	DevinRequestBodyLimit = 10 << 20
	// DevinConnectContentEncodingHeader announces a compressed Connect request frame.
	DevinConnectContentEncodingHeader = "Connect-Content-Encoding"
	// DevinConnectAcceptEncodingHeader advertises compressed response frames.
	DevinConnectAcceptEncodingHeader = "Connect-Accept-Encoding"
)

// BuildDevinRequestFrame wraps a GetChatMessage protobuf in a Connect frame.
// Requests above DevinRequestCompressThreshold are gzip-compressed (Connect
// flag 0x01), which Devin accepts; it reports whether compression was applied.
func BuildDevinRequestFrame(protoBytes []byte) ([]byte, bool, error) {
	if len(protoBytes) <= DevinRequestCompressThreshold {
		return WrapConnectEnvelope(protoBytes), false, nil
	}
	var buf bytes.Buffer
	gz, errWriter := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if errWriter != nil {
		return nil, false, fmt.Errorf("create gzip writer for devin request: %w", errWriter)
	}
	if _, errWrite := gz.Write(protoBytes); errWrite != nil {
		return nil, false, fmt.Errorf("gzip devin request: %w", errWrite)
	}
	if errClose := gz.Close(); errClose != nil {
		return nil, false, fmt.Errorf("finish gzip devin request: %w", errClose)
	}
	return WrapConnectEnvelopeWithFlag(ConnectFlagCompressed, buf.Bytes()), true, nil
}
