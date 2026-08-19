// Package auth 提供安全相关的密码学能力：
// 1) AES-256-GCM：加密/解密数据库连接密码（密文存配置，密钥走环境变量）；
// 2) bcrypt：用户密码加盐哈希，避免明文存储。
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost 是 bcrypt 计算成本：开发环境 10，生产环境建议 12。
const BcryptCost = 10

// ErrInvalidKey 表示 AES 密钥格式不正确（应为 64 位十六进制，即 32 字节）。
var ErrInvalidKey = errors.New("AES 密钥必须为 64 位十六进制字符串（32 字节）")

// parseKey 将十六进制密钥串解析为 32 字节的 AES-256 密钥。
func parseKey(keyHex string) ([]byte, error) {
	if len(keyHex) != 64 {
		return nil, ErrInvalidKey
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, ErrInvalidKey
	}
	return key, nil
}

// EncryptAESGCM 使用 AES-256-GCM 加密明文。
// 输出格式：base64(iv || ciphertext || tag)，与设计文档 5.4 节一致。
func EncryptAESGCM(keyHex, plaintext string) (string, error) {
	key, err := parseKey(keyHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES 密码块失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}
	// GCM 建议使用 12 字节随机 nonce；与 iv 一同拼接到密文前
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机 nonce 失败: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAESGCM 解密 EncryptAESGCM 生成的密文。
// GCM 自带完整性校验：密文或密钥被篡改会直接返回错误，调用方必须启动失败。
func DecryptAESGCM(keyHex, ciphertextB64 string) (string, error) {
	key, err := parseKey(keyHex)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("密文 base64 解码失败: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES 密码块失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("密文长度非法")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("GCM 校验失败（密钥或密文可能被篡改）: %w", err)
	}
	return string(plaintext), nil
}

// HashPassword 使用 bcrypt 对用户密码加盐哈希。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("密码哈希失败: %w", err)
	}
	return string(hash), nil
}

// CheckPassword 恒定时间比对用户密码与 bcrypt 哈希。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
