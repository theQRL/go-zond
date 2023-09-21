//go:build !nacl && !js && cgo && !gofuzz
// +build !nacl,!js,cgo,!gofuzz

package pqcrypto

import (
	"fmt"

	"github.com/theQRL/go-qrllib/dilithium"
)

func Sign(digestHash []byte, d *dilithium.Dilithium) ([]byte, error) {
	if len(digestHash) != DigestLength {
		return nil, fmt.Errorf("hash is required to be exactly %d bytes (%d)", DigestLength, len(digestHash))
	}
	signature, err := d.Sign(digestHash)
	return signature[:], err
}
