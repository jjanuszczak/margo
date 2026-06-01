import sys
import subprocess
import json
import re

def check_convention(issue_type, title):
    try:
        # Get current branch
        result = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            capture_output=True,
            text=True,
            check=True
        )
        branch_name = result.stdout.strip()
        
        errors = []
        
        # 1. Check for lowercase
        if branch_name != branch_name.lower():
            errors.append("Branch name must be all lowercase")
            
        # 2. Check for invalid characters
        if not re.match(r'^[a-z0-9/-]+$', branch_name):
            errors.append("Branch name contains invalid characters (only lowercase, numbers, '/', and '-' allowed)")
            
        # 3. Check for prefix
        expected_prefix = "feature/" if issue_type == "enhancement" else f"{issue_type}/"
        if not branch_name.startswith(expected_prefix):
            errors.append(f"Branch name must start with '{expected_prefix}'")
            
        # 4. Check for issue number linking pattern: prefix/number-slug
        pattern = rf"^{expected_prefix}\d+-"
        if not re.match(pattern, branch_name):
            errors.append(f"Branch name must include the issue number linked as '{expected_prefix}NUMBER-slug'")

        if errors:
            print(json.dumps({"errors": errors, "branch": branch_name}))
            return 1
        else:
            print(json.dumps({"message": f"Branch '{branch_name}' follows all naming and linking conventions", "status": "VALID"}))
            return 0
            
    except Exception as e:
        print(json.dumps({"errors": [f"Convention check failed: {str(e)}"]}))
        return 1

if __name__ == "__main__":
    if len(sys.argv) < 3:
        sys.exit(1)
    sys.exit(check_convention(sys.argv[1], sys.argv[2]))
