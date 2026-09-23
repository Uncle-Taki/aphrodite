package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"testing"

	"aphrodite/pkg/config"
)

func signedConfig() config.StorageConfig {
	return config.StorageConfig{
		ImgProxyURL:  "http://localhost:8081/",
		Bucket:       "aphrodite-media",
		ImgProxyKey:  "aabbccdd",
		ImgProxySalt: "11223344",
	}
}

func TestURLUnsignedWhenKeyOrSaltMissing(t *testing.T) {
	cfg := signedConfig()
	cfg.ImgProxyKey = ""
	got := URL(cfg, "images/a.png", 200, 0)
	want := "http://localhost:8081" + expectedProcessing("images/a.png", 200, 0)
	if got != want {
		t.Fatalf("unsigned URL mismatch:\n got %s\nwant %s", got, want)
	}

	cfg = signedConfig()
	cfg.ImgProxySalt = ""
	if again := URL(cfg, "x", 1, 1); !strings.HasPrefix(again, "http://localhost:8081/rs:") {
		t.Fatalf("missing salt must fall back to unsigned: %s", again)
	}
}

func TestURLSignedMatchesHMAC(t *testing.T) {
	cfg := signedConfig()
	got := URL(cfg, "/images/a.png", 300, 200)

	processing := expectedProcessing("/images/a.png", 300, 200)
	mac := hmac.New(sha256.New, decodeHex(cfg.ImgProxyKey))
	mac.Write(decodeHex(cfg.ImgProxySalt))
	mac.Write([]byte(processing))
	want := "http://localhost:8081/" + hex.EncodeToString(mac.Sum(nil)) + processing
	if got != want {
		t.Fatalf("signed URL mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestURLInvalidHexFallsBackToRawBytes(t *testing.T) {
	cfg := signedConfig()
	cfg.ImgProxyKey = "not-hex!"
	cfg.ImgProxySalt = "also not hex"
	got := URL(cfg, "a.png", 10, 10)

	processing := expectedProcessing("a.png", 10, 10)
	mac := hmac.New(sha256.New, []byte("not-hex!"))
	mac.Write([]byte("also not hex"))
	mac.Write([]byte(processing))
	want := "http://localhost:8081/" + hex.EncodeToString(mac.Sum(nil)) + processing
	if got != want {
		t.Fatalf("raw-key signature mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestURLNonPositiveDimensions(t *testing.T) {
	cfg := signedConfig()
	cfg.ImgProxyKey = ""
	got := URL(cfg, "a.png", 0, -5)
	if !strings.Contains(got, "/rs:fit:0:0:0/") {
		t.Fatalf("non-positive dimensions must clamp to zero: %s", got)
	}
}

// expectedProcessing mirrors the processing segment URL builds, deriving the
// escaped source path from url.PathEscape so the assertions don't depend on
// which characters a Go version chooses to escape.
func expectedProcessing(objectPath string, width, height int) string {
	escaped := url.PathEscape("s3://aphrodite-media/" + strings.TrimLeft(objectPath, "/"))
	return "/rs:fit:" + itoa(width) + ":" + itoa(height) + ":0/g:sm/plain/" + escaped
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", -3: "0", 7: "7", 10: "10", 305: "305", 1234567890: "1234567890"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d) = %q, want %q", in, got, want)
		}
	}
}
