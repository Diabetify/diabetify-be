package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateVerificationCode() string {
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	return fmt.Sprintf("%06d", r.Intn(1000000))
}
