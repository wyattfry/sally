package share

import (
	"regexp"
	"strings"
	"testing"
)

func TestSlugWordsAreCleanAndExact256(t *testing.T) {
	if len(shareSlugWords) != 256 {
		t.Fatalf("shareSlugWords must contain exactly 256 entries, got %d", len(shareSlugWords))
	}
	seen := make(map[string]bool, 256)
	wordRE := regexp.MustCompile(`^[a-z]{3,8}$`)
	for i, w := range shareSlugWords {
		if !wordRE.MatchString(w) {
			t.Errorf("shareSlugWords[%d] = %q — must be 3–8 lowercase letters, no digits/hyphens", i, w)
		}
		if seen[w] {
			t.Errorf("shareSlugWords[%d] = %q — duplicate", i, w)
		}
		seen[w] = true
	}
}

func TestNewShareSlugShape(t *testing.T) {
	slug, err := NewShareSlug()
	if err != nil {
		t.Fatalf("NewShareSlug: %v", err)
	}
	parts := strings.Split(slug, "-")
	if len(parts) != 3 {
		t.Fatalf("expected 3 hyphen-separated parts, got %d in %q", len(parts), slug)
	}
	for _, p := range parts {
		if !regexp.MustCompile(`^[a-z]{3,8}$`).MatchString(p) {
			t.Errorf("slug part %q does not match expected shape", p)
		}
	}
}

func TestNewShareSlugDifferentEachCall(t *testing.T) {
	// 16M-combo space — two adjacent calls should virtually never match.
	a, _ := NewShareSlug()
	b, _ := NewShareSlug()
	if a == b {
		t.Fatalf("two consecutive slugs were identical (%q) — RNG broken or seed reused", a)
	}
}

func TestTryNewShareSlugRetriesOnConflict(t *testing.T) {
	taken := map[string]bool{}
	first, _ := NewShareSlug()
	taken[first] = true
	calls := 0
	got, err := TryNewShareSlug(5, func(slug string) (bool, error) {
		calls++
		if taken[slug] {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("expected eventual success, got %v after %d calls", err, calls)
	}
	if got == first {
		t.Fatalf("returned slug should differ from the taken one")
	}
}
