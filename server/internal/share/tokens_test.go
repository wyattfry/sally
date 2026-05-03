package share

import "testing"

func TestNewTokenReturnsOpaqueHighEntropyToken(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if len(token) < 40 {
		t.Fatalf("expected long opaque token, got %q", token)
	}
}

func TestHashTokenIsStableAndDoesNotExposeToken(t *testing.T) {
	token := "share_test_token"

	first := HashToken(token)
	second := HashToken(token)

	if first != second {
		t.Fatalf("expected stable hash")
	}
	if first == token || first == "" {
		t.Fatalf("expected non-empty hash distinct from token")
	}
}
