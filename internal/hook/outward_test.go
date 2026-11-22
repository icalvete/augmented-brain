package hook

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/state"
)

func TestOutward(t *testing.T) {
	yes := []string{
		"git push origin HEAD",
		"git -C /home/u/repo push -u origin main",
		"cd /x && git push",
		"gh pr merge 12 --auto --rebase",
		"gh pr create --title x",
		"gh workflow run release -R o/r --ref main",
		"glab mr merge 4",
		"aws cloudformation deploy --stack-name s --template-file t.yml",
		"aws --profile PRO lambda update-function-code --function-name f",
		"AWS_PROFILE=PRO aws ecs update-service --cluster c",
		"aws s3 cp f s3://b/f",
		"./launch-event-bus.sh -e PRO -r eu-west-1",
		"scripts/deploy.sh",
		"kubectl apply -f x.yaml",
		"docker push repo/img:1",
		"terraform apply",
		"curl -s -X POST https://api.example.com/x",
		"curl -XDELETE https://api.example.com/x",
		"echo ok; npm publish",
	}
	no := []string{
		"git status", "git -C /r log --oneline", "git pull", "gh pr view 12", "gh pr checks 12",
		"aws s3 ls", "aws cloudformation describe-stacks", "aws --profile PRO ec2 describe-instances",
		"curl -s https://example.com", "kubectl get pods", "docker build .", "ls deploy/",
		"cat launch-event-bus.sh", "go test ./...", "echo git push",
	}
	for _, c := range yes {
		if outward(c) == "" {
			t.Errorf("not seen as outward: %s", c)
		}
	}
	for _, c := range no {
		if got := outward(c); got != "" {
			t.Errorf("seen as outward: %s (%s)", c, got)
		}
	}
	if got := outward("aws lambda update-function-code " + strings.Repeat("x", 200)); len([]rune(got)) > 81 {
		t.Errorf("not cut short: %d", len(got))
	}
}

// A command that changes something outside the machine gets the journal reminder at the
// end of that same answer, not 20 minutes later, and also in a session with no task
// active; a write in a task since then, or a reminder already given, stops it.
func TestStopAfterOutward(t *testing.T) {
	cfgDir, work := setupSpace(t)
	start := time.Date(2026, 11, 19, 14, 0, 0, 0, time.Local)
	now := clock(t, start)
	id := openTask(t, cfgDir)
	touch(t, cfgDir, "s1", id, start)
	if err := state.Started(cfgDir, "work", "s2", "", "", start); err != nil {
		t.Fatal(err)
	}
	bash := func(sid, command string) {
		run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
			map[string]any{"cwd": work, "hook_event_name": "PreToolUse", "session_id": sid,
				"tool_name": "Bash", "tool_input": map[string]any{"command": command}})
	}
	stop := func(sid string) string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) },
			map[string]any{"cwd": work, "hook_event_name": "Stop", "session_id": sid})
		if o == nil {
			return ""
		}
		return o["additionalContext"].(string)
	}
	*now = start.Add(2 * time.Minute)
	bash("s1", "ls -la")
	if got := stop("s1"); got != "" {
		t.Errorf("reminded after a harmless command: %s", got)
	}
	bash("s1", "./launch-platform.sh -e PRO")
	got := stop("s1")
	if !strings.Contains(got, "./launch-platform.sh -e PRO") || !strings.Contains(got, id) || !strings.Contains(got, "14:02") {
		t.Errorf("no reminder right after the deploy: %q", got)
	}
	if got := stop("s1"); got != "" {
		t.Errorf("reminded twice for one deploy: %s", got)
	}
	*now = start.Add(3 * time.Minute)
	bash("s1", "git push origin main")
	touch(t, cfgDir, "s1", id, start.Add(4*time.Minute)) // the session wrote in its task
	*now = start.Add(5 * time.Minute)
	if got := stop("s1"); got != "" {
		t.Errorf("reminded although the task was written after the push: %s", got)
	}
	// A command the guard refuses changed nothing: no reminder.
	bash("s2", "gh pr merge 7 --auto --rebase")
	if got := stop("s2"); got != "" {
		t.Errorf("reminded after a refused command: %s", got)
	}
	// A session with no task active is asked to find the task it belongs to.
	bash("s2", "aws s3 cp report.csv s3://shop-reports/")
	if got := stop("s2"); !strings.Contains(got, "aws s3 cp report.csv") || !strings.Contains(got, "task_new") {
		t.Errorf("session with no task: %q", got)
	}
}
