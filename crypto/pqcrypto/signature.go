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
	if err != nil {
		return nil, err
	}
	return signature[:], nil
}
