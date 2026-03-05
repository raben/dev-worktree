# dev-worktree

ワンコマンドで「ブランチごとの独立開発環境」を立ち上げるCLIツール。

git worktree + devcontainer + Claude Code を組み合わせて、複数ブランチの並行開発を実現する。

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

| ツール | インストール |
|--------|-------------|
| [jq](https://jqlang.github.io/jq/) | Homebrew が自動インストール |
| [devcontainer CLI](https://github.com/devcontainers/cli) | `npm install -g @devcontainers/cli` |
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code) | `npm install -g @anthropic-ai/claude-code` |
| [Docker](https://www.docker.com/) | Docker Desktop など |

## 使い方

### 1. プロジェクトに `.devcontainer/` を用意する

```bash
cd your-project
dev init
```

Claude がプロジェクトを分析し、`.devcontainer/` を対話的に生成する。

### 2. ブランチごとの開発環境を立ち上げる

```bash
dev up feature-auth
```

以下が自動実行される:

1. ポート割り当て（複数環境でも衝突しない）
2. `git worktree add` でブランチ作成
3. `.devcontainer/` をコピー＆ポート設定を反映
4. `devcontainer up` でコンテナ起動
5. コンテナ内で Claude Code セッションを起動

```bash
# 別ターミナルで2つ目の環境も同時に起動できる
dev up feature-billing
```

### 3. 環境を停止・削除する

```bash
dev down feature-auth
```

コンテナ停止 → worktree 削除 → ポート解放を行う。未コミットの変更がある場合は確認される。

## コマンド

| コマンド | 説明 |
|---------|------|
| `dev init` | `.devcontainer/` を対話的に生成 |
| `dev up <name> [base]` | worktree 作成 → コンテナ起動 → Claude 起動 |
| `dev down <name>` | コンテナ停止 → worktree 削除 |
| `dev list` | 稼働中の worktree 一覧 |
| `dev ports` | ポート割り当て管理 |

各コマンドの詳細は `dev <command> --help` で確認できる。

## ポート管理

`.devcontainer/.env.example` で定義したポートが自動管理される。

```ini
# .devcontainer/.env.example
WT_NAME=myapp
COMPOSE_PROJECT_NAME=myapp
WT_API_PORT=3000
WT_WEB_PORT=3001
WT_DB_PORT=5432
```

- `_PORT` で終わる変数は `dev up` 時に空きポートが自動割り当てされる
- それ以外の変数はそのまま渡される
- 割り当て状態は `~/.dev-worktree/port-registry.json` でグローバル管理

## ディレクトリ構造

```
your-project/              # メインリポジトリ
your-project-feature-auth/ # dev up で作られる worktree
your-project-feature-billing/
```

worktree はメインリポジトリと同階層に `<project>-<name>` の形で作られる。

## License

MIT
