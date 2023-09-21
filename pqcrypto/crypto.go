package pqcrypto

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/theQRL/go-qrllib/common"
	"github.com/theQRL/go-qrllib/dilithium"
)

// DigestLength sets the signature digest exact length
const DigestLength = 32

// LoadDilithium loads Dilithium from the given file having hex seed (not extended hex seed).
func LoadDilithium(file string) (*dilithium.Dilithium, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	r := bufio.NewReader(fd)
	buf := make([]byte, common.SeedSize*2)
	n, err := readASCII(buf, r)
	if err != nil {
		return nil, err
	} else if n != len(buf) {
		return nil, fmt.Errorf("key file too short, want %v hex characters", common.SeedSize*2)
	}
	if err := checkKeyFileEnd(r); err != nil {
		return nil, err
	}

	return HexToDilithium(string(buf))
}

func GenerateDilithiumKey() (*dilithium.Dilithium, error) {
	return dilithium.New()
}

// readASCII reads into 'buf', stopping when the buffer is full or
// when a non-printable control character is encountered.
func readASCII(buf []byte, r *bufio.Reader) (n int, err error) {
	for ; n < len(buf); n++ {
		buf[n], err = r.ReadByte()
		switch {
		case err == io.EOF || buf[n] < '!':
			return n, nil
		case err != nil:
			return n, err
		}
	}
	return n, nil
}

// checkKeyFileEnd skips over additional newlines at the end of a key file.
func checkKeyFileEnd(r *bufio.Reader) error {
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case b != '\n' && b != '\r':
			return fmt.Errorf("invalid character %q at end of key file", b)
		case i >= 2:
			return errors.New("key file too long, want 64 hex characters")
		}
	}
}

// ToDilithiumUnsafe blindly converts a binary blob to a private key. It should almost
// never be used unless you are sure the input is valid and want to avoid hitting
// errors due to bad origin encoding (0 prefixes cut off).
func ToDilithiumUnsafe(seed []byte) *dilithium.Dilithium {
	var sizedSeed [common.SeedSize]uint8
	copy(sizedSeed[:], seed)
	d, err := dilithium.NewDilithiumFromSeed(sizedSeed)
	if err != nil {
		return nil
	}
	return d
}

// HexToDilithium parses a hex seed (not extended hex seed).
func HexToDilithium(hexSeedStr string) (*dilithium.Dilithium, error) {
	b, err := hex.DecodeString(hexSeedStr)
	if byteErr, ok := err.(hex.InvalidByteError); ok {
		return nil, fmt.Errorf("invalid hex character %q in private key", byte(byteErr))
	} else if err != nil {
		return nil, errors.New("invalid hex data for private key")
	}

	var hexSeed [common.SeedSize]uint8
	copy(hexSeed[:], b)

	return dilithium.NewDilithiumFromSeed(hexSeed)
}

func zeroBytes(d **dilithium.Dilithium) {
	*d = nil
}
