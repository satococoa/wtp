# Test Audit Findings - Phase 1

## Overview
This document summarizes the findings from the audit of coverage-driven anti-patterns in the wtp test suite, as part of Issue #3 and #4.

## Coverage-Driven Tests Removed

### 1. TestParseWorktreeListIntegration_InvalidFormat
- **Location**: `internal/git/repository_integration_test.go`
- **Issue**: Tests unrealistic scenario - git would never output "invalid format"
- **Action**: Removed - The parseWorktreeList function only receives input from `git worktree list --porcelain`, which has a well-defined format
- **Impact**: Removes meaningless test that existed only for coverage

### 2. TestGetWorktrees_DetachedHead  
- **Location**: `internal/git/repository_integration_test.go`
- **Issue**: Skipped with comment "not critical for coverage goals" - classic coverage-driven anti-pattern
- **Action**: Removed - Detached HEAD functionality is already properly tested in e2e tests (`test/e2e/worktree_test.go`)
- **Impact**: Removes redundant skipped test

## Tests Requiring Renaming

Multiple tests use "Integration" suffix which describes the test type rather than the behavior being tested. These need to be renamed to reflect user expectations:

1. **TestParseWorktreeListIntegration** → Should describe what parsing worktree list means for users
2. **TestParseWorktreeListIntegration_EmptyInput** → Should describe behavior when no worktrees exist
3. **TestShellIntegrationRequired** → Should describe the user-facing error scenario
4. **TestConfigBaseDirIntegration** → Should describe how base_dir configuration affects worktree creation
5. **TestCompletionScriptIntegration** → Should describe shell completion functionality
6. **TestShellIntegration** → Should describe cd command behavior
7. **TestCdToWorktree_NoShellIntegration** → Should describe error when shell integration isn't set up

## Other Observations

### Positive Findings
- Most tests do express user behaviors (e.g., "AddUsesConfigBaseDir", "CDCommandWithoutIntegration")
- E2E tests are well-structured and test real user scenarios
- Good coverage of error cases in many areas

### Areas for Improvement
- Test names should follow Given-When-Then pattern or describe expected outcomes
- Some test files have good structure but poor naming conventions
- Integration vs unit test distinction in names adds no value for users

## Metrics

### Before Cleanup
- Coverage: 84.6% (per Issue #3)
- Removed tests: 2
- Tests needing rename: ~7-10

### Expected After Full Implementation
- Coverage: ~80%+ (natural result of meaningful tests)
- All tests express clear user value
- No coverage-driven tests remain

## Next Steps
1. Rename tests with "Integration" suffix (Phase 2 - Issue #5)
2. Structure tests in Given-When-Then format (Phase 2 - Issue #5)
3. Add meaningful edge case tests (Phase 3 - Issue #6)
4. Create testing guidelines (Phase 4 - Issue #7)