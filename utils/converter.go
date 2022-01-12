package utils

import (
	"strconv"
)

func String2Uint(str string) uint {
	i, e := strconv.Atoi(str)
	if e != nil {
		return 0
	}
	return uint(i)
}
