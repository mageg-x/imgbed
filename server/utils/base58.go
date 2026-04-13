package utils

import (
	"strings"
)

// Base58Alphabet Base58 字母表（不含 0OIl）
const Base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Base58Encode 将字符串编码为 Base58
func Base58Encode(input string) string {
	return Base58EncodeBytes([]byte(input))
}

func Base58EncodeBytes(bytes []byte) string {
	if len(bytes) == 0 {
		return ""
	}

	leadingZeros := 0
	for _, b := range bytes {
		if b == 0 {
			leadingZeros++
		} else {
			break
		}
	}

	// 转换为大整数
	var num []int
	for _, b := range bytes {
		num = append(num, int(b))
	}

	// 转换过程
	base := len(Base58Alphabet)
	result := ""
	for len(num) > 0 {
		var quotient []int
		remainder := 0
		for _, n := range num {
			digit := n + remainder*256
			q := digit / base
			r := digit % base
			if len(quotient) > 0 || q > 0 {
				quotient = append(quotient, q)
			}
			remainder = r
		}
		result = string(Base58Alphabet[remainder]) + result
		num = quotient
	}

	// 添加前导 '1'（对应零字节）
	for i := 0; i < leadingZeros; i++ {
		result = "1" + result
	}

	return result
}

// Base58Decode 将 Base58 字符串解码为原始字符串
func Base58Decode(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// 计算前导 '1' 的个数
	leadingOnes := 0
	for _, c := range input {
		if c == '1' {
			leadingOnes++
		} else {
			break
		}
	}

	// 转换为大整数
	var num []int
	for _, c := range input {
		idx := strings.Index(Base58Alphabet, string(c))
		if idx == -1 {
			return "", nil
		}
		num = append(num, idx)
	}

	// 转换过程
	result := make([]byte, 0)
	for len(num) > 0 {
		var quotient []int
		remainder := 0
		for _, n := range num {
			digit := n + remainder*58
			q := digit / 256
			r := digit % 256
			if len(quotient) > 0 || q > 0 {
				quotient = append(quotient, q)
			}
			remainder = r
		}
		result = append([]byte{byte(remainder)}, result...)
		num = quotient
	}

	// 添加前导零字节
	for i := 0; i < leadingOnes; i++ {
		result = append([]byte{0}, result...)
	}

	return string(result), nil
}
