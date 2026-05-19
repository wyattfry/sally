package share

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strings"
)

// NewShareSlug generates a human-friendly slug for project share links:
// three random words from shareSlugWords joined by "-" (e.g.
// "copper-rabbit-glacier"). Built for typability over unguessability —
// share links are not security-critical (they're meant to be printed on
// blueprints and read aloud). For login + API tokens that DO need to be
// unguessable, use NewToken instead.
//
// 256³ ≈ 16.7M combinations, so unique-constraint collisions on the
// share_links token_hash column are rare but possible. Callers should
// retry on conflict; see TryNewShareSlug below for a small helper.
func NewShareSlug() (string, error) {
	if len(shareSlugWords) < 8 {
		return "", errors.New("share: shareSlugWords list is too small")
	}
	parts := make([]string, 3)
	for i := range parts {
		idx, err := randIndex(len(shareSlugWords))
		if err != nil {
			return "", err
		}
		parts[i] = shareSlugWords[idx]
	}
	return strings.Join(parts, "-"), nil
}

// TryNewShareSlug generates a new slug, retrying up to attempts times if
// inUse reports that the slug is taken. attempts of 0 falls back to 5.
// Returns the first slug that inUse reports as free, or an error if every
// attempt collided (which should be effectively impossible with 16M
// combos and modest share-link counts, but we surface it rather than
// loop forever).
func TryNewShareSlug(attempts int, inUse func(slug string) (bool, error)) (string, error) {
	if attempts <= 0 {
		attempts = 5
	}
	for i := 0; i < attempts; i++ {
		slug, err := NewShareSlug()
		if err != nil {
			return "", err
		}
		taken, err := inUse(slug)
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
	}
	return "", errors.New("share: could not find an unused slug after retries")
}

// randIndex returns a uniformly random integer in [0, n) using
// crypto/rand. Avoids math/rand bias and seeding boilerplate.
func randIndex(n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("share: randIndex needs n > 0")
	}
	// Read 8 random bytes; reject-sample if we'd introduce modulo bias.
	// With n ≤ 256 and a uint64 source, bias is vanishingly small, but
	// we do the reject pattern anyway for correctness.
	max := uint64(n)
	limit := (^uint64(0) / max) * max
	var buf [8]byte
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			return 0, err
		}
		v := binary.BigEndian.Uint64(buf[:])
		if v < limit {
			return int(v % max), nil
		}
	}
}
