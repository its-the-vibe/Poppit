## Poppit integration

### 1) How to submit commands

Push a JSON notification onto the Redis list (default `poppit:notifications`) using `RPUSH`.

Expected payload shape:

```json name=notification.json
{
  "repo": "<your repo>",
  "branch": "refs/heads/main",
  "type": "<your command type>",
  "dir": "/tmp",
  "commands": ["echo hello", "echo world"],
  "metadata": {
    "taskId": "task-12345",
    "userId": "user-456",
    "source": "<your source>"
  }
}
```

- `repo`, `branch`, `type`, `dir`, `commands` are used by execution flow.
- `metadata` is optional, but important if you want command output events.

---

### 2) How to receive command output

Poppit publishes per-command output to a Redis **Pub/Sub channel** (default `poppit:command-output`) via `publishCommandOutput(...)`.

Subscribe with:

Output message format:

```json name=command-output.json
{
  "metadata": {
    "taskId": "task-12345",
    "userId": "user-456",
    "source": "<your source>"
  },
  "type": "git-webhook",
  "command": "echo hello",
  "output": "hello\n",
  "stderr": "",
  "status_code": 0
}
```

Notes:
- `output` = stdout
- `stderr` = stderr
- `status_code` = command exit code (`0` success, non-zero failure)
- metadata is echoed back when provided in input

---

### 4) Relevant env vars

- `POPPIT_SERVICE_REDIS_LIST_NAME` (default `poppit:notifications`)
- `POPPIT_SERVICE_COMMAND_OUTPUT_CHANNEL` (default `poppit:command-output`)

---

**Sources**  
- README: https://github.com/its-the-vibe/Poppit/blob/main/README.md  
- Code (`Notification`, `CommandOutput`, `publishCommandOutput`): https://github.com/its-the-vibe/Poppit/blob/main/main.go  
- Env defaults: https://github.com/its-the-vibe/Poppit/blob/main/.env.example
