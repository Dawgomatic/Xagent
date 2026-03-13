#!/usr/bin/env python3
"""
FIS 3.2.0 SubAgent 完整生命周期管理
自动化：Ticket创建 → 工牌生成 → Spawn → 完成归档 → 交付物收集
"""

import json
import os
import sys
import subprocess
import shutil
from datetime import datetime
from pathlib import Path

# 路径配置
WORKSPACE = Path.home() / ".openclaw" / "workspace"
SHARED_HUB = Path.home() / ".openclaw" / "fis-hub"
BADGE_GENERATOR = WORKSPACE / "skills" / "fis-architecture" / "lib" / "badge_generator_v7.py"
TICKETS_DIR = SHARED_HUB / "tickets"
RESULTS_DIR = SHARED_HUB / "results"

class SubAgentLifecycle:
    """FIS 3.2.0 子代理生命周期管理器"""
    
    def __init__(self, parent_agent="cybermao"):
        self.parent = parent_agent
        self.output_formats = ['md', 'json', 'txt', 'py', 'png', 'pdf']
    
    def create_task(self, agent_name, task_desc, role="worker", 
                   output_requirements=None, deadline_days=1):
        """
        创建完整任务包
        
        Args:
            output_requirements: ["技术报告.md", "代码.py", "结果图.png"]
        """
        timestamp = datetime.now()
        ticket_id = f"TASK_{self.parent.upper()}_{timestamp.strftime('%Y%m%d_%H%M%S')}_{agent_name}"
        
        # 构建任务数据结构
        task_package = {
            "ticket_id": ticket_id,
            "agent_id": agent_name,
            "parent": self.parent,
            "role": role,
            "task": {
                "description": task_desc,
                "created_at": timestamp.isoformat(),
                "deadline": (timestamp.replace(day=timestamp.day + deadline_days)).isoformat(),
                "status": "pending"
            },
            "output_requirements": output_requirements or ["report.md"],
            "deliverables": [],  # 完成后填写
            "workspace": f"workspace-{agent_name.lower()}",
            "badge_path": None,
            "completed_at": None
        }
        
        # 保存 Ticket
        ticket_path = TICKETS_DIR / "active" / f"{ticket_id}.json"
        ticket_path.write_text(json.dumps(task_package, indent=2, ensure_ascii=False))
        
        # 生成工牌
        badge_path = self._generate_badge(agent_name, role, task_desc, 
                                          task_package["output_requirements"])
        
        # 更新 ticket 记录工牌路径
        task_package["badge_path"] = str(badge_path)
        ticket_path.write_text(json.dumps(task_package, indent=2, ensure_ascii=False))
        
        # 自动发送工牌到 WhatsApp
        self._send_badge_whatsapp(badge_path, agent_name, ticket_id)
        
        print(f" Task created: {ticket_id}")
        print(f" Ticket: {ticket_path}")
        print(f" Badge: {badge_path}")
        print(f" Output requirements: {task_package['output_requirements']}")
        
        return ticket_id, task_package
    
    def _generate_badge(self, agent_name, role, task_desc, requirements):
        """生成工牌"""
        if not BADGE_GENERATOR.exists():
            print(f" Badge generator not found")
            return None
        
        badge_script = f"""
import sys
sys.path.insert(0, '{BADGE_GENERATOR.parent}')
from badge_generator_v7 import generate_badge_with_task

req_list = {requirements}
output = generate_badge_with_task(
    agent_name='{agent_name}',
    role='{role}',
    task_desc='{task_desc[:50]}',
    task_requirements=req_list[:3] if len(req_list) > 3 else req_list,
    output_dir=None
)
print(output)
"""
        try:
            result = subprocess.run(
                [sys.executable, "-c", badge_script],
                capture_output=True, text=True, timeout=30
            )
            # 解析输出路径
            for line in result.stdout.split('\n'):
                if 'Badge:' in line or '/output/badges/' in line:
                    return line.strip()
            return result.stdout.strip()
        except Exception as e:
            print(f" Badge generation error: {e}")
            return None
    
    def _send_badge_whatsapp(self, badge_path, agent_name, ticket_id):
        """自动发送工牌到 WhatsApp"""
        if not badge_path:
            return
        
        # 清理路径（移除可能的 " Badge: " 前缀）
        badge_str = str(badge_path).replace(' Badge: ', '').strip()
        src = Path(badge_str)
        
        # 如果路径是相对路径或包含环境变量，尝试解析
        if not src.exists() and 'openclaw' in badge_str:
            # 尝试从 home 目录构建完整路径
            src = Path.home() / badge_str.replace('/home/muselinn/', '').replace('/home/user/', '')
        
        if not src.exists():
            print(f" Badge file not found: {badge_str}")
            # 尝试在常见位置查找
            alt_paths = [
                Path.home() / ".openclaw" / "output" / "badges" / f"badge_v7_{agent_name}.png",
                Path.home() / ".openclaw" / "output" / "badges" / f"badge_v7_CYBERMAO-SA-*.png"
            ]
            for alt in alt_paths:
                if alt.exists():
                    src = alt
                    print(f"   Found alternative: {src}")
                    break
            else:
                return
        
        # WhatsApp 允许的发送目录
        allowed_dir = WORKSPACE / "output"
        allowed_dir.mkdir(parents=True, exist_ok=True)
        
        # 使用更短的文件名
        dst = allowed_dir / f"badge_{ticket_id.split('_')[-1][:20]}.png"
        try:
            shutil.copy2(src, dst)
            print(f" Badge ready for WhatsApp: {dst.name}")
        except Exception as e:
            print(f" Failed to copy badge: {e}")
            return
        
        # 生成发送命令（供外部调用或自动执行）
        caption = f" 新任务工牌\\nAgent: {agent_name}\\nTicket: {ticket_id[:40]}..."
        
        # 尝试使用 openclaw CLI 发送
        try:
            send_cmd = [
                "openclaw", "message", "send",
                "--channel", "whatsapp",
                "--target", "+8618009073880",
                "--media", str(dst),
                "--message", caption
            ]
            result = subprocess.run(send_cmd, capture_output=True, text=True, timeout=30)
            if result.returncode == 0:
                print(f" Badge sent to WhatsApp!")
            else:
                print(f" WhatsApp send: openclaw message send --channel whatsapp --target +8618009073880 --media {dst} --message \"{caption}\"")
        except Exception as e:
            print(f" To send: openclaw message send --channel whatsapp --target +8618009073880 --media {dst} --message \"{caption}\"")
    
    def verify_deliverables(self, ticket_id):
        """
        验证交付物是否完整
        检查子代理工作区的 output/ 目录
        """
        ticket_path = TICKETS_DIR / "active" / f"{ticket_id}.json"
        if not ticket_path.exists():
            print(f" Ticket not found: {ticket_id}")
            return False
        
        task = json.loads(ticket_path.read_text())
        agent_name = task["agent_id"]
        requirements = task.get("output_requirements", [])
        
        # 检查子代理工作区 (OpenClaw 创建的 workspace-xxx 在 .openclaw/ 下)
        agent_workspace = WORKSPACE.parent / f"workspace-{agent_name.lower()}"
        # 兼容两种可能的路径
        if not agent_workspace.exists():
            alt_workspace = WORKSPACE.parent / "workspace" / f"workspace-{agent_name.lower()}"
            if alt_workspace.exists():
                agent_workspace = alt_workspace
        output_dir = agent_workspace / "output"
        
        found_files = []
        missing_files = []
        
        if output_dir.exists():
            for req in requirements:
                # 模糊匹配，不要求完全匹配文件名
                req_base = req.replace('.md', '').replace('.py', '').replace('.png', '')
                matched = False
                
                for f in output_dir.iterdir():
                    if req_base.lower() in f.name.lower() or req in f.name:
                        found_files.append(str(f))
                        matched = True
                        break
                
                if not matched:
                    missing_files.append(req)
        else:
            missing_files = requirements
        
        print(f"\n Deliverables check for {ticket_id}:")
        print(f"    Found: {len(found_files)}/{len(requirements)}")
        for f in found_files:
            print(f"      - {Path(f).name}")
        if missing_files:
            print(f"    Missing: {missing_files}")
        
        return len(missing_files) == 0, found_files, missing_files
    
    def complete_task(self, ticket_id, auto_collect=True):
        """
        完成任务：归档 Ticket + 收集交付物
        
        Args:
            auto_collect: 是否自动收集交付物到 results/
        """
        active_path = TICKETS_DIR / "active" / f"{ticket_id}.json"
        completed_path = TICKETS_DIR / "completed" / f"{ticket_id}.json"
        
        if not active_path.exists():
            print(f" Ticket not found in active: {ticket_id}")
            return False
        
        task = json.loads(active_path.read_text())
        
        # 验证交付物
        is_complete, found_files, missing = self.verify_deliverables(ticket_id)
        
        if not is_complete:
            print(f" Task {ticket_id} has missing deliverables!")
            response = input("   Force complete? (y/N): ")
            if response.lower() != 'y':
                return False
        
        # 收集交付物
        if auto_collect and found_files:
            result_dir = self._collect_deliverables(ticket_id, task["agent_id"], found_files)
            task["deliverables"] = [str(f) for f in found_files]
            task["result_directory"] = str(result_dir)
        
        # 更新状态
        task["status"] = "completed"
        task["completed_at"] = datetime.now().isoformat()
        task["verification"] = {
            "all_deliverables_present": is_complete,
            "missing": missing
        }
        
        # 移动到 completed
        completed_path.write_text(json.dumps(task, indent=2, ensure_ascii=False))
        active_path.unlink()
        
        print(f"\n Task completed: {ticket_id}")
        print(f" Archived to: {completed_path}")
        if auto_collect and found_files:
            print(f" Deliverables collected to: {result_dir}")
        
        return True
    
    def _collect_deliverables(self, ticket_id, agent_name, files):
        """收集交付物到 results/ 目录"""
        # 创建结果目录
        result_dir = RESULTS_DIR / ticket_id
        result_dir.mkdir(parents=True, exist_ok=True)
        
        # 复制文件
        for src in files:
            src_path = Path(src)
            if src_path.exists():
                dst = result_dir / src_path.name
                shutil.copy2(src_path, dst)
        
        # 创建索引文件
        index = {
            "ticket_id": ticket_id,
            "agent": agent_name,
            "completed_at": datetime.now().isoformat(),
            "files": [Path(f).name for f in files]
        }
        (result_dir / "INDEX.json").write_text(json.dumps(index, indent=2))
        
        return result_dir
    
    def list_active(self):
        """列出活跃任务"""
        active_dir = TICKETS_DIR / "active"
        tickets = list(active_dir.glob("*.json"))
        
        print(f"\n Active Tasks ({len(tickets)}):")
        for t in tickets:
            task = json.loads(t.read_text())
            print(f"   • {task['ticket_id'][:50]}... [{task['role']}] {task['task']['description'][:30]}")
        
        return tickets


def main():
    import argparse
    
    parser = argparse.ArgumentParser(description="FIS 3.2.0 SubAgent Lifecycle")
    subparsers = parser.add_subparsers(dest='command')
    
    # create 命令
    create_parser = subparsers.add_parser('create', help='Create new task')
    create_parser.add_argument('--agent', required=True, help='Agent name')
    create_parser.add_argument('--task', required=True, help='Task description')
    create_parser.add_argument('--role', default='worker', choices=['worker', 'researcher', 'reviewer', 'formatter'])
    create_parser.add_argument('--outputs', nargs='+', default=['report.md'], help='Required output files')
    create_parser.add_argument('--deadline', type=int, default=1, help='Deadline in days')
    
    # verify 命令
    verify_parser = subparsers.add_parser('verify', help='Verify deliverables')
    verify_parser.add_argument('--ticket-id', required=True, help='Ticket ID')
    
    # complete 命令
    complete_parser = subparsers.add_parser('complete', help='Complete task')
    complete_parser.add_argument('--ticket-id', required=True, help='Ticket ID')
    complete_parser.add_argument('--no-collect', action='store_true', help='Skip collecting deliverables')
    
    # list 命令
    subparsers.add_parser('list', help='List active tasks')
    
    args = parser.parse_args()
    
    lifecycle = SubAgentLifecycle()
    
    if args.command == 'create':
        ticket_id, task = lifecycle.create_task(
            args.agent, args.task, args.role, args.outputs, args.deadline
        )
        print(f"\n Ready to spawn:")
        print(f"   sessions_spawn(task='{args.task}', label='{args.agent}')")
        print(f"\n   After completion, run:")
        print(f"   fis_lifecycle complete --ticket-id {ticket_id}")
    
    elif args.command == 'verify':
        lifecycle.verify_deliverables(args.ticket_id)
    
    elif args.command == 'complete':
        lifecycle.complete_task(args.ticket_id, not args.no_collect)
    
    elif args.command == 'list':
        lifecycle.list_active()
    
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
