package httpx

import (
	"testing"

	"github.com/google/uuid"
)

func TestVersionTagRoundtrip(t *testing.T) {
	secret := "secreto-de-prueba"
	tag := EncodeVersionTag(secret, 7)

	version, err := DecodeVersionTag(secret, tag)
	if err != nil {
		t.Fatalf("DecodeVersionTag: %v", err)
	}
	if version != 7 {
		t.Fatalf("version = %d, esperado 7", version)
	}
}

func TestVersionTagRejectsTampering(t *testing.T) {
	tag := EncodeVersionTag("secreto", 3)
	if _, err := DecodeVersionTag("otro-secreto", tag); err == nil {
		t.Fatal("DecodeVersionTag aceptó una firma ajena")
	}
	if _, err := DecodeVersionTag("secreto", tag+"x"); err == nil {
		t.Fatal("DecodeVersionTag aceptó un tag corrupto")
	}
}

func TestParseIfMatch(t *testing.T) {
	cases := []struct {
		header string
		want   string
		ok     bool
	}{
		{`"abc"`, "abc", true},
		{`W/"abc"`, "abc", true},
		{` abc `, "abc", true},
		{"", "", false},
		{"*", "", false},
		{`"a", "b"`, "", false},
	}
	for _, tc := range cases {
		got, ok := ParseIfMatch(tc.header)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("ParseIfMatch(%q) = (%q,%v), esperado (%q,%v)", tc.header, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCursorRoundtrip(t *testing.T) {
	secret := "secreto-de-prueba"
	actor := uuid.New()
	filters := FilterHash("juan")
	cursor := EncodeCursor(secret, actor, "users.list", filters, "interno-123")

	plain, err := DecodeCursor(secret, cursor, actor, "users.list", filters)
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if plain != "interno-123" {
		t.Fatalf("plain = %q, esperado interno-123", plain)
	}
}

func TestCursorBoundToActorAndFilters(t *testing.T) {
	secret := "secreto-de-prueba"
	actor := uuid.New()
	cursor := EncodeCursor(secret, actor, "users.list", FilterHash("juan"), "interno")

	if _, err := DecodeCursor(secret, cursor, uuid.New(), "users.list", FilterHash("juan")); err == nil {
		t.Fatal("DecodeCursor aceptó un cursor de otro actor")
	}
	if _, err := DecodeCursor(secret, cursor, actor, "users.list", FilterHash("pedro")); err == nil {
		t.Fatal("DecodeCursor aceptó un cursor de otros filtros")
	}
	if _, err := DecodeCursor(secret, cursor, actor, "users.me.sessions", FilterHash("juan")); err == nil {
		t.Fatal("DecodeCursor aceptó un cursor de otro ámbito")
	}
}

func TestDecodeCursorEmpty(t *testing.T) {
	plain, err := DecodeCursor("secreto", "", uuid.New(), "scope", "filtros")
	if err != nil || plain != "" {
		t.Fatalf("DecodeCursor vacío = (%q,%v), esperado (\"\",nil)", plain, err)
	}
}
