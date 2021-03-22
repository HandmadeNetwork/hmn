package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/oops"
	"golang.org/x/crypto/pbkdf2"
)

type HashAlgorithm string

const (
	PBKDF2_SHA256 = "pbkdf2_sha256"
)

const PKBDF2KeyLength = 64

type HashedPassword struct {
	Algorithm  HashAlgorithm
	Iterations int
	Salt       string
	Hash       string
}

func ParseDjangoPasswordString(s string) (HashedPassword, error) {
	pieces := strings.SplitN(s, "$", 4)
	if len(pieces) < 4 {
		return HashedPassword{}, oops.New(nil, "unrecognized password string format")
	}

	iterations, err := strconv.Atoi(pieces[1])
	if err != nil {
		return HashedPassword{}, oops.New(err, "could not parse password iterations")
	}

	return HashedPassword{
		Algorithm:  HashAlgorithm(pieces[0]),
		Iterations: iterations,
		Salt:       pieces[2],
		Hash:       pieces[3],
	}, nil
}

func CheckPassword(password string, hashedPassword HashedPassword) (bool, error) {
	switch hashedPassword.Algorithm {
	case PBKDF2_SHA256:
		decoded, err := base64.StdEncoding.DecodeString(hashedPassword.Hash)
		if err != nil {
			return false, oops.New(nil, "failed to get key length of hashed password")
		}

		newHash := pbkdf2.Key(
			[]byte(password),
			[]byte(hashedPassword.Salt),
			hashedPassword.Iterations,
			len(decoded),
			sha256.New,
		)
		newHashEncoded := base64.StdEncoding.EncodeToString(newHash)

		return bytes.Equal([]byte(newHashEncoded), []byte(hashedPassword.Hash)), nil
	default:
		return false, oops.New(nil, "unrecognized password hash algorithm: %s", hashedPassword.Algorithm)
	}
}
