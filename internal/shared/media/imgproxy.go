package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"

	"aphrodite/pkg/config"
)

func URL(cfg config.StorageConfig, objectPath string, width, height int) string {
	base := strings.TrimRight(cfg.ImgProxyURL, "/")
	encoded := url.PathEscape("s3://" + cfg.Bucket + "/" + strings.TrimLeft(objectPath, "/"))
	processing := "/rs:fit:" + itoa(width) + ":" + itoa(height) + ":0/g:sm/plain/" + encoded
	if cfg.ImgProxyKey == "" || cfg.ImgProxySalt == "" {
		return base + processing
	}
	mac := hmac.New(sha256.New, decodeHex(cfg.ImgProxyKey))
	mac.Write(decodeHex(cfg.ImgProxySalt))
	mac.Write([]byte(processing))
	return base + "/" + hex.EncodeToString(mac.Sum(nil)) + processing
}

func decodeHex(value string) []byte {
	b, err := hex.DecodeString(value)
	if err != nil {
		return []byte(value)
	}
	return b
}

func itoa(value int) string {
	if value <= 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}
