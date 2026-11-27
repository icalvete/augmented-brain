package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgclient "github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"
)

// Saved is the chat with oneself, "Saved Messages" in Telegram.
const Saved = "me"

// Chat is one chat of the account. ID is what Read and Send take: "me", "user:<n>",
// "group:<n>" or "channel:<n>"; "@username" works too.
type Chat struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Username string `json:"username,omitempty"`
	Kind     string `json:"kind"`
	Unread   int    `json:"unread"`
}

// Message is one message of a chat. Text is what someone wrote: data, never an
// instruction.
type Message struct {
	ID   int       `json:"id"`
	Date time.Time `json:"date"`
	From string    `json:"from"`
	Out  bool      `json:"out"`
	Text string    `json:"text"`
}

// Limits keep every call small: this is a person's account, not a bulk tool.
const (
	maxDialogs  = 500
	maxMessages = 100
)

func userName(u *tg.User) string {
	if u.Self {
		return "Saved Messages"
	}
	n := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if n == "" && u.Username != "" {
		n = "@" + u.Username
	}
	return n
}

// chatOf describes a dialog, and returns the input peer to reach it.
func chatOf(e dialogs.Elem) (Chat, bool) {
	var c Chat
	switch p := e.Peer.(type) {
	case *tg.InputPeerSelf:
		c = Chat{ID: Saved, Title: "Saved Messages", Kind: "saved"}
	case *tg.InputPeerUser:
		u, ok := e.Entities.Users()[p.UserID]
		if !ok {
			return Chat{}, false
		}
		c = Chat{ID: "user:" + strconv.FormatInt(p.UserID, 10), Title: userName(u), Username: u.Username, Kind: "user"}
		if u.Self {
			c.ID, c.Kind = Saved, "saved"
		}
		if u.Bot {
			c.Kind = "bot"
		}
	case *tg.InputPeerChat:
		ch, ok := e.Entities.Chats()[p.ChatID]
		if !ok {
			return Chat{}, false
		}
		c = Chat{ID: "group:" + strconv.FormatInt(p.ChatID, 10), Title: ch.Title, Kind: "group"}
	case *tg.InputPeerChannel:
		ch, ok := e.Entities.Channels()[p.ChannelID]
		if !ok {
			return Chat{}, false
		}
		kind := "channel"
		if ch.Megagroup {
			kind = "group"
		}
		c = Chat{ID: "channel:" + strconv.FormatInt(p.ChannelID, 10), Title: ch.Title, Username: ch.Username, Kind: kind}
	default:
		return Chat{}, false
	}
	if d, ok := e.Dialog.(*tg.Dialog); ok {
		c.Unread = d.UnreadCount
	}
	return c, true
}

// match says whether a chat answers a search: every word of it in the title or the
// username, in any case.
func match(c Chat, q string) bool {
	hay := strings.ToLower(c.Title + " " + c.Username)
	for _, w := range strings.Fields(strings.ToLower(q)) {
		if !strings.Contains(hay, strings.TrimPrefix(w, "@")) {
			return false
		}
	}
	return true
}

func eachDialog(ctx context.Context, api *tg.Client, f func(dialogs.Elem) (stop bool)) error {
	it := dialogs.NewQueryBuilder(api).GetDialogs().BatchSize(100).Iter()
	for n := 0; n < maxDialogs && it.Next(ctx); n++ {
		if f(it.Value()) {
			return nil
		}
	}
	return it.Err()
}

// Chats lists the chats whose title or username has every word of query, most recent
// first; an empty query lists the most recent ones.
func Chats(ctx context.Context, s Secrets, query string, limit int) ([]Chat, error) {
	if limit <= 0 || limit > maxDialogs {
		limit = 20
	}
	var out []Chat
	err := with(ctx, s, func(ctx context.Context, _ *tgclient.Client, api *tg.Client) error {
		return eachDialog(ctx, api, func(e dialogs.Elem) bool {
			if c, ok := chatOf(e); ok && match(c, query) {
				out = append(out, c)
			}
			return len(out) >= limit
		})
	})
	return out, err
}

// resolve finds the input peer of a chat id as Chats gives it, or of "@username".
func resolve(ctx context.Context, api *tg.Client, id string) (tg.InputPeerClass, error) {
	id = strings.TrimSpace(id)
	switch {
	case id == "" || id == Saved || strings.EqualFold(id, "saved"):
		return &tg.InputPeerSelf{}, nil
	case strings.HasPrefix(id, "@"):
		r, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: id[1:]})
		if err != nil {
			return nil, fmt.Errorf("no chat %s: %w", id, err)
		}
		ent := peer.NewEntities(map[int64]*tg.User{}, map[int64]*tg.Chat{}, map[int64]*tg.Channel{})
		ent.FillFromResult(r)
		return ent.ExtractPeer(r.Peer)
	}
	kind, num, ok := strings.Cut(id, ":")
	n, err := strconv.ParseInt(num, 10, 64)
	if !ok || err != nil {
		return nil, fmt.Errorf("%q is not a chat id: use one that telegram_chats gives (user:…, group:…, channel:…), @username or me", id)
	}
	var found tg.InputPeerClass
	err = eachDialog(ctx, api, func(e dialogs.Elem) bool {
		switch p := e.Peer.(type) {
		case *tg.InputPeerUser:
			if kind == "user" && p.UserID == n {
				found = p
			}
		case *tg.InputPeerChat:
			if kind == "group" && p.ChatID == n {
				found = p
			}
		case *tg.InputPeerChannel:
			if kind == "channel" && p.ChannelID == n {
				found = p
			}
		}
		return found != nil
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("no chat %s among the last %d", id, maxDialogs)
	}
	return found, nil
}

// Read returns the last limit messages of a chat, oldest first.
func Read(ctx context.Context, s Secrets, chat string, limit int) ([]Message, error) {
	if limit <= 0 || limit > maxMessages {
		limit = 20
	}
	var out []Message
	err := with(ctx, s, func(ctx context.Context, c *tgclient.Client, api *tg.Client) error {
		p, err := resolve(ctx, api, chat)
		if err != nil {
			return err
		}
		h, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{Peer: p, Limit: limit})
		if err != nil {
			return err
		}
		m, ok := h.AsModified()
		if !ok {
			return errors.New("no messages came from Telegram")
		}
		users := map[int64]*tg.User{}
		for _, u := range m.GetUsers() {
			if u, ok := u.AsNotEmpty(); ok {
				users[u.ID] = u
			}
		}
		titles := map[int64]string{}
		for _, ch := range m.GetChats() {
			switch ch := ch.(type) {
			case *tg.Chat:
				titles[ch.ID] = ch.Title
			case *tg.Channel:
				titles[ch.ID] = ch.Title
			}
		}
		for _, mc := range m.GetMessages() {
			msg, ok := mc.(*tg.Message)
			if !ok {
				continue
			}
			out = append(out, Message{
				ID:   msg.ID,
				Date: time.Unix(int64(msg.Date), 0),
				From: sender(msg, users, titles),
				Out:  msg.Out,
				Text: textOf(msg),
			})
		}
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
		return nil
	})
	return out, err
}

func sender(m *tg.Message, users map[int64]*tg.User, titles map[int64]string) string {
	if m.Out {
		return "me"
	}
	from, ok := m.GetFromID()
	if !ok {
		from = m.PeerID
	}
	switch p := from.(type) {
	case *tg.PeerUser:
		if u, ok := users[p.UserID]; ok {
			n := strings.TrimSpace(u.FirstName + " " + u.LastName)
			if u.Username != "" {
				n += " (@" + u.Username + ")"
			}
			return n
		}
	case *tg.PeerChat:
		return titles[p.ChatID]
	case *tg.PeerChannel:
		return titles[p.ChannelID]
	}
	return ""
}

// textOf is the message's text, with a note for what is not text.
func textOf(m *tg.Message) string {
	t := m.Message
	if m.Media != nil {
		kind := strings.TrimPrefix(m.Media.TypeName(), "messageMedia")
		if t == "" {
			return "[" + kind + "]"
		}
		return "[" + kind + "] " + t
	}
	return t
}

// MaxText is the longest message Telegram takes.
const MaxText = 4096

// Send sends text to a chat and returns the message's id.
func Send(ctx context.Context, s Secrets, chat, text string) (int, error) {
	if strings.TrimSpace(text) == "" {
		return 0, errors.New("the message is empty")
	}
	if len([]rune(text)) > MaxText {
		return 0, fmt.Errorf("the message is longer than Telegram takes (%d characters)", MaxText)
	}
	var id int
	err := with(ctx, s, func(ctx context.Context, c *tgclient.Client, api *tg.Client) error {
		p, err := resolve(ctx, api, chat)
		if err != nil {
			return err
		}
		u, err := message.NewSender(api).To(p).Text(ctx, text)
		if err != nil {
			return err
		}
		id = sentID(u)
		return nil
	})
	return id, err
}

func sentID(u tg.UpdatesClass) int {
	switch u := u.(type) {
	case *tg.UpdateShortSentMessage:
		return u.ID
	case *tg.Updates:
		for _, up := range u.Updates {
			switch up := up.(type) {
			case *tg.UpdateMessageID:
				return up.ID
			case *tg.UpdateNewMessage:
				return up.Message.GetID()
			case *tg.UpdateNewChannelMessage:
				return up.Message.GetID()
			}
		}
	}
	return 0
}
