import sys
import subprocess
import json

def check_github(issue_type, title):
    try:
        # 1. Search for the issue using gh CLI
        result = subprocess.run(
            ["gh", "issue", "list", "--label", issue_type, "--search", f"in:title {title}", "--json", "title,labels,number,url"],
            capture_output=True,
            text=True,
            check=True
        )
        
        issues = json.loads(result.stdout)
        matching_issues = [i for i in issues if i['title'] == title]
        
        if not matching_issues:
            print(json.dumps({"errors": [f"No issue found with title '{title}' and label '{issue_type}'"]}))
            return 1
            
        issue = matching_issues[0]
        issue_number = str(issue['number'])
        
        # 2. Check for linked branches using 'gh issue develop --list'
        link_result = subprocess.run(
            ["gh", "issue", "develop", "--list", issue_number],
            capture_output=True,
            text=True,
            check=True
        )
        
        # The output of 'gh issue develop --list' is a bit raw, usually something like:
        # branch-name  repo-name  state
        # Or it might be empty if no branches are linked.
        
        linked_output = link_result.stdout.strip()
        
        if linked_output:
            print(json.dumps({
                "message": f"Found matching issue #{issue_number} with linked branches.",
                "url": issue['url'],
                "linked_branches": linked_output.split('\n')
            }))
            return 0
        else:
            # We created the branch with 'gh issue develop', so it SHOULD be linked.
            print(json.dumps({
                "message": f"Found matching issue #{issue_number}, but NO linked branches found in GitHub metadata.",
                "url": issue['url'],
                "status": "ISSUE_FOUND_UNLINKED"
            }))
            # Returning 0 for now as the issue itself is found, but maybe it should be 1 if linking is strict.
            # User specifically asked to ENSURE they are linked.
            return 1
            
    except Exception as e:
        print(json.dumps({"errors": [f"GitHub check failed: {str(e)}"]}))
        return 1

if __name__ == "__main__":
    if len(sys.argv) < 3:
        sys.exit(1)
    sys.exit(check_github(sys.argv[1], sys.argv[2]))
