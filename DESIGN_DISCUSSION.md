# git-wtp 透明ラッパー設計議論のまとめ

## 議論の流れ

### 1. 初期の問題：`-b`オプションの挙動

**発見した問題：**
- `git-wtp add my-worktree -b feature/auth`で既存ブランチがある場合、エラーにならず使用してしまう
- Git標準（`git checkout -b`）では既存ブランチがあればエラーになる

**解決：**
- `-b`フラグを新規ブランチ作成専用に修正
- Git標準の挙動に合わせた

### 2. コマンド体系の簡素化

**問題：**
- `git-wtp add my-worktree feature/auth`の引数順序が混乱を招く
- worktree名とブランチ名を別々で指定する意味が薄い

**解決：**
- worktree名 = ブランチ名に統一
- `git-wtp add <branch-name> [-b]`に簡素化

### 3. Git Worktreeとの機能重複の発見

**気づき：**
- `git worktree add`は既に`-b/-B`、`--track`等の豊富なオプションを持つ
- 自前でブランチ解決ロジックを実装する必要はない
- 自動リモート追跡も`git worktree add`に内蔵されている

**結論：**
- 複雑な自前実装を削除
- 透明ラッパーアプローチに移行

### 4. コアバリューの再定義

**git-wtpの真の価値：**
1. **パス管理** - 設定可能なworktreeディレクトリ構造
2. **フック** - Post-create hooks（ファイルコピー、コマンド実行）
3. **削除時のブランチ同時削除**（未実装）

**不要になった機能：**
- ~~自動リモート追跡~~ → `git worktree add`の`--track`で十分
- ~~ブランチ解決ロジック~~ → `git worktree add`に任せる
- ~~複雑なブランチ作成ロジック~~ → `git worktree add -b/-B`で十分

### 5. 透明ラッパー実装の課題

**問題：**
- `worktree ≒ branch`の前提がgit worktreeの柔軟性を殺している
- Detached HEAD worktreeが作れない
- 特定コミットからの作成が困難

### 6. 元々の課題の再確認

**根本的な問題：**
```bash
# git worktreeの冗長性
git worktree add ../worktrees/feature/auth feature/auth
#                 ^^^^^^^^^^^^^^^^^^^ ^^^^^^^^^^^
#                 パス                ブランチ名（ほぼ同じ）
```

## 最終的な結論と設計方針

### 解決すべき課題
- **冗長性の解消**：ブランチ名を2回打つ問題
- **柔軟性の維持**：git worktreeの全機能をサポート
- **パス管理**：チーム共通のディレクトリ構造

### 採用する設計

**ハイブリッドアプローチ：**

```bash
# ✅ シンプル：ブランチ名から自動パス生成
git-wtp add feature/auth
# → git worktree add ../worktrees/feature/auth feature/auth

# ✅ 柔軟性：明示的パス指定はそのまま
git-wtp add /custom/path feature/auth
git-wtp add --detach /tmp/experiment abc1234

# ✅ 新規ブランチも簡潔
git-wtp add -b feature/new
# → git worktree add ../worktrees/feature/new -b feature/new
```

### 実装のポイント

1. **引数解析の改善**
   - 絶対パス/相対パス → そのまま使用
   - ブランチ名っぽい文字列 → 自動パス生成

2. **git worktreeとの完全互換性**
   - 全オプションをパススルー
   - エラーメッセージもgitのまま

3. **コアバリューの維持**
   - パス管理（設定ファイルベース）
   - Post-create hooks
   - 将来：削除時ブランチ同時削除

### 期待される効果

- **学習コスト削減**：git worktreeを知っていれば即使える
- **冗長性解消**：ブランチ名の重複入力が不要
- **柔軟性維持**：git worktreeの全機能が使える
- **チーム統一**：設定ファイルによる共通パス管理

## 次のステップ

### 実装すべき機能

1. **引数解析の改善**
   ```go
   func resolveWorktreePath(cfg *config.Config, firstArg string, flags) (path, branchName string) {
       if isPath(firstArg) {
           // 明示的パス指定
           return firstArg, ""
       }
       // ブランチ名から自動生成
       return cfg.ResolveWorktreePath(repo.Path(), firstArg), firstArg
   }
   ```

2. **パス判定ロジック**
   - 絶対パス（`/`で始まる）
   - 相対パス（`./`や`../`で始まる）
   - その他はブランチ名として扱う

3. **git worktreeコマンド構築の修正**
   - パス明示時：引数をそのまま使用
   - パス省略時：自動生成パスを挿入

4. **テストケースの追加**
   - 明示的パス指定のテスト
   - detached HEAD worktreeのテスト
   - 各種git worktreeオプションのテスト

### 将来の拡張

1. **削除時ブランチ同時削除**
   - `git-wtp remove <worktree> --with-branch`

2. **設定ファイルの拡張**
   - パスパターンの設定
   - ブランチタイプ別のディレクトリ構造

3. **Shell統合**
   - `git-wtp cd <worktree>`コマンド
   - Shell補完の改善