// Command augmented-brain-telegram is augmented brain's Telegram: the person's own
// account as one more device of it. It is a program of its own because Telegram's
// library triples the size of a program, and only who turns Telegram on should pay
// for that: augmented brain downloads it then (augmented-brain telegram connect, or the
// panel's Telegram page) and calls it.
//
// People use it through augmented-brain telegram connect | status | disconnect, which
// runs it with the same words. augmented brain's panel and MCP tools use its JSON
// side: every command with --json writes one JSON object per line, and an error as
// {"error": "…", "code": "…"}.
//
//	login --json            {"qr": url} each time the code changes, {"password": true}
//	                        when two-step verification needs its password (one line on
//	                        standard input), then {"account": …}
//	app --json              reads {"api_id": "…", "api_hash": "…"} on standard input
//	status --json           {"account": …} or {"connected": false}
//	chats --json [--query Q] [--limit N]
//	read --json --chat ID [--limit N] [--after MSG]
//	send --json --chat ID   the text on standard input; {"sent": id}
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/telegram"
	"github.com/icalvete/augmented-brain/internal/termqr"
	"github.com/icalvete/augmented-brain/internal/version"
)

// loginWait is how long a login waits for the QR code to be scanned.
const loginWait = 5 * time.Minute

// callWait bounds every other call: one look at Telegram.
const callWait = time.Minute

// lockWait is how long a call waits for another connection to Telegram to end.
const lockWait = 30 * time.Second

// configDir is augmented brain's configuration folder, where the lock lives; tests
// replace it.
var configDir = config.Dir

// secrets is the keychain; tests replace it.
var secrets telegram.Secrets = telegram.Keychain{}

// supported says whether this computer can keep the session; tests replace it.
var supported = telegram.Supported

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type opts struct {
	json, forget bool
	query, chat  string
	limit, after int
}

func parse(name string, args []string, errOut io.Writer) (*opts, []string, error) {
	o := &opts{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.BoolVar(&o.json, "json", false, "answer in JSON, one object per line")
	fs.BoolVar(&o.forget, "forget", false, "also take the application (api_id, api_hash) out of the keychain")
	fs.StringVar(&o.query, "query", "", "words of the chat's name or username")
	fs.StringVar(&o.chat, "chat", "", "the chat: an id from chats, @username or me")
	fs.IntVar(&o.limit, "limit", 20, "how many")
	fs.IntVar(&o.after, "after", 0, "read only the messages after the one with this id")
	var rest []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, nil, err
		}
		if fs.NArg() == 0 {
			return o, rest, nil
		}
		rest = append(rest, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func accountName(a telegram.Account) string {
	if a.Username != "" {
		return a.Name + " (@" + a.Username + ")"
	}
	return a.Name
}

// emit writes one JSON line.
func emit(out io.Writer, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintln(out, string(b))
}

func code(err error) string {
	switch {
	case errors.Is(err, telegram.ErrNotConnected):
		return "not_connected"
	case errors.Is(err, telegram.ErrNotFound):
		return "no_app"
	case errors.Is(err, telegram.ErrNoKeychain):
		return "no_keychain"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, telegram.ErrBusy):
		return "busy"
	}
	return "error"
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	l := i18n.FromEnv()
	o, rest, err := parse("augmented-brain-telegram", args, errOut)
	if err != nil || len(rest) != 1 {
		fmt.Fprintln(errOut, i18n.T(l, "usage.telegram"))
		return 2
	}
	if rest[0] == "version" {
		fmt.Fprintln(out, version.Version)
		return 0
	}
	fail := func(err error) int {
		if o.json {
			// The error in both languages: the panel shows it in the person's, the
			// assistant reads the English one.
			emit(out, map[string]string{"error": err.Error(), "error_es": i18n.Message("es", err), "code": code(err)})
		} else {
			fmt.Fprintln(errOut, i18n.Message(l, err))
		}
		return 1
	}
	if !supported() {
		return fail(fmt.Errorf("%s: %w", i18n.T(l, "telegram.unsupported"), telegram.ErrNoKeychain))
	}
	// One connection at a time: they all share the session in the keychain.
	dir, err := configDir()
	if err != nil {
		return fail(err)
	}
	unlock, err := telegram.Lock(dir, lockWait)
	if err != nil {
		return fail(err)
	}
	defer unlock()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	s := secrets
	switch rest[0] {
	case "connect":
		return connect(ctx, l, s, out, errOut)
	case "login":
		return login(ctx, s, in, out, fail)
	case "app":
		var v struct {
			ID   string `json:"api_id"`
			Hash string `json:"api_hash"`
		}
		if err := json.NewDecoder(in).Decode(&v); err != nil {
			return fail(errors.New("expected {\"api_id\": …, \"api_hash\": …} on standard input"))
		}
		if _, err := telegram.SaveApp(s, v.ID, v.Hash); err != nil {
			return fail(err)
		}
		emit(out, map[string]bool{"saved": true})
		return 0
	case "status":
		ctx, cancel := context.WithTimeout(ctx, callWait)
		defer cancel()
		a, err := telegram.Status(ctx, s)
		if errors.Is(err, telegram.ErrNotConnected) {
			if o.json {
				_, appErr := telegram.LoadApp(s)
				emit(out, map[string]bool{"connected": false, "app": appErr == nil})
			} else {
				fmt.Fprintln(out, i18n.T(l, "telegram.status.off"))
			}
			return 0
		}
		if err != nil {
			return fail(err)
		}
		if o.json {
			emit(out, map[string]any{"connected": true, "account": a})
		} else {
			fmt.Fprintln(out, i18n.T(l, "telegram.status.on", accountName(a)))
		}
		return 0
	case "disconnect":
		ctx, cancel := context.WithTimeout(ctx, callWait)
		defer cancel()
		if err := telegram.Logout(ctx, s, o.forget); err != nil {
			return fail(err)
		}
		if o.json {
			emit(out, map[string]bool{"disconnected": true})
			return 0
		}
		msg := i18n.T(l, "telegram.disconnected")
		if o.forget {
			msg += i18n.T(l, "telegram.forgot")
		}
		fmt.Fprintln(out, msg)
		return 0
	case "chats":
		ctx, cancel := context.WithTimeout(ctx, callWait)
		defer cancel()
		chats, err := telegram.Chats(ctx, s, o.query, o.limit)
		if err != nil {
			return fail(err)
		}
		if chats == nil {
			chats = []telegram.Chat{}
		}
		emit(out, map[string]any{"chats": chats})
		return 0
	case "read":
		ctx, cancel := context.WithTimeout(ctx, callWait)
		defer cancel()
		msgs, err := telegram.ReadAfter(ctx, s, o.chat, o.after, o.limit)
		if err != nil {
			return fail(err)
		}
		if msgs == nil {
			msgs = []telegram.Message{}
		}
		emit(out, map[string]any{"chat": o.chat, "messages": msgs})
		return 0
	case "send":
		text, err := io.ReadAll(io.LimitReader(in, 4*telegram.MaxText+1))
		if err != nil {
			return fail(err)
		}
		ctx, cancel := context.WithTimeout(ctx, callWait)
		defer cancel()
		id, err := telegram.Send(ctx, s, o.chat, strings.TrimRight(string(text), "\n"))
		if err != nil {
			return fail(err)
		}
		emit(out, map[string]any{"sent": id, "chat": o.chat})
		return 0
	}
	fmt.Fprintln(errOut, i18n.T(l, "usage.telegram"))
	return 2
}

// login is the panel's side of logging in: the QR code and the password question go
// out as JSON lines, the password comes in on standard input.
func login(ctx context.Context, s telegram.Secrets, in io.Reader, out io.Writer, fail func(error) int) int {
	ctx, cancel := context.WithTimeout(ctx, loginWait)
	defer cancel()
	lines := bufio.NewScanner(in)
	a, err := telegram.Login(ctx, s, func(url string) error {
		emit(out, map[string]string{"qr": url})
		return nil
	}, func(context.Context) (string, error) {
		emit(out, map[string]bool{"password": true})
		if !lines.Scan() {
			return "", i18n.Errorf("telegram.err.nopassword")
		}
		return lines.Text(), nil
	})
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("%w: %w", context.DeadlineExceeded, i18n.Errorf("telegram.timeout", loginWait))
		}
		return fail(err)
	}
	emit(out, map[string]any{"account": a})
	return 0
}

// connect is the terminal's side: it asks for the application if it is not saved yet,
// draws the QR code and asks for the password.
func connect(ctx context.Context, l string, s telegram.Secrets, out, errOut io.Writer) int {
	if _, err := telegram.LoadApp(s); err != nil {
		if !errors.Is(err, telegram.ErrNotFound) {
			fmt.Fprintln(errOut, i18n.Message(l, err))
			return 1
		}
		fmt.Fprintln(out, i18n.T(l, "telegram.app.intro"))
		fmt.Fprint(out, i18n.T(l, "telegram.ask.id"))
		id, ok := readLine(false)
		if !ok {
			fmt.Fprintln(errOut, "\n"+i18n.T(l, "telegram.noterminal"))
			return 1
		}
		fmt.Fprint(out, i18n.T(l, "telegram.ask.hash"))
		hash, ok := readLine(true)
		if !ok {
			fmt.Fprintln(errOut, "\n"+i18n.T(l, "telegram.noterminal"))
			return 1
		}
		if _, err := telegram.SaveApp(s, id, hash); err != nil {
			fmt.Fprintln(errOut, i18n.Message(l, err))
			return 1
		}
	}
	ctx, cancel := context.WithTimeout(ctx, loginWait)
	defer cancel()
	a, err := telegram.Login(ctx, s, func(url string) error {
		fmt.Fprintln(out, "\n"+i18n.T(l, "telegram.qr"))
		fmt.Fprint(out, termqr.Text(url))
		return nil
	}, func(context.Context) (string, error) {
		fmt.Fprint(out, i18n.T(l, "telegram.ask.password"))
		p, ok := readLine(true)
		if !ok {
			return "", errors.New(i18n.T(l, "telegram.noterminal"))
		}
		return p, nil
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(errOut, i18n.T(l, "telegram.timeout", loginWait))
		return 1
	}
	if err != nil {
		fmt.Fprintln(errOut, i18n.Message(l, err))
		return 1
	}
	fmt.Fprintln(out, i18n.T(l, "telegram.connected", accountName(a)))
	return 0
}
