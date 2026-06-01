import argparse
import subprocess
import sys
import re

def run_command(command):
    try:
        result = subprocess.run(command, capture_output=True, text=True, check=True)
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        print(f"Error executing command: {' '.join(command)}")
        print(f"Stderr: {e.stderr}")
        sys.exit(1)

def slugify(text, max_length=40):
    text = text.lower()
    text = re.sub(r'[^\w\s-]', '', text)
    text = re.sub(r'[\s_-]+', '-', text)
    text = text.strip('-')
    if len(text) > max_length:
        # Cut at the last hyphen before max_length to avoid breaking words
        text = text[:max_length]
        last_hyphen = text.rfind('-')
        if last_hyphen > 10:
            text = text[:last_hyphen]
    return text

def main():
    parser = argparse.ArgumentParser(description="Create a GitHub issue and a corresponding branch.")
    parser.add_argument("title", help="Title of the issue")
    parser.add_argument("description", help="Description of the issue")
    parser.add_argument("--type", choices=["bug", "enhancement", "documentation"], required=True, help="Type of issue")
    
    args = parser.parse_args()

    # Map 'enhancement' to 'feature' for branch naming
    branch_type = "feature" if args.type == "enhancement" else args.type

    # 1. Create the issue using gh CLI
    print(f"Creating {args.type} issue: {args.title}...")
    
    issue_command = [
        "gh", "issue", "create",
        "--title", args.title,
        "--body", args.description,
        "--label", args.type
    ]
    
    issue_output = run_command(issue_command)
    issue_url = issue_output
    issue_number = issue_url.split("/")[-1]
    
    print(f"Issue created: {issue_url}")

    # 2. Create the branch using 'gh issue develop'
    # This creates the branch AND links it to the issue in GitHub's metadata.
    # We use the naming convention: type/number-slug
    slug = slugify(args.title)
    branch_name = f"{branch_type}/{issue_number}-{slug}"
    
    print(f"Creating linked branch: {branch_name}...")
    
    run_command(["gh", "issue", "develop", issue_number, "--name", branch_name, "--checkout"])
    
    print(f"Successfully created issue #{issue_number} and switched to linked branch {branch_name}.")

if __name__ == "__main__":
    main()
