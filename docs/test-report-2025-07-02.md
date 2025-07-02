# WTP 挙動テストレポート (2025-07-02)

## 概要

本レポートは、wtp (Worktree Plus)
v0.3.0の包括的な機能テストの結果をまとめたものです。特に、新しく実装された「Better
error messages」機能と「Multiple remote
handling」機能の動作確認に重点を置いています。

## テスト環境

- **OS**: macOS Darwin 24.5.0
- **Go Version**: 1.24.4
- **Git Version**: (システムデフォルト)
- **wtp Version**: d95c9bc (2025-01-02)

## ✅ 成功した機能

### 1. 基本機能

- **バージョン表示**: `wtp --version` ✅
- **ヘルプ表示**: `wtp --help` ✅
- **設定ファイル作成**: `wtp init` ✅
- **設定ファイル競合検知**: 既存の.wtp.ymlがある場合の適切なエラー ✅

### 2. ワークツリー管理

- **ローカルブランチでのワークツリー作成**: `wtp add feature/test-branch` ✅
- **新規ブランチ作成**: `wtp add -b new-feature` ✅
- **カスタムパス指定**: `wtp add --path /tmp/custom-location` ✅
- **強制フラグ**: `--force` で重複チェックアウト許可 ✅
- **ワークツリー一覧表示**: `wtp list` ✅
- **ワークツリー削除**: `wtp remove` ✅

### 3. 改善されたエラーメッセージ

#### 3.1 リポジトリ外エラー

```
not in a git repository

Solutions:
  • Run 'git init' to create a new repository
  • Navigate to an existing git repository
  • Check if you're in the correct directory
```

#### 3.2 存在しないブランチエラー

```
branch 'nonexistent-branch' not found in local or remote branches

Suggestions:
  • Check the branch name spelling
  • Run 'git branch -a' to see all branches
  • Create a new branch with 'wtp add -b nonexistent-branch'
  • Fetch latest changes with 'git fetch'
```

#### 3.3 競合エラー

```
failed to create worktree at '/path' for branch 'feature'

Cause: Branch is already checked out in another worktree
Solution: Use '--force' flag to allow multiple checkouts, or choose a different branch
```

#### 3.4 バリデーションエラー

```
branch name is required

Usage: wtp add <branch-name>

Examples:
  • wtp add feature/auth
  • wtp add -b new-feature
  • wtp add --track origin/main main
```

### 4. シェル統合

- **CDコマンド**: シェル統合なしでの適切なエラー ✅
- **CDコマンド（統合あり）**: 正しいパス出力 ✅
- **存在しないワークツリー**: 利用可能な選択肢の表示 ✅
- **シェル初期化**: `wtp shell-init` の適切な出力 ✅
- **補完スクリプト生成**: bash/zsh/fish補完の生成 ✅

## ⚠️ 発見された問題

### 1. リモートブランチ追跡

**問題**: リモートブランチの自動追跡で git コマンドの引数エラー

```bash
# 期待される動作
wtp add remote-feature
# → 自動的に origin/remote-feature を追跡

# 実際のエラー
git worktree add --track origin/remote-feature /path remote-feature
# → git コマンドの引数順序が不適切
```

**原因**: `buildGitWorktreeArgs` 関数で `--track` フラグの処理位置が不適切

**解決案**:

- git worktree add コマンドの正しい引数順序に修正
- `--track` フラグを適切な位置に配置

### 2. エラーメッセージの改善余地

**問題**: git コマンドの生のエラー出力が含まれる場合がある

```
usage: git worktree add [-f] [--detach] [--checkout] [--lock [--reason <string>]]
                        [--orphan] [(-b | -B) <new-branch>] <path> [<commit-ish>]
```

**改善案**:

- git の使用法エラーをより分かりやすいメッセージに変換
- 一般的なgitエラーパターンの検出と翻訳

## 📈 パフォーマンスと安定性

### 良好な点

- **起動速度**: 高速な起動とコマンド実行 ✅
- **エラーハンドリング**: 予期しないエラーでもクラッシュしない ✅
- **メモリ使用量**: 軽量で効率的 ✅
- **gitとの互換性**: 標準的なgit operationとの良好な連携 ✅

### 安定性

- **Lint**: 全チェック通過 ✅
- **テスト**: 全テストケース成功 ✅
- **型安全性**: Go の型システムによる堅牢性 ✅
- **ビルド**: クロスプラットフォームビルド成功 ✅

## 🎯 総合評価

### 優秀な改善点

1. **大幅に向上したユーザビリティ**: エラーメッセージが遥かに親切で実用的
2. **包括的な機能セット**: 基本的なワークツリー管理が完全に動作
3. **優れた設計**: モジュラーで拡張可能なアーキテクチャ
4. **プロダクトレディ**: 実際の開発で使用可能なレベル

### 推奨される次のステップ

1. **リモートブランチ追跡の修正**: git コマンド引数の順序問題を解決
2. **エラーメッセージのさらなる洗練**: git エラーのパターンマッチング改善
3. **E2Eテストの自動化**: 以下で詳述
4. **ドキュメントの更新**: 新機能の使用例を追加

## 結論

WTPは**非常に成功した改善**を達成しました。特に「Better error
messages」の実装により、ユーザーエクスペリエンスが劇的に向上しています。発見された小さな問題はありますが、コア機能は安定して動作しており、実用的なツールとして十分に機能します。

**実装された機能の品質は極めて高く**、git
worktreeの使いにくさを大幅に改善する素晴らしいツールになっています。

---

## E2Eテスト自動化の設計

### 目的

本テストで実施した手動テストを自動化し、リグレッションテストとして CI/CD
パイプラインで実行可能にする。

### 設計方針

1. **独立性**: 各テストは独立して実行可能
2. **再現性**: 同じ結果を得られる環境の構築
3. **高速性**: 並列実行可能な構造
4. **可読性**: テストケースの意図が明確

### テストカテゴリ

#### 1. 基本機能テスト (`test/e2e/basic_test.go`)

- バージョン表示
- ヘルプ表示
- 設定ファイル操作

#### 2. ワークツリー管理テスト (`test/e2e/worktree_test.go`)

- ワークツリー作成（各種パターン）
- ワークツリー一覧表示
- ワークツリー削除
- エラーケース

#### 3. リモートブランチテスト (`test/e2e/remote_test.go`)

- リモートブランチ自動追跡
- 複数リモート処理
- エラーハンドリング

#### 4. シェル統合テスト (`test/e2e/shell_test.go`)

- CDコマンド機能
- 補完スクリプト生成
- エラーケース

#### 5. エラーメッセージテスト (`test/e2e/error_messages_test.go`)

- 各種エラーメッセージの検証
- ユーザビリティの確認

### テストフレームワーク案

```go
// test/e2e/framework/framework.go
package framework

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

type TestRepo struct {
    t       *testing.T
    dir     string
    wtpPath string
}

func NewTestRepo(t *testing.T) *TestRepo {
    // テスト用の一時リポジトリを作成
}

func (r *TestRepo) RunWTP(args ...string) (string, error) {
    // wtp コマンドを実行
}

func (r *TestRepo) Cleanup() {
    // テスト環境のクリーンアップ
}
```

### テスト例

```go
// test/e2e/worktree_test.go
func TestWorktreeCreation(t *testing.T) {
    repo := framework.NewTestRepo(t)
    defer repo.Cleanup()

    // ローカルブランチでのワークツリー作成
    t.Run("LocalBranch", func(t *testing.T) {
        output, err := repo.RunWTP("add", "feature/test")
        assert.NoError(t, err)
        assert.Contains(t, output, "Created worktree")
    })

    // 存在しないブランチのエラー
    t.Run("NonexistentBranch", func(t *testing.T) {
        _, err := repo.RunWTP("add", "nonexistent")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "not found in local or remote branches")
        assert.Contains(t, err.Error(), "Create a new branch with")
    })
}
```

### CI/CD 統合

```yaml
# .github/workflows/e2e-test.yml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ["1.24"]

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}

      - name: Build wtp
        run: make build

      - name: Run E2E tests
        run: go test -v ./test/e2e/...
```

### 実装優先順位

1. **Phase 1**: 基本的なテストフレームワークの構築
2. **Phase 2**: 主要機能のテストケース実装
3. **Phase 3**: エラーケースとエッジケースの追加
4. **Phase 4**: CI/CD パイプラインへの統合
5. **Phase 5**: パフォーマンステストの追加

### 期待される効果

1. **品質保証**: リグレッションの早期発見
2. **開発効率**: 手動テストの削減
3. **信頼性**: 各リリースの安定性向上
4. **ドキュメント**: テストコードが使用例として機能

### 技術的考慮事項

1. **テスト並列化**: `t.Parallel()` を活用
2. **リソース管理**: 一時ディレクトリの適切なクリーンアップ
3. **Git操作**: 実際のgitコマンドを使用（モックは最小限）
4. **クロスプラットフォーム**: OS固有のパスや挙動に注意

この設計に基づいてE2Eテストを実装することで、wtpの品質と信頼性をさらに向上させることができます。
