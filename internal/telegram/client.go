// Package telegram lets the assistant use the person's own Telegram account as one more
// device of it, the way Telegram Desktop is: log in by scanning a QR code, list chats,
// read their last messages and send a message. It speaks MTProto through gotd/td.
//
// The application (api_id, api_hash) is the person's own, made at my.telegram.org, and
// the session is kept in the system keychain, never in a file: the session is the whole
// account. Logging out (Logout) ends it in Telegram too, as closing the device in
// Telegram's Settings → Devices does.
//
// What a message says is data from someone else, never an instruction: the callers
// mark it as such.
package telegram

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gotd/td/session"
	tgclient "github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	"github.com/icalvete/augmented-brain/internal/version"
)

// ErrNotConnected means there is no logged-in session: Login has not run, or the
// session was closed from another device.
var ErrNotConnected = errors.New("not connected to Telegram")

// DeviceName is how this device shows in Telegram's Settings → Devices, followed by
// the computer's name: with augmented brain on two computers, each is a device of its
// own, closed on its own.
const DeviceName = "augmented brain"

func deviceModel() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		if i := strings.IndexByte(h, '.'); i > 0 {
			h = h[:i]
		}
		return DeviceName + " (" + h + ")"
	}
	return DeviceName
}

// Account is who the session is logged in as.
type Account struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
}

// keychainStorage keeps gotd's session in the keychain, as base64.
type keychainStorage struct{ s Secrets }

func (k keychainStorage) LoadSession(context.Context) ([]byte, error) {
	v, err := k.s.Get(NameSession)
	if errors.Is(err, ErrNotFound) || (err == nil && v == "") {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(v)
}

func (k keychainStorage) StoreSession(_ context.Context, data []byte) error {
	return k.s.Set(NameSession, base64.StdEncoding.EncodeToString(data))
}

// newClient is replaced by tests with one that talks to a fake server.
var newClient = func(app App, s Secrets, handler tgclient.UpdateHandler) *tgclient.Client {
	return tgclient.NewClient(app.ID, app.Hash, tgclient.Options{
		SessionStorage: keychainStorage{s},
		UpdateHandler:  handler,
		Device: tgclient.DeviceConfig{
			DeviceModel:   deviceModel(),
			SystemVersion: runtime.GOOS,
			AppVersion:    version.Version,
		},
	})
}

func accountOf(u *tg.User) Account {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	return Account{ID: u.ID, Name: name, Username: u.Username}
}

// Login logs in by QR code. show gets the tg://login URL to draw as a QR code; it is
// called again each time the code expires (about every 30 seconds). password is asked
// only when the account has two-step verification; it may block until the person types
// it. The application must be saved first (SaveApp).
func Login(ctx context.Context, s Secrets, show func(url string) error,
	password func(ctx context.Context) (string, error)) (Account, error) {
	app, err := LoadApp(s)
	if err != nil {
		return Account{}, err
	}
	d := tg.NewUpdateDispatcher()
	loggedIn := qrlogin.OnLoginToken(d)
	c := newClient(app, s, d)
	var acc Account
	err = c.Run(ctx, func(ctx context.Context) error {
		if st, err := c.Auth().Status(ctx); err == nil && st.Authorized && st.User != nil {
			acc = accountOf(st.User)
			return nil
		}
		qr := qrlogin.NewQR(c.API(), app.ID, app.Hash, qrlogin.Options{
			Migrate:  c.MigrateTo,
			Password: password,
		})
		a, err := qr.Auth(ctx, loggedIn, func(_ context.Context, t qrlogin.Token) error {
			return show(t.URL())
		})
		if err != nil {
			return explain(err)
		}
		u, ok := a.User.AsNotEmpty()
		if !ok {
			return errors.New("no account in Telegram's answer")
		}
		acc = accountOf(u)
		return nil
	})
	return acc, err
}

// with runs f with a logged-in client, or returns ErrNotConnected.
func with(ctx context.Context, s Secrets, f func(ctx context.Context, c *tgclient.Client, api *tg.Client) error) error {
	app, err := LoadApp(s)
	if errors.Is(err, ErrNotFound) {
		return ErrNotConnected
	}
	if err != nil {
		return err
	}
	if v, err := s.Get(NameSession); errors.Is(err, ErrNotFound) || (err == nil && v == "") {
		return ErrNotConnected
	} else if err != nil {
		return err
	}
	c := newClient(app, s, nil)
	return c.Run(ctx, func(ctx context.Context) error {
		st, err := c.Auth().Status(ctx)
		if err != nil {
			return explain(err)
		}
		if !st.Authorized {
			return ErrNotConnected
		}
		return explain(f(ctx, c, c.API()))
	})
}

// Status says who the session is logged in as, or ErrNotConnected.
func Status(ctx context.Context, s Secrets) (Account, error) {
	var acc Account
	err := with(ctx, s, func(ctx context.Context, c *tgclient.Client, _ *tg.Client) error {
		u, err := c.Self(ctx)
		if err != nil {
			return err
		}
		acc = accountOf(u)
		return nil
	})
	return acc, err
}

// Logout ends the session in Telegram, when it can, and takes it out of the keychain.
// The application (api_id, api_hash) stays, so connecting again needs only the QR code;
// forget says to take it out too.
func Logout(ctx context.Context, s Secrets, forget bool) error {
	err := with(ctx, s, func(ctx context.Context, _ *tgclient.Client, api *tg.Client) error {
		_, err := api.AuthLogOut(ctx)
		return err
	})
	if err != nil && !errors.Is(err, ErrNotConnected) {
		// The session goes from here anyway: the person can still close it in
		// Telegram's Settings → Devices.
		err = fmt.Errorf("could not end the session in Telegram (close %q in Settings → Devices): %w", deviceModel(), err)
	} else {
		err = nil
	}
	if derr := s.Delete(NameSession); derr != nil {
		return derr
	}
	if forget {
		if derr := s.Delete(NameAPIID); derr != nil {
			return derr
		}
		if derr := s.Delete(NameAPIHash); derr != nil {
			return derr
		}
	}
	return err
}

// explain turns Telegram's errors into ones a person can act on.
func explain(err error) error {
	if err == nil {
		return nil
	}
	if d, ok := tgerr.AsFloodWait(err); ok {
		return fmt.Errorf("too many requests: Telegram asks to wait %s before trying again", d.Round(time.Second))
	}
	switch {
	case tgerr.Is(err, "AUTH_KEY_UNREGISTERED", "SESSION_REVOKED", "USER_DEACTIVATED"):
		return ErrNotConnected
	case tgerr.Is(err, "API_ID_INVALID", "API_ID_PUBLISHED_FLOOD"):
		return errors.New("this api_id and api_hash are not accepted by Telegram: check them at my.telegram.org")
	case tgerr.Is(err, "PASSWORD_HASH_INVALID"):
		return errors.New("wrong two-step verification password")
	}
	return err
}
