import os
import sys
import yaml
import json
import subprocess
from datetime import datetime

def run_eval(task, identifier):
    base_dir = os.path.dirname(os.path.abspath(__file__))
    config_path = os.path.join(base_dir, "config.yaml")
    
    if not os.path.exists(config_path):
        print(f"Error: Config file not found at {config_path}")
        return 1

    with open(config_path, 'r') as f:
        config = yaml.safe_load(f)
    
    if task not in config['tasks']:
        print(f"Error: Unknown task '{task}'. Available tasks: {list(config['tasks'].keys())}")
        return 1

    task_config = config['tasks'][task]
    
    results = {
        "timestamp": datetime.now().isoformat(),
        "task": task,
        "identifier": identifier,
        "overall_status": "PASS",
        "checks": []
    }
    
    for step in task_config['steps']:
        script_path = os.path.join(base_dir, step['script'])
        check_id = step['id']
        name = step['name']
        
        print(f"Running check: {name}...")
        
        try:
            # Pass identifier (e.g. branch name) to each check script
            result = subprocess.run(
                [sys.executable, script_path, identifier],
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
    report_path = os.path.join(base_dir, "reports", f"latest_{task}_results.json")
    with open(report_path, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"Evaluation complete for {task}. Status: {results['overall_status']}")
    print(f"Report saved to: {report_path}")
    
    return 0 if results["overall_status"] == "PASS" else 1

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python runner.py <task> <identifier>")
        print("Example: python runner.py submit feature/123-slug")
        print("Example: python runner.py cleanup feature/123-slug")
        sys.exit(1)
        
    task_name = sys.argv[1]
    task_id = sys.argv[2]
    sys.exit(run_eval(task_name, task_id))
