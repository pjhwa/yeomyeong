package persist

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters (OWASP 2024 interactive recommendation):
//
//	time    = 2
//	memory  = 19 MiB (19456 KiB)
//	threads = 1
//	keyLen  = 32
//
// Salt is 16 random bytes. Stored as a PHC string. Do not add a second hash.
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

var (
	dummyOnce sync.Once
	dummyPHC  string
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return encodePHC(argonTime, argonMemory, argonThreads, salt, key), nil
}

func verifyPassword(encoded, password string) bool {
	t, mem, par, salt, key, ok := decodePHC(encoded)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, mem, par, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1
}

func dummyPasswordHash() string {
	dummyOnce.Do(func() {
		h, err := hashPassword("yeomyeong-timing-dummy")
		if err != nil {
			dummyPHC = encodePHC(argonTime, argonMemory, argonThreads, make([]byte, argonSaltLen), make([]byte, argonKeyLen))
			return
		}
		dummyPHC = h
	})
	return dummyPHC
}

func encodePHC(t, mem uint32, par uint8, salt, key []byte) string {
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, mem, t, par, b64.EncodeToString(salt), b64.EncodeToString(key))
}

func decodePHC(s string) (t, mem uint32, par uint8, salt, key []byte, ok bool) {
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return
	}
	var ver int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &ver); err != nil || ver != argon2.Version {
		return
	}
	var p int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &t, &p); err != nil || p < 1 || p > 255 {
		return
	}
	par = uint8(p)
	var err error
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil || len(salt) == 0 {
		return
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return
	}
	ok = true
	return
}
