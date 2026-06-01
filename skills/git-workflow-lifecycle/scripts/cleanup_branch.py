import subprocess
import sys
import json

def run_command(command):
    try:
        result = subprocess.run(command, capture_output=True, text=True, check=True)
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        return None

def main():
    # 1. Get PR status for current branch
    branch = run_command(["git", "branch", "--show-current"])
    if not branch or branch == "main":
        print("Error: You are not on a feature branch.")
        sys.exit(1)

    print(f"Checking status of PR for branch '{branch}'...")
    pr_json = run_command(["gh", "pr", "view", branch, "--json", "state,number"])
    
    if not pr_json:
        print(f"No PR found for branch '{branch}'.")
        confirm = input("Do you want to delete this branch anyway? (y/n): ")
        if confirm.lower() != 'y':
            sys.exit(0)
    else:
        pr_data = json.loads(pr_json)
        state = pr_data.get("state")
        
        if state != "MERGED":
            print(f"Warning: PR #{pr_data.get('number')} is currently {state}.")
            confirm = input("Are you sure you want to delete the branch? (y/n): ")
            if confirm.lower() != 'y':
                sys.exit(0)
        else:
            print(f"PR #{pr_data.get('number')} is merged. Starting cleanup...")

    # 2. Cleanup
    print("Switching to main...")
    run_command(["git", "checkout", "main"])
    
    print("Pulling latest changes from main...")
    run_command(["git", "pull", "origin", "main"])
    
    print(f"Deleting local branch '{branch}'...")
    run_command(["git", "branch", "-d", branch]) or run_command(["git", "branch", "-D", branch])
    
    print(f"Deleting remote branch 'origin/{branch}'...")
    run_command(["git", "push", "origin", "--delete", branch])
    
    print("Cleanup complete.")

if __name__ == "__main__":
    main()
