## Git Workflow Rules

- Never commit or push without explicit user approval
- Always work on the dev branch
- Never push directly to main

## Commit Preparation

Before committing:

- Show git status
- List all changed files
- Summarize changes clearly

## Commit Messages

Use conventional commits:

- feat: new feature
- fix: bug fix
- refactor: code improvement
- test: test changes
- chore: setup/config

Format:
<type>: <short description>

Example:
feat: implement interactions endpoint with validation

## Push Rules

- Only push after user approval
- Push to origin/dev

## Safety

- Never commit secrets (.env, keys)
- Respect .gitignore
