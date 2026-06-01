import sys
import subprocess
import json
import re

def check_pr(branch_name):
    try:
        # 1. Get PR info
        result = subprocess.run(
            ["gh", "pr", "view", branch_name, "--json", "number,body,url,state"],
            capture_output=True,
            text=True,
            check=True
        )
        
        pr_data = json.loads(result.stdout)
        
        errors = []
        
        # 2. Verify Linkage (Closes #XX)
        # Extract issue number from branch name
        issue_match = re.search(r'/(\d+)-', branch_name)
        if issue_match:
            issue_number = issue_match.group(1)
            expected_link = f"Closes #{issue_number}"
            if expected_link not in pr_data['body']:
                errors.append(f"PR body missing mandatory linkage '{expected_link}'")
        
        if errors:
            print(json.dumps({"errors": errors, "pr": pr_data}))
            return 1
        else:
            print(json.dumps({
                "message": f"PR #{pr_data['number']} verified and correctly linked.",
                "url": pr_data['url'],
                "status": "VALID"
            }))
            return 0
            
    except Exception as e:
        print(json.dumps({"errors": [f"PR check failed: {str(e)}"]}))
        return 1

if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(1)
    sys.exit(check_pr(sys.argv[1]))
