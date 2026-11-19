#!/usr/bin/env python3
"""Exercise local Codex with scripted models, temporary homes and real AB.

Build AB first, then run: python3 -I scripts/codex-check.py /path/to/augmented-brain
Add --cross-client to exercise the task handoff with Claude Code CLI.
The shell probe is a fixed, harmless command. The test records dispatch to the
shell even if the host's sandbox prevents the command from starting.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys
import tempfile
import threading


MODEL = "gpt-5.4"
CLAUDE_MODEL = "claude-sonnet-4-6"
CONTEXT = "AB_TEST_HOOK_CONTEXT"
TASK = "Check the sample reports"
NEXT = "Compare the two sample reports"
ID = "task-check-the-sample-reports"


class ModelHandler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def do_POST(self):
        model = self.server
        request = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        model.requests.append(request)
        index = len(model.requests)
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()

        if self.path.startswith("/v1/messages"):
            self.anthropic_turn(index, *model.call)
        else:
            self.openai_turn(index, *model.call)

    def emit(self, event, **fields):
        payload = json.dumps({"type": event, **fields})
        self.wfile.write(f"event: {event}\ndata: {payload}\n\n".encode())

    def openai_turn(self, index, name, arguments):
        response = {
            "id": f"resp_probe_{index}", "object": "response", "created_at": 0,
            "model": MODEL, "status": "in_progress", "output": [],
        }
        self.emit("response.created", response=response)
        if index == 1:
            namespace, _, name = name.rpartition(".")
            item = {
                "id": "fc_probe", "type": "function_call", "name": name,
                "call_id": "call_probe", "arguments": json.dumps(arguments),
                "status": "completed",
            }
            if namespace:
                item["namespace"] = namespace
        else:
            item = {
                "id": "msg_probe", "type": "message", "role": "assistant",
                "content": [{"type": "output_text", "text": "Probe finished.", "annotations": []}],
                "status": "completed",
            }
        self.emit("response.output_item.added", output_index=0, item=item)
        self.emit("response.output_item.done", output_index=0, item=item)
        response["output"] = [item]
        response["status"] = "completed"
        self.emit("response.completed", response=response)


    def anthropic_turn(self, index, name, arguments):
        self.emit("message_start", message={
            "id": f"msg_probe_{index}", "type": "message", "role": "assistant",
            "content": [], "model": CLAUDE_MODEL, "stop_reason": None, "stop_sequence": None,
            "usage": {"input_tokens": 10, "output_tokens": 0},
        })
        if index == 1:
            block = {"type": "tool_use", "id": "toolu_probe", "name": name, "input": {}}
            delta = {"type": "input_json_delta", "partial_json": json.dumps(arguments)}
            reason = "tool_use"
        else:
            block = {"type": "text", "text": ""}
            delta = {"type": "text_delta", "text": "Probe finished."}
            reason = "end_turn"
        self.emit("content_block_start", index=0, content_block=block)
        self.emit("content_block_delta", index=0, delta=delta)
        self.emit("content_block_stop", index=0)
        self.emit("message_delta", delta={"stop_reason": reason, "stop_sequence": None},
                  usage={"output_tokens": 10})
        self.emit("message_stop")


@contextmanager
def scripted_model(call):
    with ThreadingHTTPServer(("127.0.0.1", 0), ModelHandler) as model:
        model.requests = []
        model.call = call
        server = threading.Thread(target=model.serve_forever, daemon=True)
        server.start()
        try:
            yield model
        finally:
            model.shutdown()
            server.join()


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def codex_turn(codex, root, work, env, call, hooks, mcp=None, trust=True):
    root.mkdir()
    (root / "hooks.json").write_text(json.dumps({"hooks": hooks}), encoding="utf-8")
    with scripted_model(call) as model:
        config = f"""
model = "{MODEL}"
model_provider = "probe"
approval_policy = "never"
sandbox_mode = "read-only"

[model_providers.probe]
name = "Local fixture"
base_url = "http://127.0.0.1:{model.server_port}/v1"
wire_api = "responses"
request_max_retries = 0
stream_max_retries = 0
"""
        if mcp:
            config += f"""
[mcp_servers.augmented-brain]
command = {json.dumps(str(mcp))}
args = ["mcp"]
required = true
enabled_tools = ["where_am_i", "pending", "task_new", "task_get", "task_park", "task_journal", "task_update", "task_step", "task_section"]
default_tools_approval_mode = "approve"

[mcp_servers.augmented-brain.env]
AB_CONFIG_DIR = {json.dumps(env["AB_CONFIG_DIR"])}
AB_MCP_TASKS_ONLY = "1"
"""
        (root / "config.toml").write_text(config, encoding="utf-8")
        command = [codex, "exec", "--json", "--skip-git-repo-check"]
        if trust:
            # This process loads only fixture hooks from its temporary CODEX_HOME.
            command.append("--dangerously-bypass-hook-trust")
        command.append("Run the scripted probe.")
        # stdin closed: codex exec reads more of the prompt from an open stdin, and waits.
        result = subprocess.run(command, cwd=work, env={**env, "CODEX_HOME": str(root)},
                                stdin=subprocess.DEVNULL, text=True, capture_output=True, timeout=25)
        require(result.returncode == 0 and len(model.requests) == 2,
                f"Codex turn failed ({result.returncode}): {result.stderr[-2000:]}\n{result.stdout[-2000:]}")
        outputs = [item["output"] for item in model.requests[1]["input"]
                   if item.get("type") == "function_call_output" and item.get("call_id") == "call_probe"]
        require(len(outputs) == 1, f"Expected one result for {call[0]}, got {len(outputs)}")
        return json.dumps(model.requests[0]["input"], ensure_ascii=False), json.dumps(outputs[0], ensure_ascii=False)


def claude_turn(claude, ab, root, work, env, step, arguments):
    root.mkdir()
    mcp = root / "mcp.json"
    mcp.write_text(json.dumps({"mcpServers": {
        "augmented_brain": {"command": str(ab), "args": ["mcp"]},
    }}), encoding="utf-8")
    tool = f"mcp__augmented_brain__task_{step}"
    with scripted_model((tool, arguments)) as model:
        result = subprocess.run(
            [claude, "--bare", "--print", "--model", CLAUDE_MODEL, "--strict-mcp-config",
             "--mcp-config", str(mcp), "--tools", "", "--allowedTools", tool, "--",
             "Run the scripted probe."],
            cwd=work,
            env={**env, "CLAUDE_CONFIG_DIR": str(root),
                 "ANTHROPIC_BASE_URL": f"http://127.0.0.1:{model.server_port}",
                 "ANTHROPIC_API_KEY": "fixture-key", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"},
            stdin=subprocess.DEVNULL, text=True, capture_output=True, timeout=25,
        )
        require(result.returncode == 0 and len(model.requests) == 2,
                f"Claude turn failed ({result.returncode}): {result.stderr[-2000:]}")
        output = [item for turn in model.requests[1]["messages"]
                  if isinstance(turn["content"], list)
                  for item in turn["content"]
                  if item.get("type") == "tool_result" and item.get("tool_use_id") == "toolu_probe"]
        require(len(output) == 1 and not output[0].get("is_error"),
                f"Claude task_{step} failed: {output}")
        return json.dumps(output[0]["content"], ensure_ascii=False)


def hook(command, timeout=5):
    return [{"hooks": [{"type": "command", "command": shlex.join(command), "timeout": timeout}]}]


def codex_start(ab):
    """The nine SessionStart hooks of the Codex start, one part each."""
    return {"SessionStart": [{"hooks": [
        {"type": "command", "timeout": 10,
         "command": shlex.join(["env", "AB_CODEX_START_HOOK=1", str(ab), "hook", "session-start", str(n)])}
        for n in range(1, 10)]}]}


# A rule in Spanish long enough that its part, within Claude's budget in characters,
# is over Codex's cut in bytes: its last line must still arrive.
RULE_END = "FIN_DE_LA_REGLA_LARGA"


def write_long_rule(notes):
    body = "Año tras año, señoría: «cañón», ¿mañana? — ñoño. " * 170  # 8,330 characters, 10,710 bytes
    (notes / "rules" / "rule-larga.md").write_text(
        f"---\nid: rule-larga\ntype: rule\ntitle: Regla larga\nsummary: despacio\n---\n{body}\n{RULE_END}\n",
        encoding="utf-8")


def check_memory(ab, codex, claude, root, work, env):
    hooks = codex_start(ab)
    write_long_rule(root / "notes")
    for step, args, expected in (
        ("new", {"title": TASK}, TASK),
        ("park", {"id": ID, "next": NEXT}, "parked"),
        ("update", {"id": ID, "status": "plan", "next": NEXT}, "plan"),
        ("section", {"id": ID, "section": "plan", "text": "PLAN_FROM_CODEX"}, "Plan"),
        ("step", {"id": ID, "add": "STEP_FROM_CODEX"}, "STEP_FROM_CODEX"),
        ("get", {"id": ID}, "PLAN_FROM_CODEX"),
    ):
        if claude and step == "new":
            output = claude_turn(claude, ab, root / "claude-new", work, env, step, args)
            require(expected in output, f"Claude task_new did not return {expected!r}: {output[:1200]}")
            continue
        initial, output = codex_turn(codex, root / step, work, env,
                                     (f"mcp__augmented_brain.task_{step}", args), hooks, mcp=ab)
        require(expected in output, f"task_{step} did not return {expected!r}: {output[:1200]}")
        if step == "get":
            require(TASK in initial and NEXT in initial, "The task did not reach a new session's context")
            require("## In Codex" in initial and "command guard runs here only if" in initial,
                    "Codex did not receive the note on what differs there")
            require(RULE_END in initial and "truncated" not in initial,
                    "A part of the Codex start was cut: the long rule did not arrive whole")
            require("start_session" not in initial, "The Codex start carries what only Claude Code has")
    note = (root / "notes" / "tasks" / f"{ID}.md").read_text(encoding="utf-8")
    require("status: plan" in note and "- [ ] STEP_FROM_CODEX" in note and "PLAN_FROM_CODEX" in note,
            "The shared note does not hold what Codex wrote")
    if claude:
        output = claude_turn(claude, ab, root / "claude-get", work, env, "get", {"id": ID})
        require("STEP_FROM_CODEX" in output and NEXT in output, "Claude did not read the task as Codex left it")
        print("OK  Claude opened the task, Codex parked, planned and stepped it, and both read it in fresh sessions")
    else:
        print("OK  Codex opened, parked, planned, stepped and read a task in fresh sessions, with the whole start in nine parts")


def check_spaces(ab, codex, root, first_work, env):
    second_work = root / "second-work"
    subprocess.run(
        [str(ab), "space", "add", "--yes", "--space", "second", "--work", str(second_work),
         "--memory", str(root / "second-notes")],
        cwd=root, env=env, check=True, capture_output=True, text=True, timeout=20,
    )
    spaces = [("lab", first_work), ("second", second_work), ("lab", root / "notes" / "tasks")] * 2

    def locate(indexed_space):
        index, (name, work) = indexed_space
        _, output = codex_turn(codex, root / f"space-{index}", work, env,
                               ("mcp__augmented_brain.where_am_i", {}), {}, mcp=ab)
        require(f"space: {name}" in output, f"Wrong space for {name}: {output[:1200]}")

    with ThreadPoolExecutor(max_workers=len(spaces)) as pool:
        list(pool.map(locate, enumerate(spaces)))
    _, output = codex_turn(codex, root / "outside", root, env,
                           ("mcp__augmented_brain.where_am_i", {}), {}, mcp=ab)
    require("no space" in output.lower(), f"Unexpected answer outside the spaces: {output[:1200]}")
    _, output = codex_turn(codex, root / "other-space", first_work, env,
                           ("mcp__augmented_brain.pending", {"space": "second"}), {}, mcp=ab)
    require("current space" in output, f"A call naming another space was not refused: {output[:1200]}")
    print("OK  Six concurrent Codex sessions resolved their spaces, notes folder included; another space and an unassigned folder were refused")


def check_hooks(codex, root, work, env):
    fixture = root / "fixture.py"
    fixture.write_text(
        """import json
import sys
import time

case, path = sys.argv[1:]
data = json.load(sys.stdin)
event = data["hook_event_name"]
with open(path, "a", encoding="utf-8") as log:
    print(event, file=log)
if event == "SessionStart":
    fields = {"additionalContext": "AB_TEST_HOOK_CONTEXT"}
elif event == "PreToolUse" and case in {"deny", "ask"}:
    fields = {"permissionDecision": case, "permissionDecisionReason": "AB fixture policy"}
elif event == "PreToolUse" and case == "error":
    sys.exit(1)
elif event == "PreToolUse" and case == "malformed":
    print("{invalid")
    sys.exit(0)
elif event == "PreToolUse" and case == "timeout":
    time.sleep(3)
    sys.exit(0)
else:
    sys.exit(0)
print(json.dumps({"hookSpecificOutput": {"hookEventName": event, **fields}}))
""",
        encoding="utf-8",
    )

    for case in ("allow", "deny", "ask", "error", "malformed", "timeout", "untrusted"):
        events = root / f"{case}.log"
        hooks = {event: hook([sys.executable, str(fixture), case, str(events)], timeout=1)
                 for event in ("SessionStart", "PreToolUse", "PostToolUse", "SessionEnd")}
        initial, output = codex_turn(codex, root / case, work, env, ("exec_command", {"cmd": "printf ab-hook-test"}),
                                     hooks, trust=case != "untrusted")
        observed = events.read_text(encoding="utf-8").splitlines() if events.exists() else []
        blocked = "AB fixture policy" in output and "blocked" in output.lower()
        if case == "untrusted":
            require(not observed and CONTEXT not in initial, f"Unexpected untrusted hook activity: {observed}")
        else:
            expected = ["SessionStart", "PreToolUse"]
            if case != "deny":
                expected.append("PostToolUse")
            expected.append("SessionEnd")
            require(observed == expected and CONTEXT in initial,
                    f"Unexpected hook events for {case}: {observed}")
        require(blocked == (case == "deny"), f"Codex changed how it handles {case}: {output[:1200]}")
        print(f"OK  {case}: " + ("blocked" if blocked else "not blocked by the hook"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("augmented_brain", type=Path)
    parser.add_argument("--cross-client", action="store_true", help="hand off a task between both CLIs")
    args = parser.parse_args()
    ab = args.augmented_brain.resolve()
    codex = shutil.which("codex")
    require(codex and ab.is_file(), "Need Codex CLI on PATH and a built augmented-brain binary")
    claude = shutil.which("claude") if args.cross_client else None
    require(not args.cross_client or claude, "--cross-client needs Claude Code on PATH")
    with tempfile.TemporaryDirectory(prefix="ab-codex-check-") as temp:
        root = Path(temp)
        home = root / "home"
        home.mkdir()
        runtime = root / "runtime"
        runtime.mkdir(mode=0o700)
        work = root / "work"
        env = {
            "PATH": os.environ["PATH"], "HOME": str(home), "XDG_RUNTIME_DIR": str(runtime),
            "AB_CONFIG_DIR": str(root / "config"), "AB_NO_UPDATE_CHECK": "1", "AB_NO_AUTO_BACKUP": "1",
        }
        subprocess.run([str(ab), "init", "--yes", "--no-plugin", "--language", "en",
                        "--space", "lab", "--work", str(work), "--memory", str(root / "notes")],
                       cwd=root, env=env, check=True, capture_output=True, text=True, timeout=20)
        check_memory(ab, codex, claude, root, work, env)
        check_spaces(ab, codex, root, work, env)
        check_hooks(codex, root, work, env)


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.SubprocessError) as err:
        print(f"FAIL  {err}", file=sys.stderr)
        sys.exit(1)
