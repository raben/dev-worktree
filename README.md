# dev

Docker サンドボックス内で Claude Code を安全に自律実行する CLI。

コンテナ内で `--dangerously-skip-permissions` を使い、複数リポジトリにまたがる並列 AI 開発を実現する。ホストは保護されたまま、Claude Code の subagent が worktree 隔離で並列作業する。

## インストール

```bash
brew install raben/tap/dev-worktree
```

### 依存

- [Docker](https://www.docker.com/)
- `ANTHROPIC_API_KEY` 環境変数

## 使い方

```bash
# 環境を起動して Claude Code セッション開始
dev

# リポジトリをマウント追加（プロジェクトディレクトリで実行）
cd ~/repos/app-x
dev add

cd ~/repos/app-y
dev add

# 再接続
dev

# 状態確認
dev status

# ブラウザで確認
dev open -p 3000

# 停止
dev stop
```

## コマンド

| コマンド | 説明 |
|---------|------|
| `dev` | コンテナ起動 + Claude Code セッション開始（再接続） |
| `dev add` | cwd の git リポジトリをコンテナにマウント |
| `dev status` | 環境状態・マウント済みリポジトリ一覧 |
| `dev open -p PORT` | ブラウザでポートを開く（デフォルト 3000） |
| `dev stop` | 環境停止・削除 |

## 仕組み

1. `dev` は Docker コンテナを起動し、Claude Code をインストールして `--dangerously-skip-permissions` で対話セッションを開始
2. `dev add` でリポジトリを `/workspace/<name>` にマウント（コンテナ再作成）
3. コンテナ内の Claude Code が subagent（`isolation: "worktree"`、`run_in_background: true`）で並列にタスクを実行
4. 人間はオーケストレーターの Claude Code と対話しながら進捗管理・介入

## フラグ

- `--safe` — `--dangerously-skip-permissions` なしで Claude Code を起動
- `-p, --port PORT` — `dev open` で開くポート番号（デフォルト 3000）

## License

MIT
