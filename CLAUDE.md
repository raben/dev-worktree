# dev — CLAUDE.md

## 概要
Docker サンドボックス内で Claude Code を `--dangerously-skip-permissions` で実行する CLI。
コンテナ内の Claude Code が subagent で並列自律開発し、人間は監視・介入するだけ。

## アーキテクチャ
- `dev` → コンテナ起動 + Claude Code セッション開始
- `dev add` → cwd の git リポジトリをコンテナにマウント追加
- `dev status` → 環境状態表示
- `dev open` → ブラウザでポートを開く
- `dev stop` → 環境停止・削除
- セッション状態: `~/.dev/session.json`

## 開発ルール
- 変更後は `go build ./... && go vet ./...` で確認
- テスト: `go test ./... -count=1`
- リリースは `/release` スキルで実行
- スキル一覧は `.claude/commands/` を参照

## コミュニケーション
- 日本語で回答
- 簡潔に
