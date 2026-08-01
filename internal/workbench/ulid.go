package workbench

import (
	"crypto/rand"
	"time"
)

// crockford Crockford Base32 字母表（不含 I L O U，避免歧义）。
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// newULID 生成符合 ULID 规范的 26 字符标识符。
// 结构：48 位毫秒时间戳 + 80 位 crypto/rand 随机数，编码为 Crockford Base32。
// 并发安全：crypto/rand.Read 内部持锁；时间戳从 time.Now 读取无共享状态。
func newULID() string {
	var buf [16]byte
	ms := uint64(time.Now().UnixMilli())
	buf[0] = byte(ms >> 40)
	buf[1] = byte(ms >> 32)
	buf[2] = byte(ms >> 24)
	buf[3] = byte(ms >> 16)
	buf[4] = byte(ms >> 8)
	buf[5] = byte(ms)
	_, _ = rand.Read(buf[6:])
	return encodeCrockford(buf[:])
}

// encodeCrockford 将 16 字节编码为 26 字符 Crockford Base32。
// 128 位 / 5 = 25.6，故 26 字符（末字符仅 2 位有效，左移补零）。
func encodeCrockford(data []byte) string {
	result := make([]byte, 26)
	var bits uint64
	var nbits uint
	idx := 0
	for _, b := range data {
		bits = (bits << 8) | uint64(b)
		nbits += 8
		for nbits >= 5 {
			nbits -= 5
			result[idx] = crockford[(bits>>nbits)&0x1F]
			idx++
		}
	}
	if nbits > 0 {
		result[idx] = crockford[(bits<<(5-nbits))&0x1F]
	}
	return string(result)
}
