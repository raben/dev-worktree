dev-worktree の Homebrew パッケージを更新・インストール確認。

## フロー

### Step 1: tap 同期 & 更新
1. `git -C "$(brew --repository raben/dev-worktree)" pull origin main` で tap を最新化
2. `brew upgrade dev-worktree`

### Step 2: 確認
1. `dev --version` でバージョンを表示
