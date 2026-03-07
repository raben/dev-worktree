# dev-worktree

ワンコマンドで「ブランチごとの独立開発環境」を立ち上げるCLIツール。

git worktree + devcontainer + AI coding CLI を組み合わせて、複数ブランチの並行開発を実現する。

## インストール

```bash
brew tap raben/dev-worktree
brew install dev-worktree
```

<details>
<summary>手動インストール</summary>

```bash
git clone https://github.com/raben/dev-worktree.git
cd dev-worktree
./install.sh
```

</details>

### 依存

| ツール | 用途 | インストール |
|--------|------|-------------|
| [jq](https://jqlang.github.io/jq/) | ポート管理 | Homebrew が自動インストール |
| [devcontainer CLI](https://github.com/devcontainers/cli) | コンテナ管理 | `npm install -g @devcontainers/cli` |
| [Docker](https://www.docker.com/) | コンテナ実行 | Docker Desktop など |
| [tmux](https://github.com/tmux/tmux) | ダッシュボード（`dev open`） | `brew install tmux` |

AI CLI（いずれか）:

| CLI | インストール |
|-----|-------------|
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (デフォルト) | `npm install -g @anthropic-ai/claude-code` |
| その他（Aider, Codex 等） | `--ai-cmd` / `WT_EXEC_CMD` で指定 |

## 使い方

### 1. プロジェクトに `.devcontainer/` を用意する

```bash
cd your-project
dev init
```

AI CLI がプロジェクトを分析し、`.devcontainer/` を対話的に生成する。

```bash
# Claude 以外の AI CLI を使う場合
dev init --ai-cmd codex
```

### 2. ブランチごとの開発環境を立ち上げる

```bash
dev up feature-auth
```

以下が自動実行される:

1. ポート割り当て（複数環境でも衝突しない）
2. `git worktree add` でブランチ作成
3. `.devcontainer/` をコピー＆ポート設定を反映
4. `devcontainer up` でコンテナ起動

```bash
# 名前省略で自動生成
dev up

# 別ターミナルで2つ目も同時起動
dev up feature-billing

# 既存の worktree は自動で再利用（コンテナが停止していれば再起動）
dev up feature-auth

# main/master/develop を指定すると、自動でブランチを作成（2回目以降は再利用）
dev up main          # → dev-main ブランチを main から作成
dev up develop       # → dev-develop ブランチを develop から作成
```

### 3. AI セッションを開く

```bash
# tmux ダッシュボード（全環境を分割表示、新環境を自動検知）
dev open

# 単一環境に直接接続
dev open feature-auth
```

### 4. 環境を停止・削除する

```bash
# 停止のみ（worktree・ポートは保持）
dev down feature-auth

# 完全削除（コンテナ・worktree・ブランチ・ポート）
dev prune feature-auth
```

## コマンド

| コマンド | 説明 |
|---------|------|
| `dev init` | `.devcontainer/` を対話的に生成 |
| `dev up [name] [base]` | worktree 作成 → コンテナ起動（base デフォルト: `main`） |
| `dev open [name]` | AI セッションを開く（tmux ダッシュボード） |
| `dev down [name]` | コンテナ停止（worktree は保持） |
| `dev prune [name]` | コンテナ・worktree・ブランチを完全削除 |
| `dev list` | 稼働中の worktree 一覧 |

各コマンドの詳細は `dev <command> --help` で確認できる。

## ライフサイクル

環境の状態遷移と、各コマンドが何を操作するか：

```
dev up        dev down      dev prune
 作成 ──────→ 停止 ──────→ 完全削除
   ↑            │
   └────────────┘
     dev up（再起動）
```

| 操作 | コンテナ | worktree ディレクトリ | git ブランチ | ポート割り当て |
|------|---------|---------------------|-------------|-------------|
| `dev up` (新規) | 作成・起動 | 作成 | 作成 | 割り当て |
| `dev up` (既存) | 再起動 | そのまま | そのまま | そのまま |
| `dev down` | **停止** | そのまま | そのまま | そのまま |
| `dev prune` | **削除** | **削除** | **削除（merged時）** | 解放 |

- `dev down` はコンテナを停止するだけ。worktree 内のファイルやブランチは一切変更しない。`dev up` で再起動できる
- `dev prune` はコンテナ・worktree ディレクトリ・ブランチを全て削除する。未マージのブランチは安全のため保持される
- `dev prune` は常に確認プロンプトを表示する。未コミットの変更がある場合は2回確認する
- メインリポジトリのファイルには一切影響しない

## ポート管理

`.devcontainer/.env.example` で定義したポートが自動管理される。

```ini
# .devcontainer/.env.example
WT_NAME=myapp
COMPOSE_PROJECT_NAME=myapp
WT_EXEC_CMD=claude --dangerously-skip-permissions
WT_API_PORT=3000
WT_WEB_PORT=3001
WT_DB_PORT=5432
```

- `_PORT` で終わる変数は `dev up` 時に `lsof` で空きポートを検出して自動割り当て
- `WT_EXEC_CMD` は `dev open` で実行されるコマンド
- それ以外の変数はそのまま渡される
- 割り当て結果は worktree の `.devcontainer/.env` に保存される

## ディレクトリ構造

```
your-project/              # メインリポジトリ
your-project-feature-auth/ # dev up で作られる worktree
your-project-feature-billing/
```

worktree はメインリポジトリと同階層に `<project>-<name>` の形で作られる。

## ブランチ命名規則

`dev up` で作成される git ブランチは `dev/<project>/<name>` の形式になる。

```bash
dev up feature-auth
# → ブランチ: dev/your-project/feature-auth
# → worktree: your-project-feature-auth/
```

- `git branch` で `dev/` 配下に全環境がまとまる
- `dev/your-project/*` でプロジェクト別にフィルタできる
- ツール内部の dev ID（`your-project/feature-auth`）とは独立

## License

MIT
