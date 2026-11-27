package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// memSecrets is a keychain in memory.
type memSecrets map[string]string

func (m memSecrets) Get(name string) (string, error) {
	v, ok := m[name]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
func (m memSecrets) Set(name, value string) error { m[name] = value; return nil }
func (m memSecrets) Delete(name string) error     { delete(m, name); return nil }

func TestParseApp(t *testing.T) {
	good := "0123456789abcdef0123456789ABCDEF"
	app, err := ParseApp(" 12345 ", good+"\n")
	if err != nil || app.ID != 12345 || app.Hash != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("ParseApp = %+v, %v", app, err)
	}
	for _, c := range [][2]string{
		{"", good}, {"abc", good}, {"-1", good}, {"1", ""}, {"1", good[:31]}, {"1", good[:31] + "g"},
	} {
		if _, err := ParseApp(c[0], c[1]); err == nil {
			t.Errorf("ParseApp(%q, %q) took it", c[0], c[1])
		}
	}
}

func TestSaveAndLoadApp(t *testing.T) {
	s := memSecrets{}
	if _, err := LoadApp(s); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty keychain: %v", err)
	}
	if _, err := SaveApp(s, "x", "y"); err == nil || len(s) != 0 {
		t.Fatalf("a bad application was saved: %v %v", err, s)
	}
	if _, err := SaveApp(s, "42", "0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	app, err := LoadApp(s)
	if err != nil || app.ID != 42 {
		t.Fatalf("LoadApp = %+v, %v", app, err)
	}
}

// The session goes to the keychain as base64, and a missing one is gotd's ErrNotFound.
func TestKeychainStorage(t *testing.T) {
	s := memSecrets{}
	k := keychainStorage{s}
	ctx := context.Background()
	if _, err := k.LoadSession(ctx); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("no session: %v", err)
	}
	data := []byte{0, 1, 2, 255, '\n'}
	if err := k.StoreSession(ctx, data); err != nil {
		t.Fatal(err)
	}
	if v := s[NameSession]; v != "AAEC/wo=" {
		t.Fatalf("stored %q", v)
	}
	got, err := k.LoadSession(ctx)
	if err != nil || string(got) != string(data) {
		t.Fatalf("LoadSession = %v, %v", got, err)
	}
}

// Without the application or the session, nothing tries to reach Telegram.
func TestNotConnected(t *testing.T) {
	ctx := context.Background()
	s := memSecrets{}
	if _, err := Status(ctx, s); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("no application: %v", err)
	}
	if _, err := SaveApp(s, "42", "0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if _, err := Chats(ctx, s, "", 5); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("no session: %v", err)
	}
	if _, err := Send(ctx, s, Saved, "hi"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("no session: %v", err)
	}
	// Logging out without a session still clears what is there, and forget takes the
	// application too.
	s[NameSession] = ""
	if err := Logout(ctx, s, true); err != nil {
		t.Fatal(err)
	}
	if len(s) != 0 {
		t.Fatalf("left %v", s)
	}
}

func TestSendRefusesBeforeConnecting(t *testing.T) {
	s := memSecrets{}
	if _, err := Send(context.Background(), s, Saved, "  "); err == nil {
		t.Fatal("an empty message was taken")
	}
	long := make([]rune, MaxText+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := Send(context.Background(), s, Saved, string(long)); err == nil {
		t.Fatal("a message too long was taken")
	}
}

func TestChatOfAndMatch(t *testing.T) {
	ent := peer.NewEntities(
		map[int64]*tg.User{
			7: {ID: 7, FirstName: "Lucía", LastName: "Pérez", Username: "lucia_p"},
			8: {ID: 8, FirstName: "Bot", Bot: true},
			9: {ID: 9, FirstName: "Me", Self: true},
		},
		map[int64]*tg.Chat{3: {ID: 3, Title: "Family"}},
		map[int64]*tg.Channel{4: {ID: 4, Title: "Club news", Username: "clubnews"}, 5: {ID: 5, Title: "Team", Megagroup: true}},
	)
	cases := []struct {
		peer tg.InputPeerClass
		want Chat
	}{
		{&tg.InputPeerUser{UserID: 7}, Chat{ID: "user:7", Title: "Lucía Pérez", Username: "lucia_p", Kind: "user", Unread: 2}},
		{&tg.InputPeerUser{UserID: 8}, Chat{ID: "user:8", Title: "Bot", Kind: "bot", Unread: 2}},
		{&tg.InputPeerUser{UserID: 9}, Chat{ID: Saved, Title: "Saved Messages", Kind: "saved", Unread: 2}},
		{&tg.InputPeerChat{ChatID: 3}, Chat{ID: "group:3", Title: "Family", Kind: "group", Unread: 2}},
		{&tg.InputPeerChannel{ChannelID: 4}, Chat{ID: "channel:4", Title: "Club news", Username: "clubnews", Kind: "channel", Unread: 2}},
		{&tg.InputPeerChannel{ChannelID: 5}, Chat{ID: "channel:5", Title: "Team", Kind: "group", Unread: 2}},
	}
	for _, c := range cases {
		got, ok := chatOf(dialogs.Elem{Peer: c.peer, Entities: ent, Dialog: &tg.Dialog{UnreadCount: 2}})
		if !ok || got != c.want {
			t.Errorf("chatOf(%T) = %+v, %v; want %+v", c.peer, got, ok, c.want)
		}
	}
	if _, ok := chatOf(dialogs.Elem{Peer: &tg.InputPeerUser{UserID: 99}, Entities: ent}); ok {
		t.Error("a user Telegram did not describe was taken")
	}
	lucia := cases[0].want
	for q, want := range map[string]bool{"": true, "lucía": true, "LUCÍA pérez": true, "@lucia_p": true, "lucía gómez": false} {
		if match(lucia, q) != want {
			t.Errorf("match(%q) != %v", q, want)
		}
	}
}

func TestTextOfAndSentID(t *testing.T) {
	if got := textOf(&tg.Message{Message: "hola"}); got != "hola" {
		t.Errorf("text: %q", got)
	}
	if got := textOf(&tg.Message{Media: &tg.MessageMediaPhoto{}}); got != "[Photo]" {
		t.Errorf("photo: %q", got)
	}
	if got := textOf(&tg.Message{Message: "look", Media: &tg.MessageMediaDocument{}}); got != "[Document] look" {
		t.Errorf("document: %q", got)
	}
	if id := sentID(&tg.UpdateShortSentMessage{ID: 5}); id != 5 {
		t.Errorf("short: %d", id)
	}
	if id := sentID(&tg.Updates{Updates: []tg.UpdateClass{&tg.UpdateMessageID{ID: 6}}}); id != 6 {
		t.Errorf("updates: %d", id)
	}
}

func TestExplain(t *testing.T) {
	if explain(nil) != nil {
		t.Fatal("nil")
	}
	if err := explain(tgerr.New(401, "AUTH_KEY_UNREGISTERED")); !errors.Is(err, ErrNotConnected) {
		t.Errorf("revoked session: %v", err)
	}
	if err := explain(tgerr.New(420, "FLOOD_WAIT_30")); err == nil || err.Error() != "too many requests: Telegram asks to wait 30s before trying again" {
		t.Errorf("flood: %v", err)
	}
}
