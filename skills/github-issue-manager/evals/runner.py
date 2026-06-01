import os
import sys
import yaml
import json
import subprocess
from datetime import datetime

def run_eval(issue_type, title):
    base_dir = os.path.dirname(os.path.abspath(__file__))
    config_path = os.path.join(base_dir, "config.yaml")
    
    if not os.path.exists(config_path):
        print(f"Error: Config file not found at {config_path}")
        return 1

    with open(config_path, 'r') as f:
        config = yaml.safe_load(f)
    
    results = {
        "timestamp": datetime.now().isoformat(),
        "issue_type": issue_type,
        "title": title,
        "overall_status": "PASS",
        "checks": []
    }
    
    for step in config['pipeline']['steps']:
        script_path = os.path.join(base_dir, step['script'])
        check_id = step['id']
        name = step['name']
        
        print(f"Running check: {name}...")
        
        try:
            # Pass issue_type and title as arguments to each check script
            result = subprocess.run(
                [sys.executable, script_path, issue_type, title],
                capture_output=True,
                text=True
            )
            
            status = "PASS" if result.returncode == 0 else "FAIL"
            
            try:
                output_data = json.loads(result.stdout.strip())
            except json.JSONDecodeError:
                output_data = {"message": result.stdout.strip() or result.stderr.strip()}
            
            check_result = {
                "id": check_id,
                "name": name,
                "status": status,
                "details": output_data
            }
            
            results["checks"].append(check_result)
            
            if status == "FAIL":
                results["overall_status"] = "FAIL"
                if step.get('halt_on_fail', False):
                    print(f"Critical failure in {name}. Halting pipeline.")
                    break
                    
        except Exception as e:
            results["overall_status"] = "FAIL"
            results["checks"].append({
                "id": check_id,
                "name": name,
                "status": "ERROR",
                "details": {"error": str(e)}
            })
            if step.get('halt_on_fail', False):
                break

    # Save report
    report_path = os.path.join(base_dir, "reports", "latest_results.json")
    with open(report_path, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"Evaluation complete. Status: {results['overall_status']}")
    print(f"Report saved to: {report_path}")
    
    return 0 if results["overall_status"] == "PASS" else 1

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python runner.py <issue_type> <title>")
        sys.exit(1)
        
    i_type = sys.argv[1]
    i_title = sys.argv[2]
    sys.exit(run_eval(i_type, i_title))
