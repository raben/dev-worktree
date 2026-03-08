dev-worktree の e2e テスト。テスト用 git リポを作成し、dev up/list/open/down/prune を順次実行して検証する。

## 実行方法

各テストを順番に実行し、成功/失敗を記録する。失敗時は即座に報告してクリーンアップに進む。
最後にサマリーを出力する。

**重要:** `devcontainer up` は初回数分かかる。各 `dev up` コマンドはタイムアウト 600秒（10分）で実行すること。

**重要:** テストではインストール済みの `dev` ではなく、ローカルの開発版を使うこと。全ての `dev` コマンドは `$PROJECT_DIR/bin/dev` を直接実行する（`PROJECT_DIR` はこのリポジトリのルート）。

## 前提条件チェック

以下を確認し、満たさない場合は中止：
1. Docker Desktop が起動しているか（`docker info` が成功するか）
2. `dev --version` が動作するか（PATH にあるか）
3. `devcontainer --version` が動作するか

## 前回テスト残骸のクリーンアップ

テスト開始前に、前回のテスト残骸を削除する：

```bash
# 残ったコンテナを強制削除
for key in test-new-branch test-existing main; do
  docker ps -a --filter "label=dev-worktree=dev-worktree-test/$key" -q | xargs docker rm -f 2>/dev/null || true
done

# worktree prune（リポが残っている場合のみ）
cd /tmp/dev-worktree-test 2>/dev/null && git worktree prune 2>/dev/null || true

# テストリポとワークツリーを削除
rm -rf /tmp/dev-worktree-test
rm -rf /tmp/dev/dev-worktree-test
```

## テスト環境セットアップ

`/tmp/dev-worktree-test` にテスト用 git リポを作成する：

```bash
mkdir -p /tmp/dev-worktree-test
cd /tmp/dev-worktree-test
git init
git commit --allow-empty -m "initial"
mkdir -p .devcontainer
```

`.devcontainer/devcontainer.json` を作成（最小構成）：
```json
{
  "name": "dev-worktree-test",
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu"
}
```

`.devcontainer/.env.example` を作成：
```
WT_NAME=dev-worktree-test
COMPOSE_PROJECT_NAME=dev-worktree-test
WT_EXEC_CMD=echo ok
WT_APP_PORT=13100
```

作成したファイルをコミット：
```bash
git add -A
git commit -m "add devcontainer"
```

## テスト 1: dev up（新規ブランチ）

1. 実行: `cd /tmp/dev-worktree-test && dev up test-new-branch`（タイムアウト 600秒）
2. 検証:
   - exit code が 0
   - ワークツリーディレクトリ `/tmp/dev/dev-worktree-test/test-new-branch` が存在する
   - git branch `dev/dev-worktree-test/test-new-branch` が作成されている（`git -C /tmp/dev-worktree-test branch --list 'dev/dev-worktree-test/test-new-branch'` が非空）
   - コンテナが起動している（`docker ps --filter "label=dev-worktree=dev-worktree-test/test-new-branch" --filter "status=running" -q` が非空）
   - `.devcontainer/.env` がワークツリー側（`/tmp/dev/dev-worktree-test/test-new-branch/.devcontainer/.env`）に存在し、`WT_APP_PORT=` を含む

## テスト 2: dev up（既存ブランチ）

v0.9.2 で発生したバグの再発防止。既存ブランチに対して `-b` なしで worktree add されるか。

1. 準備: `git -C /tmp/dev-worktree-test branch dev/dev-worktree-test/test-existing`
2. 実行: `cd /tmp/dev-worktree-test && dev up test-existing`（タイムアウト 600秒）
3. 検証:
   - exit code が 0
   - コンテナが起動している
   - ワークツリーが正しく作成されている（`git -C /tmp/dev-worktree-test worktree list` に `test-existing` が含まれる）

## テスト 3: dev up（保護ブランチ: main）

main を指定すると自動で dev/dev-worktree-test/main ブランチが作成されることを検証。

1. 実行: `cd /tmp/dev-worktree-test && dev up main`（タイムアウト 600秒）
2. 検証:
   - exit code が 0
   - 出力に "Protected branch 'main'" が含まれる
   - コンテナが起動している（`docker ps --filter "label=dev-worktree=dev-worktree-test/main" --filter "status=running" -q` が非空）
   - ワークツリーディレクトリ `/tmp/dev/dev-worktree-test/main` が存在する

## テスト 4: dev up 冪等性（既存環境の再利用）

`dev up` を同じブランチに対して2回実行した場合、既存環境が再利用されることを検証。

1. 実行: `cd /tmp/dev-worktree-test && dev up test-new-branch`（タイムアウト 600秒）
2. 検証:
   - exit code が 0
   - 出力に "already exists" が含まれる
   - コンテナが起動している

## テスト 5: dev up 冪等性（保護ブランチの再利用）

`dev up main` を2回実行した場合、既存環境が再利用されることを検証。

1. 実行: `cd /tmp/dev-worktree-test && dev up main`（タイムアウト 600秒）
2. 検証:
   - exit code が 0
   - 出力に "Protected branch" が **含まれない**（直接 resume されるため）
   - 出力に "already exists" が含まれる

## テスト 6: dev list

1. 実行: `dev list`
2. 検証:
   - exit code が 0
   - 出力に `dev-worktree-test/test-new-branch` が含まれる
   - 出力に `dev-worktree-test/test-existing` が含まれる
   - 出力に `dev-worktree-test/main` が含まれる
   - 3つとも status が `running` と表示されている

## テスト 7: dev open（名前指定）

v0.9.4 で発生したバグの再発防止。`--id-label` が正しく渡されているか。

1. 実行: `cd /tmp/dev-worktree-test && dev open test-new-branch`
2. 検証:
   - exit code が 0
   - 出力に "ok" が含まれる（`WT_EXEC_CMD=echo ok` が実行されている証拠）

## テスト 8: dev down

1. 実行: `cd /tmp/dev-worktree-test && dev down test-new-branch`
2. 検証:
   - exit code が 0
   - コンテナが stopped 状態になっている（`docker ps --filter "label=dev-worktree=dev-worktree-test/test-new-branch" --filter "status=running" -q` が空）
   - コンテナ自体はまだ存在する（`docker ps -a --filter "label=dev-worktree=dev-worktree-test/test-new-branch" -q` が非空）

## テスト 9: dev up（停止後の再起動）

`dev down` 後に `dev up` で再起動できることを検証。

1. 実行: `cd /tmp/dev-worktree-test && dev up test-new-branch`（タイムアウト 600秒）
2. 検証:
   - exit code が 0
   - コンテナが running 状態に復帰している

## テスト 10: dev prune

devcontainer.json の .name 書き換えにより未コミット変更の警告が出るため、確認プロンプトに2回 `y` を送る（`printf "y\ny\n"`）。

1. test-existing を prune: `cd /tmp/dev-worktree-test && printf "y\ny\n" | dev prune test-existing`
2. 検証:
   - exit code が 0
   - コンテナが削除されている（`docker ps -a --filter "label=dev-worktree=dev-worktree-test/test-existing" -q` が空）
   - ワークツリーディレクトリ `/tmp/dev/dev-worktree-test/test-existing` が存在しない
   - ブランチが削除されている（`git -C /tmp/dev-worktree-test branch --list 'dev/dev-worktree-test/test-existing'` が空）
3. test-new-branch を prune: `cd /tmp/dev-worktree-test && printf "y\ny\n" | dev prune test-new-branch`
4. 検証: 同上（ディレクトリは `/tmp/dev/dev-worktree-test/test-new-branch`、ブランチは `dev/dev-worktree-test/test-new-branch`）
5. main を prune: `cd /tmp/dev-worktree-test && printf "y\ny\n" | dev prune main`
6. 検証: 同上（ディレクトリは `/tmp/dev/dev-worktree-test/main`、ブランチは `dev/dev-worktree-test/main`）
7. 空ディレクトリの掃除を検証:
   - `/tmp/dev/dev-worktree-test` が存在しない（他の worktree がなければ削除される）
   - `/tmp/dev` が存在しない（他のプロジェクトがなければ削除される）

## クリーンアップ

```bash
# 残ったコンテナを強制削除
for key in test-new-branch test-existing main; do
  docker ps -a --filter "label=dev-worktree=dev-worktree-test/$key" -q | xargs docker rm -f 2>/dev/null || true
done

# worktree prune（リポが残っている場合のみ）
cd /tmp/dev-worktree-test 2>/dev/null && git worktree prune 2>/dev/null || true

# テストリポとワークツリーを削除
rm -rf /tmp/dev-worktree-test
rm -rf /tmp/dev/dev-worktree-test
```

## サマリー出力

全テスト完了後、以下の形式でサマリーを出力：

```
=== dev-worktree e2e test results ===
Test  1 (dev up - new branch):           PASS/FAIL
Test  2 (dev up - existing branch):      PASS/FAIL
Test  3 (dev up - protected branch):     PASS/FAIL
Test  4 (dev up - idempotent):           PASS/FAIL
Test  5 (dev up - protected idempotent): PASS/FAIL
Test  6 (dev list):                      PASS/FAIL
Test  7 (dev open - named):             PASS/FAIL
Test  8 (dev down):                      PASS/FAIL
Test  9 (dev up - restart after down):   PASS/FAIL
Test 10 (dev prune):                     PASS/FAIL
=========================================
```

失敗したテストがある場合は、エラー内容と再現手順を付記する。

## 失敗時の修正フロー

テストが失敗した場合、以下の手順で修正を進める：

1. 失敗原因を分析し、`bin/dev` の修正案を提示する
2. ユーザーの承認を得て修正を実施
3. `bash -n bin/dev` で構文チェック
4. **失敗したテストのみ再実行して修正を検証**（テスト環境は保持されているため、全テストの再実行は不要）
5. 修正が確認できたら、残りのテストを続行
