package util

import (
	"time"
)

// MustParseDuration 将字符串转换成时段
func MustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}
