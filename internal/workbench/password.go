package workbench

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// 密码哈希：PBKDF2-HMAC-SHA256（标准库 crypto/pbkdf2，Go 1.24+）。
// 存储格式：pbkdf2$<iterations>$<salt-b64>$<hash-b64>。
// 迭代次数取 OWASP 2023 建议的 PBKDF2-HMAC-SHA256 量级下限（210000）。
const (
	pbkdf2Iterations = 210000
	pbkdf2KeyLen     = 32
	pbkdf2SaltLen    = 16
)

// HashPassword 生成 PBKDF2-HMAC-SHA256 密码哈希；盐来自 crypto/rand。
// 导出供 handler 使用；密码重置/CLI 工具后续实现时复用。
func HashPassword(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, pbkdf2KeyLen)
	if err != nil {
		return "", fmt.Errorf("derive key: %w", err)
	}
	return fmt.Sprintf("pbkdf2$%d$%s$%s", pbkdf2Iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// verifyPassword 校验密码；使用恒定时间比较，避免时序侧信道。
// 空哈希（旧版无密码账户）一律校验失败。
func verifyPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter <= 0 || iter > 10_000_000 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
