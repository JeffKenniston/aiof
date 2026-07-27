import subprocess

def run_cmd(cmd):
    print(f"Running command: {cmd}")
    res = subprocess.run(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    print(f"Exit code: {res.returncode}")
    print("STDOUT:")
    print(res.stdout.decode('utf-8', errors='replace'))
    print("STDERR:")
    print(res.stderr.decode('utf-8', errors='replace'))
    print("-" * 40)

run_cmd("go test ./...")
