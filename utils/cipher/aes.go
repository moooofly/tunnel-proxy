package cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

// base64 encode
func Base64Encode(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}

// base64 decode
func Base64Decode(src string) []byte {
	d, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return nil
	}
	return d
}

// AES-CBC-PKCS7 encrypt
func AES_CBC_PKCS7_encode(plaintext, aeskey, iv []byte) ([]byte, error) {
	if plaintext == nil || aeskey == nil || iv == nil {
		return nil, errors.New("plaintext or aeskey or iv is nil")
	}

	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	cbc := cipher.NewCBCEncrypter(block, iv)
	plaintext = PKCS7Padding(plaintext, blockSize)
	crypted := make([]byte, len(plaintext))
	cbc.CryptBlocks(crypted, plaintext)
	return crypted, nil
}

// AES-CBC-PKCS7 decrypt
func AES_CBC_PKCS7_decode(cyphertext, aeskey, iv []byte) ([]byte, error) {
	if cyphertext == nil || aeskey == nil || iv == nil {
		return nil, errors.New("cyphertext or aeskey or iv is nil")
	}

	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, err
	}

	cbc := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(cyphertext))
	cbc.CryptBlocks(decrypted, cyphertext)

	return PKCS7UnPadding(decrypted), nil
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(plaintext []byte) []byte {
	length := len(plaintext)
	unpadding := int(plaintext[length-1])
	return plaintext[:(length - unpadding)]
}
