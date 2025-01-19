package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

type HashAlgorithm string

const (
	Django_PBKDF2SHA256 HashAlgorithm = "pbkdf2_sha256"
	Argon2id            HashAlgorithm = "argon2id"
)

const saltLength = 16
const keyLength = 64

type HashedPassword struct {
	Algorithm  HashAlgorithm
	AlgoConfig string // arbitrary info describing the hash parameters (e.g. work factor)

	// To make it easier to handle varying implementations and encodings,
	// these fields will always store a form of the data that can be directly
	// stored in the database (usually base64-encoded or whatever).
	Salt string
	Hash string
}

func ParsePasswordString(s string) (HashedPassword, error) {
	pieces := strings.SplitN(s, "$", 4)
	if len(pieces) < 4 {
		return HashedPassword{}, oops.New(nil, "unrecognized password string format")
	}

	return HashedPassword{
		Algorithm:  HashAlgorithm(pieces[0]),
		AlgoConfig: pieces[1],
		Salt:       pieces[2],
		Hash:       pieces[3],
	}, nil
}

func (p HashedPassword) String() string {
	return fmt.Sprintf("%s$%s$%s$%s", p.Algorithm, p.AlgoConfig, p.Salt, p.Hash)
}

func (p HashedPassword) IsOutdated() bool {
	return p.Algorithm != Argon2id
}

type Argon2idConfig struct {
	Time      uint32
	Memory    uint32
	Threads   uint8
	KeyLength uint32
}

func ParseArgon2idConfig(cfg string) (Argon2idConfig, error) {
	parts := strings.Split(cfg, ",")

	t64, err := strconv.ParseUint(parts[0][2:], 10, 32)
	if err != nil {
		return Argon2idConfig{}, oops.New(err, "failed to parse time in Argon2id config")
	}

	m64, err := strconv.ParseUint(parts[1][2:], 10, 32)
	if err != nil {
		return Argon2idConfig{}, oops.New(err, "failed to parse memory in Argon2id config")
	}

	p64, err := strconv.ParseUint(parts[2][2:], 10, 8)
	if err != nil {
		return Argon2idConfig{}, oops.New(err, "failed to parse threads in Argon2id config")
	}

	l64, err := strconv.ParseUint(parts[3][2:], 10, 32)
	if err != nil {
		return Argon2idConfig{}, oops.New(err, "failed to parse key length in Argon2id config")
	}

	return Argon2idConfig{
		Time:      uint32(t64),
		Memory:    uint32(m64),
		Threads:   uint8(p64),
		KeyLength: uint32(l64),
	}, nil
}

func (c Argon2idConfig) String() string {
	return fmt.Sprintf("t=%v,m=%v,p=%v,l=%v", c.Time, c.Memory, c.Threads, c.KeyLength)
}

func CheckPassword(password string, hashedPassword HashedPassword) (bool, error) {
	switch hashedPassword.Algorithm {
	case Argon2id:
		cfg, err := ParseArgon2idConfig(hashedPassword.AlgoConfig)
		if err != nil {
			return false, err
		}

		salt, err := base64.StdEncoding.DecodeString(hashedPassword.Salt)
		if err != nil {
			return false, oops.New(err, "failed to decode salt")
		}

		newHash := argon2.IDKey([]byte(password), []byte(salt), cfg.Time, cfg.Memory, cfg.Threads, cfg.KeyLength)
		newHashEnc := base64.StdEncoding.EncodeToString(newHash)

		return bytes.Equal([]byte(newHashEnc), []byte(hashedPassword.Hash)), nil
	case Django_PBKDF2SHA256:
		decoded, err := base64.StdEncoding.DecodeString(hashedPassword.Hash)
		if err != nil {
			return false, oops.New(nil, "failed to get key length of hashed password")
		}

		iterations, err := strconv.Atoi(hashedPassword.AlgoConfig)
		if err != nil {
			return false, oops.New(nil, "failed to get PBKDF2 iterations")
		}

		newHash := pbkdf2.Key(
			[]byte(password),
			[]byte(hashedPassword.Salt),
			iterations,
			len(decoded),
			sha256.New,
		)
		newHashEncoded := base64.StdEncoding.EncodeToString(newHash)

		return bytes.Equal([]byte(newHashEncoded), []byte(hashedPassword.Hash)), nil
	default:
		return false, oops.New(nil, "unrecognized password hash algorithm: %s", hashedPassword.Algorithm)
	}
}

func HashPassword(password string) HashedPassword {
	// Follows the OWASP recommendations as of March 2021.
	// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html

	salt := make([]byte, saltLength)
	io.ReadFull(rand.Reader, salt)
	saltEnc := base64.StdEncoding.EncodeToString(salt)

	cfg := Argon2idConfig{
		Time:      1,
		Memory:    40 * 1024, // this is in KiB for some reason
		Threads:   1,
		KeyLength: keyLength,
	}

	key := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.Memory, cfg.Threads, cfg.KeyLength)
	keyEnc := base64.StdEncoding.EncodeToString(key)

	return HashedPassword{
		Algorithm:  Argon2id,
		AlgoConfig: cfg.String(),
		Salt:       saltEnc,
		Hash:       keyEnc,
	}
}

var ErrUserDoesNotExist = errors.New("user does not exist")

func UpdatePassword(ctx context.Context, conn db.ConnOrTx, username string, hp HashedPassword) error {
	tag, err := conn.Exec(ctx, "UPDATE hmn_user SET password = $1 WHERE username = $2", hp.String(), username)
	if err != nil {
		return oops.New(err, "failed to update password")
	} else if tag.RowsAffected() < 1 {
		return ErrUserDoesNotExist
	}

	return nil
}

func SetPassword(ctx context.Context, conn db.ConnOrTx, username string, password string) error {
	hp := HashPassword(password)
	return UpdatePassword(ctx, conn, username, hp)
}

func DeleteInactiveUsers(ctx context.Context, conn *pgxpool.Pool) (int64, error) {
	tag, err := conn.Exec(ctx,
		`
		DELETE FROM hmn_user
		WHERE
			status = $1 AND
			(SELECT COUNT(*) as ct FROM one_time_token AS ott WHERE ott.owner_id = hmn_user.id AND ott.expires < $2 AND ott.token_type = $3) > 0;
		`,
		models.UserStatusInactive,
		time.Now(),
		models.TokenTypeRegistration,
	)

	if err != nil {
		return 0, oops.New(err, "failed to delete inactive users")
	}

	return tag.RowsAffected(), nil
}

func DeleteExpiredPasswordResets(ctx context.Context, conn *pgxpool.Pool) (int64, error) {
	tag, err := conn.Exec(ctx,
		`
		DELETE FROM one_time_token
		WHERE
			token_type = $1
			AND expires < $2
		`,
		models.TokenTypePasswordReset,
		time.Now(),
	)

	if err != nil {
		return 0, oops.New(err, "failed to delete expired password resets")
	}

	return tag.RowsAffected(), nil
}

func PeriodicallyDeleteInactiveUsers(conn *pgxpool.Pool) *jobs.Job {
	job := jobs.New("periodically delete inactive users")
	go func() {
		defer job.Finish()

		t := time.NewTicker(1 * time.Hour)
		for {
			select {
			case <-t.C:
				err := func() (err error) {
					defer utils.RecoverPanicAsError(&err)

					n, err := DeleteInactiveUsers(job.Ctx, conn)
					if err == nil {
						if n > 0 {
							job.Logger.Info().Int64("num deleted users", n).Msg("Deleted inactive users")
						}
					} else {
						job.Logger.Error().Err(err).Msg("Failed to delete inactive users")
					}

					n, err = DeleteExpiredPasswordResets(job.Ctx, conn)
					if err == nil {
						if n > 0 {
							job.Logger.Info().Int64("num deleted password resets", n).Msg("Deleted expired password resets")
						}
					} else {
						job.Logger.Error().Err(err).Msg("Failed to delete expired password resets")
					}
					return nil
				}()
				if err != nil {
					job.Logger.Error().Err(err).Msg("Panicked in PeriodicallyDeleteInactiveUsers")
				}
			case <-job.Canceled():
				return
			}
		}
	}()
	return job
}
