package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
)

// Zeroize securely zeros a byte buffer, preventing compiler optimisation.
// Uses runtime.KeepAlive to ensure the write is not elided by the optimiser
// as "dead store" (the buffer is never read after being written).
//
//go:noinline — must NOT be inlined: an inlined Zeroize could have its write
// loop eliminated by dead-code elimination after inlining.
//
// Callers MUST defer Zeroize(buf) immediately after allocating or receiving
// a buffer containing secret material.
func Zeroize(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
	runtime.KeepAlive(buf)
}

// DeriveKey derives a 32-byte AES key from a passphrase and salt using
// PBKDF2-HMAC-SHA256 with 10,000 iterations.
//
// RESERVED, not yet wired: the current file store uses a random master key
// with no passphrase (see the package doc threat model). This helper exists
// for the planned passphrase-protected / native-backend mode (roadmap 1.10);
// GenerateSalt is its companion. Do not assume the store derives keys today.
func DeriveKey(passphrase string, salt []byte) []byte {
	iterations := 10000
	keyLen := 32
	return pbkdf2HMACSHA256([]byte(passphrase), salt, iterations, keyLen)
}

// pbkdf2HMACSHA256 implements PBKDF2-HMAC-SHA256 without external dependencies.
func pbkdf2HMACSHA256(password, salt []byte, iter, keyLen int) []byte {
	hashLen := 32 // sha256.Size
	numBlocks := int(math.Ceil(float64(keyLen) / float64(hashLen)))

	dk := make([]byte, 0, numBlocks*hashLen)
	block := make([]byte, hashLen)

	for blockNum := 1; blockNum <= numBlocks; blockNum++ {
		// U_1 = PRF(Password, Salt || INT_32_BE(i))
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		_ = binary.Write(mac, binary.BigEndian, uint32(blockNum))
		u := mac.Sum(nil)
		copy(block, u)

		// U_2 .. U_c, XOR each into block
		for i := 2; i <= iter; i++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range block {
				block[j] ^= u[j]
			}
		}
		dk = append(dk, block...)
	}

	return dk[:keyLen]
}

// encrypt encrypts plaintext using AES-256-GCM with a random nonce.
func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	// Prepend nonce to ciphertext
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts ciphertext that was encrypted with encrypt().
func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// GenerateSalt generates a random 16-byte salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}
