import sys
import subprocess
import json

def check_cleanup(branch_name):
    try:
        errors = []
        
        # 1. Check current branch is main
        curr_result = subprocess.run(
            ["git", "branch", "--show-current"],
            capture_output=True,
            text=True,
            check=True
        )
        current_branch = curr_result.stdout.strip()
        if current_branch != "main":
            errors.append(f"Cleanup incomplete: Current branch is '{current_branch}', should be 'main'")

        # 2. Check local branch deletion
        local_result = subprocess.run(
            ["git", "branch", "--list", branch_name],
            capture_output=True,
            text=True,
            check=True
        )
        if branch_name in local_result.stdout:
            errors.append(f"Cleanup incomplete: Local branch '{branch_name}' still exists")

        # 3. Check remote branch deletion
        remote_result = subprocess.run(
            ["git", "ls-remote", "--heads", "origin", branch_name],
            capture_output=True,
            text=True,
            check=True
        )
        if branch_name in remote_result.stdout:
            errors.append(f"Cleanup incomplete: Remote branch 'origin/{branch_name}' still exists")

        if errors:
            print(json.dumps({"errors": errors}))
            return 1
        else:
            print(json.dumps({
                "message": f"Branch '{branch_name}' successfully cleaned up locally and remotely. Current branch: main.",
                "status": "CLEAN"
            }))
            return 0
            
    except Exception as e:
        print(json.dumps({"errors": [f"Cleanup check failed: {str(e)}"]}))
        return 1

if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(1)
    sys.exit(check_cleanup(sys.argv[1]))
