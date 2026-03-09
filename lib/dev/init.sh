# dev-worktree init command
# Sourced by bin/dev — do not execute directly.

INIT_PROMPT='You are setting up a devcontainer for the `dev` CLI tool (dev-worktree).

## Step 1: Project validation

First, analyze the current directory and determine:
- Is this a software development project? (look for package.json, go.mod, Cargo.toml, requirements.txt, pyproject.toml, Makefile, src/, etc.)
- What language/runtime does it use?
- What services does it depend on? (databases, caches, emulators, etc.)
- Does `.devcontainer/` already exist?

Present your findings as a brief summary and ask the user to confirm before proceeding.
Example:
  "Node.js (pnpm) + Spanner emulator. Ports: MCP 3000, Web 3001, Spanner 9010/9020. Proceed?"

If this does NOT look like a development project (no recognizable project files), tell the user and stop.
If `.devcontainer/` already exists, show what is there and ask if they want to overwrite.

## Step 2: Generate files

After user confirmation, generate:

1. `.devcontainer/Dockerfile`
2. `.devcontainer/devcontainer.json`
3. `.devcontainer/docker-compose.yml`
4. `.devcontainer/.env.example`

## Conventions (MUST follow)

### Dockerfile
- Base image: use the appropriate `mcr.microsoft.com/devcontainers/*` image
- Install: git, jq, curl
- If $DEV_EXEC_CMD contains "claude", install Claude Code: `npm install -g @anthropic-ai/claude-code`
- Install project-specific package manager if needed (pnpm, yarn, etc.)
- Non-root user (node for Node.js, vscode for others)
- WORKDIR /workspace

### devcontainer.json
- `"name"`: project directory name
- `"dockerComposeFile"`: "docker-compose.yml"
- `"service"`: "app"
- `"workspaceFolder"`: "/workspace"
- `"remoteUser"`: appropriate non-root user
- `"mounts"`: bind mount `~/.gitconfig` (readonly). If $DEV_EXEC_CMD contains "claude", also add a Docker volume for Claude config: `source=claude-code-config,target=/home/<user>/.claude,type=volume` (this persists auth across container recreations; user runs `claude login` once inside the container)
- `"containerEnv"`: if $DEV_EXEC_CMD contains "claude", set `CLAUDE_CONFIG_DIR` to `/home/<user>/.claude`
- `"postCreateCommand"`: appropriate install command (npm ci, pnpm install, pip install, etc.)

### docker-compose.yml
- Use `${WT_NAME:-<project-name>}` in all `container_name:` fields
- Use `${COMPOSE_PROJECT_NAME:-<project-name>}` or rely on docker compose default
- Use `${WT_<SERVICE>_PORT:-<default>}:<container-port>` for ALL port mappings
- App service: `command: sleep infinity` (Claude Code starts services as needed)
- App service: mount `..:/workspace:cached` and a named volume for node_modules/venv/etc.
- Include all dependent services the project needs (database, cache, emulators, etc.)

### .env.example
- Header comment explaining the format
- `WT_NAME=<project-name>`
- `COMPOSE_PROJECT_NAME=<project-name>`
- `WT_EXEC_CMD=<exec-command>` (the AI CLI command to run in `dev code`; use the value from $DEV_EXEC_CMD if set, otherwise `claude`)
- One `WT_<SERVICE>_PORT=<default-port>` line per port mapping in docker-compose.yml
- Variables ending in `_PORT` are auto-allocated by `dev up`; others are passed through

## Important
- Do NOT add unnecessary services. Only include what the project actually uses.
- Do NOT modify any existing project files. Only create files in `.devcontainer/`.
- If `.devcontainer/` already exists, ask before overwriting.'

cmd_init() {
  local exec_cmd="" ai_cmd=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --exec-cmd) exec_cmd="$2"; shift 2 ;;
      --ai-cmd)   ai_cmd="$2"; shift 2 ;;
      -h|--help)
        cat <<EOF
Usage: dev init [OPTIONS]

Generate .devcontainer/ for the current project using an AI CLI.
Defaults to Claude.

Options:
  --ai-cmd CMD     AI CLI command (default: claude)
  --exec-cmd CMD   Command for 'dev code' sessions (default: claude)
EOF
        return 0 ;;
      *) echo "Unknown option: $1" >&2; return 1 ;;
    esac
  done

  [ -z "$ai_cmd" ] && ai_cmd="claude"

  if ! command -v "${ai_cmd%% *}" &>/dev/null; then
    echo "ERROR: '${ai_cmd%% *}' not found. Install it or use --ai-cmd to specify another CLI." >&2
    return 1
  fi

  _validate_exec_cmd "$ai_cmd"

  if [ -z "$exec_cmd" ]; then
    exec_cmd="$ai_cmd"
  fi
  _validate_exec_cmd "$exec_cmd"

  echo "Generating .devcontainer/ with ${ai_cmd%% *} ..."
  exec env DEV_EXEC_CMD="$exec_cmd" $ai_cmd "$INIT_PROMPT"
}
