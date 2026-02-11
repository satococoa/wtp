# Workflow

## Git and PR
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `style:`.
- Branch naming: `feature/...`, `fix/...`, `hotfix/...`.
- Before opening a PR, run: format, lint, tests.
- Include test notes and related issues in PR descriptions (for example `Closes #123`).
- Add CLI output or screenshots when UX changes.

## Config and Secrets
- Project hooks are defined in `.wtp.yml`.
- Never commit secrets.
- Prefer example files (for example `.env.example`) for hook-based setup.
