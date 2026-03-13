#!/usr/bin/env python3
"""
OpenCode 协作助手 - 实时输出并立即反馈结果
"""

import sys
import subprocess
from runner_utils import build_client_command
from venv_utils import ensure_local_skill_venv

def run_opencode_with_realtime_output(project_dir: str, task: str, timeout: int = 900):
    """运行 OpenCode 并实时显示输出"""
    
    print(f" 启动 OpenCode 任务...")
    print(f" 项目: {project_dir}")
    print(f" 任务: {task[:100]}...")
    print()
    
    cmd = build_client_command(project_dir=project_dir, task=task, timeout=timeout)
    
    # 使用 Popen 实时输出
    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        universal_newlines=True,
        bufsize=1
    )
    
    # 实时读取并显示输出
    for line in process.stdout:
        print(line, end='', flush=True)
    
    # 等待完成
    returncode = process.wait()
    
    print()
    if returncode == 0:
        print(" OpenCode 任务完成！")
    else:
        print(f" OpenCode 任务失败（退出码: {returncode}）")
    
    return returncode == 0

if __name__ == "__main__":
    ensure_local_skill_venv()

    if len(sys.argv) < 3:
        print("用法: python3 opencode_realtime.py <project_dir> <task> [timeout]")
        sys.exit(1)
    
    project_dir = sys.argv[1]
    task = sys.argv[2]
    timeout = int(sys.argv[3]) if len(sys.argv) > 3 else 900
    
    success = run_opencode_with_realtime_output(project_dir, task, timeout)
    sys.exit(0 if success else 1)
