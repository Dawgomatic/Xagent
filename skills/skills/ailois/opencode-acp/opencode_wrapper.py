#!/usr/bin/env python3
"""
OpenCode 协作包装器 - 主动监控并立即反馈结果
"""

import sys
import time
import subprocess
from runner_utils import build_client_command
from venv_utils import ensure_local_skill_venv

def run_opencode_with_monitoring(project_dir: str, task: str, timeout: int = 900):
    """运行 OpenCode 并主动监控进展"""
    
    # 启动 OpenCode
    print(f" 启动 OpenCode 任务...")
    print(f" 项目: {project_dir}")
    print(f" 任务: {task[:100]}...")
    print()
    
    cmd = build_client_command(project_dir=project_dir, task=task, timeout=timeout)
    
    # 同步执行，实时输出
    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        universal_newlines=True,
        bufsize=1
    )
    
    # 实时读取输出
    for line in process.stdout:
        print(line, end='', flush=True)
    
    # 等待完成
    return_code = process.wait()
    
    print()
    if return_code == 0:
        print(" OpenCode 任务完成！")
    else:
        print(f" OpenCode 任务失败（退出码: {return_code}）")
    
    return return_code

if __name__ == "__main__":
    ensure_local_skill_venv()

    if len(sys.argv) < 3:
        print("用法: python3 opencode_wrapper.py <project_dir> <task>")
        sys.exit(1)
    
    project_dir = sys.argv[1]
    task = sys.argv[2]
    timeout = int(sys.argv[3]) if len(sys.argv) > 3 else 900
    
    exit_code = run_opencode_with_monitoring(project_dir, task, timeout)
    sys.exit(exit_code)
