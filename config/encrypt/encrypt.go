package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encrypter 是配置加密器接口
type Encrypter interface {
	// Encrypt 加密数据
	Encrypt(data []byte) ([]byte, error)
	// Decrypt 解密数据
	Decrypt(data []byte) ([]byte, error)
}

// Option 是加密器选项函数
type Option func(*options)

// options 是加密器选项
type options struct {
	key []byte
}

// WithKey 设置加密密钥
func WithKey(key []byte) Option {
	return func(o *options) {
		o.key = key
	}
}

// aesEncrypter 是基于AES的加密器实现
type aesEncrypter struct {
	opts options
}

// NewAESEncrypter 创建一个基于AES的加密器
func NewAESEncrypter(opts ...Option) (Encrypter, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	if len(o.key) == 0 {
		return nil, errors.New("encryption key is required")
	}

	// 确保密钥长度为16, 24或32字节（AES-128, AES-192, AES-256）
	if len(o.key) != 16 && len(o.key) != 24 && len(o.key) != 32 {
		return nil, errors.New("invalid key size: must be 16, 24, or 32 bytes")
	}

	return &aesEncrypter{opts: o}, nil
}

// Encrypt 使用AES-GCM加密数据
func (e *aesEncrypter) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.opts.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// Decrypt 使用AES-GCM解密数据
func (e *aesEncrypter) Decrypt(data []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(e.opts.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Source 是加密配置源包装器
type Source struct {
	source    interface{} // 原始配置源
	encrypter Encrypter   // 加密器
}

// NewSource 创建一个加密配置源
func NewSource(source interface{}, encrypter Encrypter) *Source {
	return &Source{
		source:    source,
		encrypter: encrypter,
	}
}

// GetSource 获取原始配置源
func (s *Source) GetSource() interface{} {
	return s.source
}

// GetEncrypter 获取加密器
func (s *Source) GetEncrypter() Encrypter {
	return s.encrypter
}
