package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestCodec(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(testKey(t))
	require.NoError(t, err)

	t.Run("valid round trip", func(t *testing.T) {
		t.Parallel()
		in := Session{Subject: "s", Name: "dan", Groups: []string{"codepuke-authors"}, Expires: time.Now().Add(time.Hour).UTC()}
		token, err := codec.Seal(in)
		require.NoError(t, err)

		var out Session
		require.NoError(t, codec.Open(token, &out))
		assert.Equal(t, in.Subject, out.Subject)
		assert.Equal(t, in.Groups, out.Groups)
		assert.True(t, out.Valid())
	})

	t.Run("valid tokens differ per seal", func(t *testing.T) {
		t.Parallel()
		a, err := codec.Seal(Session{Subject: "s"})
		require.NoError(t, err)
		b, err := codec.Seal(Session{Subject: "s"})
		require.NoError(t, err)
		assert.NotEqual(t, a, b, "random nonces")
	})

	t.Run("invalid wrong key", func(t *testing.T) {
		t.Parallel()
		other, err := NewCodec(make([]byte, 32))
		require.NoError(t, err)
		token, err := other.Seal(Session{Subject: "s"})
		require.NoError(t, err)
		assert.ErrorIs(t, codec.Open(token, &Session{}), ErrBadToken)
	})

	t.Run("invalid tampered token", func(t *testing.T) {
		t.Parallel()
		token, err := codec.Seal(Session{Subject: "s"})
		require.NoError(t, err)
		head := "A"
		if strings.HasPrefix(token, "A") {
			head = "B"
		}
		assert.ErrorIs(t, codec.Open(head+token[1:], &Session{}), ErrBadToken)
	})

	t.Run("invalid garbage", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, codec.Open("not base64 !!!", &Session{}), ErrBadToken)
		assert.ErrorIs(t, codec.Open("", &Session{}), ErrBadToken)
		assert.ErrorIs(t, codec.Open("AAAA", &Session{}), ErrBadToken)
	})

	t.Run("invalid key sizes", func(t *testing.T) {
		t.Parallel()
		_, err := NewCodec(make([]byte, 16))
		assert.Error(t, err)
		_, err = NewCodec(nil)
		assert.Error(t, err)
	})
}

func TestSession(t *testing.T) {
	t.Parallel()

	t.Run("expired session is invalid", func(t *testing.T) {
		t.Parallel()
		s := Session{Expires: time.Now().Add(-time.Minute)}
		assert.False(t, s.Valid())
	})

	t.Run("group membership", func(t *testing.T) {
		t.Parallel()
		s := Session{Groups: []string{"a", "codepuke-authors"}}
		assert.True(t, s.InGroup("codepuke-authors"))
		assert.False(t, s.InGroup("other"))
		assert.False(t, Session{}.InGroup("codepuke-authors"))
	})
}

func TestSanitizeReturn(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"/admin/articles/3", "/admin/articles/3"},
		{"/", "/"},
		{"", "/admin"},
		{"https://evil.example/", "/admin"},
		{"//evil.example/", "/admin"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, sanitizeReturn(c.in), c.in)
	}
}
