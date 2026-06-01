import sys
import subprocess
import json
import re

def slugify(text, max_length=40):
    text = text.lower()
    text = re.sub(r'[^\w\s-]', '', text)
    text = re.sub(r'[\s_-]+', '-', text)
    text = text.strip('-')
    if len(text) > max_length:
        text = text[:max_length]
        last_hyphen = text.rfind('-')
        if last_hyphen > 10:
            text = text[:last_hyphen]
    return text

def check_git(issue_type, title):
    branch_prefix = "feature" if issue_type == "enhancement" else issue_type
    slug = slugify(title)
    
    try:
        # 1. Find the issue number first
        issue_result = subprocess.run(
            ["gh", "issue", "list", "--label", issue_type, "--search", f"in:title {title}", "--json", "title,number"],
            capture_output=True,
            text=True,
            check=True
        )
        issues = json.loads(issue_result.stdout)
        matching_issues = [i for i in issues if i['title'] == title]
        
        if not matching_issues:
            print(json.dumps({"errors": [f"Cannot check branch: No matching issue found for title '{title}'"]}))
            return 1
            
        issue_number = matching_issues[0]['number']
        expected_branch = f"{branch_prefix}/{issue_number}-{slug}"
        
        # 2. Get current branch
        result = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            capture_output=True,
            text=True,
            check=True
        )
        current_branch = result.stdout.strip()
        
        # 3. Validation
        if current_branch == expected_branch:
            # Check naming convention: lowercase, no spaces, starts with correct prefix
            if not re.match(r'^[a-z0-9/-]+$', current_branch):
                 print(json.dumps({"errors": [f"Branch '{current_branch}' fails naming convention (must be lowercase alphanumeric with hyphens)"]}))
                 return 1
            
            print(json.dumps({
                "message": f"Correctly on branch '{current_branch}'",
                "linked_issue": issue_number,
                "convention": "VALID"
            }))
            return 0
        else:
            # Check if the branch exists at all
            check_exists = subprocess.run(
                ["git", "branch", "--list", expected_branch],
                capture_output=True,
                text=True
            )
            if expected_branch in check_exists.stdout:
                 print(json.dumps({"errors": [f"Branch '{expected_branch}' exists but is not checked out. Current: '{current_branch}'"]}))
            else:
                 print(json.dumps({"errors": [f"Expected branch '{expected_branch}' does not exist. Current: '{current_branch}'"]}))
            return 1
            
    except Exception as e:
        print(json.dumps({"errors": [f"Git check failed: {str(e)}"]}))
        return 1

if __name__ == "__main__":
    if len(sys.argv) < 3:
        sys.exit(1)
    sys.exit(check_git(sys.argv[1], sys.argv[2]))
