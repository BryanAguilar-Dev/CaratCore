package docker

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

// TestDockerKeyFingerprint proves that the embedded key is the one Docker's
// guide names. It computes the OpenPGP v4 fingerprint (RFC 4880, 12.2): the
// SHA-1 of 0x99, the two-byte length of the public key packet and its body.
func TestDockerKeyFingerprint(t *testing.T) {
	data, err := dearmor(dockerKey)
	if err != nil {
		t.Fatal(err)
	}
	tag, body, err := firstPacket(data)
	if err != nil {
		t.Fatal(err)
	}
	if tag != 6 || len(body) == 0 || body[0] != 4 {
		t.Fatalf("first packet is not a v4 public key (tag %d)", tag)
	}

	h := sha1.New()
	h.Write([]byte{0x99})
	binary.Write(h, binary.BigEndian, uint16(len(body)))
	h.Write(body)
	got := fmt.Sprintf("%X", h.Sum(nil))

	want := strings.ReplaceAll(keyFingerprint, " ", "")
	if got != want {
		t.Fatalf("embedded key fingerprint is %s, want %s", got, want)
	}
}

// dearmor returns the binary data of an ASCII-armored block: the base64 lines
// after the headers, without the "=" checksum line.
func dearmor(armored string) ([]byte, error) {
	var b64 strings.Builder
	inBody, afterHeaders := false, false
	for _, line := range strings.Split(armored, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "-----BEGIN"):
			inBody = true
		case strings.HasPrefix(line, "-----END"):
			inBody = false
		case !inBody:
		case !afterHeaders:
			afterHeaders = line == ""
		case strings.HasPrefix(line, "="):
		default:
			b64.WriteString(line)
		}
	}
	return base64.StdEncoding.DecodeString(b64.String())
}

// firstPacket returns the tag and body of the first OpenPGP packet, in the old
// or the new header format (RFC 4880, 4.2).
func firstPacket(data []byte) (tag int, body []byte, err error) {
	if len(data) < 2 || data[0]&0x80 == 0 {
		return 0, nil, fmt.Errorf("not an OpenPGP packet")
	}
	var length, start int
	if data[0]&0x40 == 0 { // old format
		tag = int(data[0]>>2) & 0x0F
		switch data[0] & 0x03 {
		case 0:
			length, start = int(data[1]), 2
		case 1:
			length, start = int(binary.BigEndian.Uint16(data[1:])), 3
		case 2:
			length, start = int(binary.BigEndian.Uint32(data[1:])), 5
		default:
			return 0, nil, fmt.Errorf("indeterminate packet length")
		}
	} else { // new format
		tag = int(data[0] & 0x3F)
		switch o := int(data[1]); {
		case o < 192:
			length, start = o, 2
		case o < 224:
			length, start = (o-192)<<8+int(data[2])+192, 3
		case o == 255:
			length, start = int(binary.BigEndian.Uint32(data[2:])), 6
		default:
			return 0, nil, fmt.Errorf("partial packet length")
		}
	}
	if start+length > len(data) {
		return 0, nil, fmt.Errorf("packet longer than the data")
	}
	return tag, data[start : start+length], nil
}
