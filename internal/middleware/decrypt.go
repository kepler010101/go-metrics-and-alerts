package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
)

const encryptedHeader = "X-Encrypted"

func WithDecrypt(key *rsa.PrivateKey) func(http.Handler) http.Handler {
	if key == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil || r.Header.Get(encryptedHeader) != "1" {
				next.ServeHTTP(w, r)
				return
			}

			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			plain, err := decryptPayload(key, data)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			r.Header.Del(encryptedHeader)

			next.ServeHTTP(w, r)
		})
	}
}

func decryptPayload(key *rsa.PrivateKey, data []byte) ([]byte, error) {
	chunkSize := key.Size()
	if chunkSize == 0 {
		return nil, fmt.Errorf("invalid key")
	}
	if len(data)%chunkSize != 0 {
		return nil, fmt.Errorf("invalid encrypted data size")
	}

	var out bytes.Buffer
	for offset := 0; offset < len(data); offset += chunkSize {
		block := data[offset : offset+chunkSize]
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, key, block)
		if err != nil {
			return nil, err
		}
		out.Write(plain)
	}
	return out.Bytes(), nil
}
