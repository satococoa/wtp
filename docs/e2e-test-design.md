# E2E テスト設計書

## 概要

本ドキュメントは、wtp (Worktree Plus) のEnd-to-End (E2E)
テストの設計と実装方針について記述します。手動で実施したテストを自動化し、継続的に品質を保証することを目的とします。

## 設計原則

### 1. 独立性 (Independence)

- 各テストケースは他のテストに依存しない
- テスト順序に関わらず同じ結果を得られる
- 並列実行可能な構造

### 2. 再現性 (Reproducibility)

- 同じ環境で同じ結果を保証
- 外部依存を最小限に抑える
- 決定的な挙動を確保

### 3. 可読性 (Readability)

- テストコードが仕様書として機能
- 意図が明確なテストケース名
- 適切なアサーションメッセージ

### 4. 保守性 (Maintainability)

- DRY原則に基づく共通処理の抽出
- テストヘルパーの活用
- 変更に強い構造

## ディレクトリ構造

```
test/
├── e2e/
│   ├── framework/
│   │   ├── framework.go      # テストフレームワーク
│   │   ├── assertions.go     # カスタムアサーション
│   │   └── fixtures.go       # テストフィクスチャ
│   ├── basic_test.go         # 基本機能テスト
│   ├── worktree_test.go      # ワークツリー管理テスト
│   ├── remote_test.go        # リモートブランチテスト
│   ├── shell_test.go         # シェル統合テスト
│   ├── error_test.go         # エラーメッセージテスト
│   └── testdata/             # テストデータ
└── integration/              # 統合テスト（将来用）
```

## テストフレームワーク

### 基本構造

```go
// test/e2e/framework/framework.go
package framework

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// TestEnvironment はE2Eテスト環境を表す
type TestEnvironment struct {
    t           *testing.T
    tmpDir      string
    repoDir     string
    wtpBinary   string
    cleanup     []func()
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
    t.Helper()

    tmpDir := t.TempDir()
    env := &TestEnvironment{
        t:       t,
        tmpDir:  tmpDir,
        cleanup: []func(){},
    }

    // Build wtp binary for testing
    env.buildWTP()

    return env
}

// CreateTestRepo creates a new git repository for testing
func (e *TestEnvironment) CreateTestRepo(name string) *TestRepo {
    repoDir := filepath.Join(e.tmpDir, name)

    // Initialize git repository
    e.run("git", "init", repoDir)
    e.runInDir(repoDir, "git", "config", "user.name", "Test User")
    e.runInDir(repoDir, "git", "config", "user.email", "test@example.com")

    // Create initial commit
    readmePath := filepath.Join(repoDir, "README.md")
    e.writeFile(readmePath, "# Test Repository")
    e.runInDir(repoDir, "git", "add", ".")
    e.runInDir(repoDir, "git", "commit", "-m", "Initial commit")

    return &TestRepo{
        env:  e,
        path: repoDir,
    }
}

// TestRepo represents a test git repository
type TestRepo struct {
    env  *TestEnvironment
    path string
}

// RunWTP runs wtp command in the repository
func (r *TestRepo) RunWTP(args ...string) (string, error) {
    cmd := exec.Command(r.env.wtpBinary, args...)
    cmd.Dir = r.path
    output, err := cmd.CombinedOutput()
    return string(output), err
}

// CreateBranch creates a new branch
func (r *TestRepo) CreateBranch(name string) {
    r.env.runInDir(r.path, "git", "branch", name)
}

// AddRemote adds a remote repository
func (r *TestRepo) AddRemote(name, url string) {
    r.env.runInDir(r.path, "git", "remote", "add", name, url)
}

// CreateRemoteBranch simulates a remote branch
func (r *TestRepo) CreateRemoteBranch(remote, branch string) {
    refPath := filepath.Join(r.path, ".git", "refs", "remotes", remote)
    os.MkdirAll(refPath, 0755)

    // Get current HEAD commit
    output := r.env.runInDir(r.path, "git", "rev-parse", "HEAD")
    commit := strings.TrimSpace(output)

    // Create remote ref
    r.env.writeFile(filepath.Join(refPath, branch), commit)
}

// Cleanup cleans up the test environment
func (e *TestEnvironment) Cleanup() {
    for _, fn := range e.cleanup {
        fn()
    }
}
```

### カスタムアサーション

```go
// test/e2e/framework/assertions.go
package framework

import (
    "strings"
    "testing"
)

// AssertWorktreeCreated checks if worktree was created successfully
func AssertWorktreeCreated(t *testing.T, output string, branch string) {
    t.Helper()
    if !strings.Contains(output, "Created worktree") {
        t.Errorf("Expected worktree creation message, got: %s", output)
    }
    if !strings.Contains(output, branch) {
        t.Errorf("Expected branch name '%s' in output, got: %s", branch, output)
    }
}

// AssertErrorContains checks if error contains expected message
func AssertErrorContains(t *testing.T, err error, expected string) {
    t.Helper()
    if err == nil {
        t.Errorf("Expected error containing '%s', but got no error", expected)
        return
    }
    if !strings.Contains(err.Error(), expected) {
        t.Errorf("Expected error containing '%s', got: %v", expected, err)
    }
}

// AssertHelpfulError checks if error message is helpful
func AssertHelpfulError(t *testing.T, output string) {
    t.Helper()

    // Check for key elements of helpful error messages
    helpfulElements := []string{
        "Suggestions:",
        "Solutions:",
        "Cause:",
        "Tip:",
        "•", // Bullet points
    }

    found := false
    for _, element := range helpfulElements {
        if strings.Contains(output, element) {
            found = true
            break
        }
    }

    if !found {
        t.Errorf("Error message does not appear to be helpful. Got: %s", output)
    }
}
```

## テストケース実装例

### 基本機能テスト

```go
// test/e2e/basic_test.go
package e2e

import (
    "testing"
    "github.com/satococoa/wtp/test/e2e/framework"
)

func TestBasicCommands(t *testing.T) {
    env := framework.NewTestEnvironment(t)
    defer env.Cleanup()

    t.Run("Version", func(t *testing.T) {
        output, err := env.RunWTP("--version")
        if err != nil {
            t.Fatalf("Failed to run version command: %v", err)
        }
        if !strings.Contains(output, "wtp version") {
            t.Errorf("Expected version output, got: %s", output)
        }
    })

    t.Run("Help", func(t *testing.T) {
        output, err := env.RunWTP("--help")
        if err != nil {
            t.Fatalf("Failed to run help command: %v", err)
        }

        expectedCommands := []string{"add", "remove", "list", "init", "cd"}
        for _, cmd := range expectedCommands {
            if !strings.Contains(output, cmd) {
                t.Errorf("Expected command '%s' in help output", cmd)
            }
        }
    })
}

func TestInitCommand(t *testing.T) {
    env := framework.NewTestEnvironment(t)
    defer env.Cleanup()

    repo := env.CreateTestRepo("init-test")

    t.Run("CreateConfig", func(t *testing.T) {
        output, err := repo.RunWTP("init")
        if err != nil {
            t.Fatalf("Failed to init: %v", err)
        }

        if !strings.Contains(output, "Configuration file created") {
            t.Errorf("Expected success message, got: %s", output)
        }

        // Verify file exists
        configPath := filepath.Join(repo.path, ".wtp.yml")
        if _, err := os.Stat(configPath); os.IsNotExist(err) {
            t.Error("Configuration file was not created")
        }
    })

    t.Run("ConfigAlreadyExists", func(t *testing.T) {
        _, err := repo.RunWTP("init")
        if err == nil {
            t.Fatal("Expected error for existing config, got none")
        }

        framework.AssertErrorContains(t, err, "already exists")
        framework.AssertHelpfulError(t, err.Error())
    })
}
```

### ワークツリー管理テスト

```go
// test/e2e/worktree_test.go
package e2e

import (
    "testing"
    "github.com/satococoa/wtp/test/e2e/framework"
)

func TestWorktreeCreation(t *testing.T) {
    env := framework.NewTestEnvironment(t)
    defer env.Cleanup()

    repo := env.CreateTestRepo("worktree-test")
    repo.CreateBranch("feature/test")

    t.Run("LocalBranch", func(t *testing.T) {
        output, err := repo.RunWTP("add", "feature/test")
        if err != nil {
            t.Fatalf("Failed to add worktree: %v", err)
        }

        framework.AssertWorktreeCreated(t, output, "feature/test")
    })

    t.Run("NonexistentBranch", func(t *testing.T) {
        output, err := repo.RunWTP("add", "nonexistent")
        if err == nil {
            t.Fatal("Expected error for nonexistent branch")
        }

        framework.AssertErrorContains(t, err, "not found in local or remote branches")
        framework.AssertErrorContains(t, err, "Create a new branch with")
        framework.AssertHelpfulError(t, output)
    })

    t.Run("NewBranch", func(t *testing.T) {
        output, err := repo.RunWTP("add", "-b", "new-feature")
        if err != nil {
            t.Fatalf("Failed to create new branch: %v", err)
        }

        framework.AssertWorktreeCreated(t, output, "new-feature")
    })
}

func TestWorktreeRemoval(t *testing.T) {
    env := framework.NewTestEnvironment(t)
    defer env.Cleanup()

    repo := env.CreateTestRepo("remove-test")
    repo.CreateBranch("feature/remove")

    // Create worktree first
    _, err := repo.RunWTP("add", "feature/remove")
    if err != nil {
        t.Fatalf("Failed to create worktree: %v", err)
    }

    t.Run("RemoveWorktree", func(t *testing.T) {
        output, err := repo.RunWTP("remove", "feature/remove")
        if err != nil {
            t.Fatalf("Failed to remove worktree: %v", err)
        }

        if !strings.Contains(output, "Removed worktree") {
            t.Errorf("Expected removal message, got: %s", output)
        }
    })

    t.Run("RemoveNonexistent", func(t *testing.T) {
        _, err := repo.RunWTP("remove", "nonexistent")
        if err == nil {
            t.Fatal("Expected error for nonexistent worktree")
        }

        framework.AssertHelpfulError(t, err.Error())
    })
}
```

### リモートブランチテスト

```go
// test/e2e/remote_test.go
package e2e

import (
    "testing"
    "github.com/satococoa/wtp/test/e2e/framework"
)

func TestRemoteBranchHandling(t *testing.T) {
    env := framework.NewTestEnvironment(t)
    defer env.Cleanup()

    repo := env.CreateTestRepo("remote-test")
    repo.AddRemote("origin", "https://example.com/repo.git")

    t.Run("SingleRemoteBranch", func(t *testing.T) {
        repo.CreateRemoteBranch("origin", "remote-feature")

        output, err := repo.RunWTP("add", "remote-feature")
        if err != nil {
            t.Fatalf("Failed to track remote branch: %v", err)
        }

        framework.AssertWorktreeCreated(t, output, "remote-feature")
    })

    t.Run("MultipleRemotes", func(t *testing.T) {
        repo.AddRemote("upstream", "https://example.com/upstream.git")
        repo.CreateRemoteBranch("origin", "shared-branch")
        repo.CreateRemoteBranch("upstream", "shared-branch")

        _, err := repo.RunWTP("add", "shared-branch")
        if err == nil {
            t.Fatal("Expected error for ambiguous branch")
        }

        framework.AssertErrorContains(t, err, "exists in multiple remotes")
        framework.AssertErrorContains(t, err, "origin, upstream")
        framework.AssertErrorContains(t, err, "--track")
    })
}
```

## CI/CD 統合

### GitHub Actions ワークフロー

```yaml
# .github/workflows/e2e-test.yml
name: E2E Tests

on:
    push:
        branches: [main]
    pull_request:
        branches: [main]

jobs:
    e2e-test:
        name: E2E Tests (${{ matrix.os }})
        runs-on: ${{ matrix.os }}
        strategy:
            fail-fast: false
            matrix:
                os: [ubuntu-latest, macos-latest]
                go-version: ["1.24"]

        steps:
            - name: Checkout code
              uses: actions/checkout@v4

            - name: Setup Go
              uses: actions/setup-go@v5
              with:
                  go-version: ${{ matrix.go-version }}

            - name: Install dependencies
              run: go mod download

            - name: Build wtp
              run: make build

            - name: Run E2E tests
              run: |
                  go test -v -race -timeout 10m ./test/e2e/...
              env:
                  WTP_E2E_BINARY: ./wtp

            - name: Upload test results
              if: always()
              uses: actions/upload-artifact@v4
              with:
                  name: e2e-test-results-${{ matrix.os }}
                  path: test-results/
```

### ローカル実行

```bash
# E2Eテストの実行
make test-e2e

# 特定のテストのみ実行
go test -v ./test/e2e/... -run TestWorktreeCreation

# 並列実行
go test -v -parallel 4 ./test/e2e/...

# カバレッジ付き
go test -v -cover ./test/e2e/...
```

## 実装ロードマップ

### Phase 1: 基盤構築 (1週間)

- [ ] テストフレームワークの実装
- [ ] 基本的なヘルパー関数
- [ ] CI/CD パイプラインの設定

### Phase 2: コアテスト実装 (2週間)

- [ ] 基本機能テスト
- [ ] ワークツリー管理テスト
- [ ] エラーメッセージテスト

### Phase 3: 高度なテスト (1週間)

- [ ] リモートブランチテスト
- [ ] シェル統合テスト
- [ ] エッジケーステスト

### Phase 4: 最適化と拡張 (継続的)

- [ ] テスト並列化の最適化
- [ ] パフォーマンステスト追加
- [ ] ベンチマークテスト

## ベストプラクティス

### 1. テストの独立性を保つ

```go
func TestFeature(t *testing.T) {
    // 各テストで新しい環境を作成
    env := framework.NewTestEnvironment(t)
    defer env.Cleanup()

    // テスト固有のリポジトリを作成
    repo := env.CreateTestRepo("unique-name")
    // ...
}
```

### 2. 明確なテスト名

```go
// Good
func TestWorktreeCreation_LocalBranch_Success(t *testing.T)
func TestWorktreeCreation_NonexistentBranch_ReturnsHelpfulError(t *testing.T)

// Bad
func TestAdd(t *testing.T)
func TestError(t *testing.T)
```

### 3. アサーションメッセージ

```go
// Good
if !strings.Contains(output, expected) {
    t.Errorf("Expected output to contain '%s', got: %s", expected, output)
}

// Bad
if !strings.Contains(output, expected) {
    t.Error("Test failed")
}
```

### 4. テストデータの管理

```go
// testdata/ ディレクトリを活用
func loadTestData(t *testing.T, name string) string {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", name))
    if err != nil {
        t.Fatalf("Failed to load test data: %v", err)
    }
    return string(data)
}
```

## メンテナンス

### テスト追加時のチェックリスト

- [ ] 独立して実行可能か
- [ ] 並列実行に対応しているか
- [ ] クリーンアップが適切か
- [ ] エラーメッセージが分かりやすいか
- [ ] CI/CDで実行されるか

### 定期的な見直し

- 月次: テストカバレッジの確認
- 四半期: テスト実行時間の最適化
- 半年: フレームワークの改善

## まとめ

このE2Eテスト設計により、wtpの品質を継続的に保証し、ユーザーに安定した体験を提供できます。テストは単なる品質保証の手段ではなく、仕様のドキュメントとしても機能し、新規開発者の理解を助ける重要な資産となります。
