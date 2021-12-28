package util

import "regexp"

// IsZeroHash 是否是零的十六进制
func IsZeroHash(s string) bool {
	return regexp.MustCompile("^0?x?0+$").MatchString(s)
}

// IsValidHexAddress 是否是无效钱包地址
func IsValidHexAddress(s string) bool {
	if IsZeroHash(s) || !regexp.MustCompile("^0x[0-9a-fA-F]{40}$").MatchString(s) {
		return false
	}
	return true
}
