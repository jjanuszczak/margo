---
name: git-workflow-lifecycle
description: Automate the end-to-end lifecycle of a feature branch, from submission (commit, push, PR) to cleanup (merge check, sync main, delete branch). Use when you have finished work on a branch or when a PR has been merged and you need to clean up.
---

# Git Workflow Lifecycle

This skill manages the transition of code from a local development branch to GitHub and the subsequent cleanup once the work is merged.

## Workflows

### 1. Submit Work
Use this when you have finished implementation on a branch and are ready for review.
- **Action:** Stages all changes, commits with a message, pushes to origin, and creates a GitHub Pull Request.
- **Linkage:** Automatically adds "Closes #XX" if an issue number is detected in the branch name.

```bash
python3 .gemini/skills/git-workflow-lifecycle/scripts/submit_work.py --message "feat: description of work"
```

### 2. Cleanup Branch
Use this after your Pull Request has been merged on GitHub.
- **Action:** Verifies the PR is merged, switches to `main`, pulls latest updates, and deletes both local and remote branches.

```bash
python3 .gemini/skills/git-workflow-lifecycle/scripts/cleanup_branch.py
```

## Instructions
- Ensure `gh` CLI is authenticated.
- The `submit_work.py` script will prompt for a commit message if `--message` is omitted.
- The `cleanup_branch.py` script will warn you if the PR is not yet merged before proceeding with deletion.

## Scripts
- `scripts/submit_work.py`
- `scripts/cleanup_branch.py`

## Self-Evaluation & Correction

You MUST autonomously verify every git workflow task:

### 1. Verify Submission
Immediately after running `submit_work.py`:
1.  **Run Evaluation Suite:** `python3 .gemini/skills/git-workflow-lifecycle/evals/runner.py submit <branch_name>`
2.  **Analyze Report:** Read results in `.gemini/skills/git-workflow-lifecycle/evals/reports/latest_submit_results.json`.
3.  **Self-Correction:** If the PR is missing or not linked, attempt to fix the PR body using `gh pr edit`.

### 2. Verify Cleanup
Immediately after running `cleanup_branch.py`:
1.  **Run Evaluation Suite:** `python3 .gemini/skills/git-workflow-lifecycle/evals/runner.py cleanup <branch_name>`
2.  **Analyze Report:** Read results in `.gemini/skills/git-workflow-lifecycle/evals/reports/latest_cleanup_results.json`.
3.  **Self-Correction:** If branches remain, manually delete them (`git branch -D` or `git push origin --delete`) and ensure you are on `main`.

<AVAILABLE_RESOURCES>
*   **Evaluation Runner:** `.gemini/skills/git-workflow-lifecycle/evals/runner.py`
</AVAILABLE_RESOURCES>
