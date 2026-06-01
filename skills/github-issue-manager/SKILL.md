---
name: github-issue-manager
description: Create GitHub issues and development branches. Use when the user wants to start work on a bug, feature (enhancement), or documentation task and needs a structured way to track it.
---

# GitHub Issue Manager

This skill automates the creation of GitHub issues and the setup of local development branches.

## Workflow

1.  **Create Issue & Branch:** Run the `manage_issue.py` script to create a GitHub issue with appropriate labels and switch to a new branch.
    ```bash
    python3 .gemini/skills/github-issue-manager/scripts/manage_issue.py "Title" "Description" --type <bug|enhancement|documentation>
    ```

## Instructions

- **Issue Types:**
    - `bug`: For reporting errors or unexpected behavior.
    - `enhancement`: For new features or improvements.
    - `documentation`: For updates to READMEs, guides, or site content.
- **Branch Naming:** The script generates a concise branch name in the format `type/abbreviated-title`.
    - `enhancement` maps to `feature/`
    - `bug` maps to `bug/`
    - `documentation` maps to `documentation/`
- **Requirements:** Requires the `gh` (GitHub CLI) to be installed and authenticated.

## Scripts

- `scripts/manage_issue.py`: The core automation script.

## Self-Evaluation & Correction

You MUST autonomously verify every issue management task:
1.  **Run Evaluation Suite:** `python3 .gemini/skills/github-issue-manager/evals/runner.py <issue_type> "<title>"`
2.  **Analyze Report:** Read results in `.gemini/skills/github-issue-manager/evals/reports/latest_results.json`.
3.  **Self-Correction Loop:**
    - **Attempt 1:** If any checks `FAIL` (e.g., branch not created, issue missing), analyze the error and attempt to fix the issue.
    - **Attempt 2:** One final targeted fix and re-run.
4.  **Escalation:** If still failing after 2 attempts, stop and present the failure report to the user.

<AVAILABLE_RESOURCES>
*   **Evaluation Runner:** `.gemini/skills/github-issue-manager/evals/runner.py`
</AVAILABLE_RESOURCES>
