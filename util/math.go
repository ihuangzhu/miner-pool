package util

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"
	"strconv"
	"strings"
)

var (
	Ether  = math.BigPow(10, 18)
	Pow256 = math.BigPow(2, 256)
)

// 强制转换为big.Int
func MustParseBig(num string) *big.Int {
	if number, ok := new(big.Int).SetString(num, 10); ok {
		return number
	}
	return big.NewInt(0)
}

// Hex2clean 清理十六进制数字多余字符
func Hex2clean(hexStr string) string {
	if len(hexStr) == 0 {
		hexStr = "0x0"
	}

	// remove 0x suffix if found in the input string
	return strings.Replace(hexStr, "0x", "", -1)
}

// Hex2int64 将十六进制转换为十进制数字
func Hex2int64(hexStr string) int64 {
	cleaned := Hex2clean(hexStr)

	// base 16 for hexadecimal
	result, err := strconv.ParseInt(cleaned, 16, 64)
	if err != nil {
		panic(err)
	}
	return result
}

// Hex2uint64 将十六进制转换为无符号十进制数字
func Hex2uint64(hexStr string) uint64 {
	cleaned := Hex2clean(hexStr)

	// base 16 for hexadecimal
	result, err := strconv.ParseUint(cleaned, 16, 64)
	if err != nil {
		panic(err)
	}
	return result
}

// Diff2target 难度转换
func Diff2target(diff *big.Int) string {
	return fmt.Sprintf("0x%064x", new(big.Int).Div(Pow256, diff).Bytes())
}

// Target2diff 难度转换
func Target2diff(targetHex string) *big.Int {
	targetBytes, err := hexutil.Decode(targetHex)
	if err != nil {
		panic(err)
	}
	return new(big.Int).Div(Pow256, new(big.Int).SetBytes(targetBytes))
}
