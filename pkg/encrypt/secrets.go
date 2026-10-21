// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package encrypt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"io"

	"github.com/shoenig/go-conceal"
	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters, following the OWASP / RFC 9106 "moderate" tier.
	//
	// NOTE: these make each key derivation cost roughly a second and 64 MiB of
	// RAM. The login endpoint must be rate-limited (see issue #4) or it becomes
	// a memory-exhaustion DoS vector.
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // 64 MiB
	argonThreads uint8  = 4
	keyLength    uint32 = 32
)

// saltLength is the number of random bytes in a per-user salt.
const saltLength = 16

// verifierDomain is a domain-separation constant used when computing the key
// check value.
const verifierDomain = "onlylogins-verifier"

var (
	// DummySalt is a fixed salt used to equalize the timing of failed logins,
	// so that the existence of a username is not leaked through response time.
	DummySalt = conceal.NewBytes(make([]byte, saltLength))

	// DummyVerifier is a fixed verifier that never matches a real key, used
	// alongside DummySalt to equalize the timing of failed logins.
	DummyVerifier = conceal.NewBytes(make([]byte, sha256.Size))
)

// Salt generates a random per-user salt.
func Salt() *conceal.Bytes {
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		// if crypto/rand is not working we may as well stop the process
		// and go into an outage state
		panic("encrypt: unable to read from crypto/rand")
	}
	return conceal.NewBytes(salt)
}

// DeriveKey derives the 32-byte encryption key from the passphrase and salt
// using Argon2id.
func DeriveKey(passphrase *conceal.Text, salt *conceal.Bytes) *conceal.Bytes {
	key := argon2.IDKey(
		[]byte(passphrase.Unveil()),
		salt.Unveil(),
		argonTime,
		argonMemory,
		argonThreads,
		keyLength,
	)
	return conceal.NewBytes(key)
}

// MakeVerifier produces a key check value for the given key. The verifier is
// stored alongside the salt and used to confirm a passphrase is correct
// without revealing the key itself.
func MakeVerifier(key *conceal.Bytes) *conceal.Bytes {
	mac := hmac.New(sha256.New, key.Unveil())
	mac.Write([]byte(verifierDomain))
	return conceal.NewBytes(mac.Sum(nil))
}

// Verify reports whether the given key matches the stored verifier.
func Verify(key, verifier *conceal.Bytes) bool {
	expected := MakeVerifier(key)
	return subtle.ConstantTimeCompare(expected.Unveil(), verifier.Unveil()) == 1
}
