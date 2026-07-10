package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAESGCMEncryptDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	plaintext := "13812346288"

	ciphertext, err := AESGCMEncrypt(key, plaintext)
	assert.NoError(t, err)
	assert.NotNil(t, ciphertext)
	assert.Greater(t, len(ciphertext), len(plaintext), "ciphertext should be longer than plaintext (nonce prepended)")

	decrypted, err := AESGCMDecrypt(key, ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestAESGCMDecrypt_WrongKey(t *testing.T) {
	key1 := []byte("0123456789abcdef0123456789abcdef")
	key2 := []byte("fedcba9876543210fedcba9876543210")
	plaintext := "13900001111"

	ciphertext, err := AESGCMEncrypt(key1, plaintext)
	assert.NoError(t, err)

	_, err = AESGCMDecrypt(key2, ciphertext)
	assert.Error(t, err, "decrypting with wrong key should fail")
}

func TestAESGCMDecrypt_ShortCiphertext(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	_, err := AESGCMDecrypt(key, []byte("short"))
	assert.Error(t, err, "ciphertext too short should error")
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"13812346288", "138****6288"},
		{"15987654321", "159****4321"},
		{"12345", "12345"},
		{"1234567", "123****"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskPhone(tt.input))
		})
	}
}

func TestHashIDCard(t *testing.T) {
	idCard := "452123199001011234"
	hash := HashIDCard(idCard)

	assert.Len(t, hash, 64, "SHA-256 should produce 64 hex chars")
	assert.NotEqual(t, idCard, hash, "hash should not equal plaintext")
	assert.Equal(t, hash, HashIDCard(idCard), "same input should produce same hash")
}
