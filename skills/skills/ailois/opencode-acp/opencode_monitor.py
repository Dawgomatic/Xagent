#!/usr/bin/env python3
"""
OpenCode 协作助手 - 自动监控并立即反馈结果
"""

import sys
import time
import subprocess
import threading
from runner_utils import build_client_command
from venv_utils import ensure_local_skill_venv

class OpenCodeMonitor:
    def __init__(self, project_dir: str, task: str, timeout: int = 900):
        self.project_dir = project_dir
        self.task = task
        self.timeout = timeout
        self.completed = False
        self.result = None
        
    def run_opencode(self):
        """在后台运行 OpenCode"""
        cmd = build_client_command(
            project_dir=self.project_dir,
            task=self.task,
            timeout=self.timeout,
        )
        
        try:
            result = subprocess.run(
                cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                universal_newlines=True,
                timeout=self.timeout
            )
            self.result = {
                'stdout': result.stdout,
                'stderr': "",
                'returncode': result.returncode
            }
            self.completed = True
        except subprocess.TimeoutExpired:
            self.result = {'error': 'timeout'}
            self.completed = True
        except Exception as e:
            self.result = {'error': str(e)}
            self.completed = True
    
    def monitor_and_report(self):
        """监控 OpenCode 并立即报告结果"""
        print(f" 启动 OpenCode 任务...")
        print(f" 项目: {self.project_dir}")
        print(f" 任务: {self.task[:100]}...")
        print()
        
        # 在后台线程运行 OpenCode
        thread = threading.Thread(target=self.run_opencode)
        thread.daemon = True
        thread.start()
        
        # 主动轮询检查完成状态
        start_time = time.time()
        last_check = start_time
        
        while not self.completed:
            time.sleep(0.5)  # 每 0.5 秒检查一次
            
            # 每 10 秒显示一次进度
            current_time = time.time()
            if current_time - last_check >= 10:
                elapsed = int(current_time - start_time)
                print(f"  运行中... ({elapsed}秒)")
                last_check = current_time
            
            # 超时检查
            if current_time - start_time > self.timeout:
                print(f" 超时（{self.timeout}秒）")
                return False
        
        # 任务完成，立即报告结果
        elapsed = int(time.time() - start_time)
        print(f"\n OpenCode 任务完成！（用时 {elapsed}秒）")
        print()
        
        if self.result:
            if 'error' in self.result:
                print(f" 错误: {self.result['error']}")
                return False
            
            # 显示输出
            if self.result.get('stdout'):
                print(self.result['stdout'])
            
            if self.result.get('stderr'):
                print("错误输出:", file=sys.stderr)
                print(self.result['stderr'], file=sys.stderr)
            
            return self.result['returncode'] == 0
        
        return False

if __name__ == "__main__":
    ensure_local_skill_venv()

    if len(sys.argv) < 3:
        print("用法: python3 opencode_monitor.py <project_dir> <task> [timeout]")
        sys.exit(1)
    
    project_dir = sys.argv[1]
    task = sys.argv[2]
    timeout = int(sys.argv[3]) if len(sys.argv) > 3 else 900
    
    monitor = OpenCodeMonitor(project_dir, task, timeout)
    success = monitor.monitor_and_report()
    
    sys.exit(0 if success else 1)
