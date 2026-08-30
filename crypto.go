package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	defaultIterations = 600000
	minIterations     = 1000
	maxIterations     = 10000000
	saltSize          = 16
	nonceSize         = 12
	tagSize           = 16
)

// formatMagic 是新密文格式的前缀。
// 新格式: "AES1" + 4字节大端迭代次数 + salt + nonce + ciphertext，整体 Base64 编码。
// 旧格式(无前缀): salt + nonce + ciphertext，固定按 600000 次迭代解密，保持向后兼容。
var formatMagic = []byte("AES1")

func encrypt(plaintext []byte, password string, iterations int) ([]byte, error) {
	// 生成随机salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// 从密码派生密钥
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// 创建AES分组加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建GCM模式加密器
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 创建随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// 按顺序拼接: magic + iterations + salt + nonce + ciphertext
	out := make([]byte, 0, len(formatMagic)+4+len(salt)+len(nonce)+len(ciphertext))
	out = append(out, formatMagic...)
	out = binary.BigEndian.AppendUint32(out, uint32(iterations))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	// 使用标准Base64编码
	return []byte(base64.StdEncoding.EncodeToString(out)), nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	// 去掉粘贴内容可能携带的空白和BOM
	s := strings.TrimSpace(string(data))
	s = strings.TrimPrefix(s, "\ufeff")

	// 解码Base64
	fullCiphertext, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	// 识别格式并提取迭代次数
	iterations := defaultIterations
	body := fullCiphertext
	if len(fullCiphertext) >= len(formatMagic)+4 && string(fullCiphertext[:len(formatMagic)]) == string(formatMagic) {
		iterations = int(binary.BigEndian.Uint32(fullCiphertext[len(formatMagic) : len(formatMagic)+4]))
		body = fullCiphertext[len(formatMagic)+4:]
		if iterations < 1 || iterations > maxIterations {
			return nil, errors.New("invalid iteration count in ciphertext")
		}
	}

	// 检查长度是否合法
	if len(body) < saltSize+nonceSize+tagSize {
		return nil, errors.New("invalid ciphertext")
	}

	// 提取salt, nonce和ciphertext
	salt := body[:saltSize]
	nonce := body[saltSize : saltSize+nonceSize]
	ciphertext := body[saltSize+nonceSize:]

	// 派生密钥
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// 创建解密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 尝试解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("incorrect password or corrupted data")
	}

	return plaintext, nil
}
