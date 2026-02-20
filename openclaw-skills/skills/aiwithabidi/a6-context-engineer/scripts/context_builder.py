#!/usr/bin/env python3
"""Context Builder — Builds optimal context windows for LLM tasks.

Takes a task description and generates a structured prompt with system prompt,
user prompt, and examples ready for use.

Usage:
    python context_builder.py "Analyze quarterly financials and produce board summary"
    python context_builder.py --task "Code review" --role "Senior engineer" --output json
    python context_builder.py --task "Classify support tickets" --examples 3 --cot
"""

import sys
import argparse
import json
import textwrap


# Task type detection patterns and their optimal prompt structures
TASK_PROFILES = {
    "analysis": {
        "keywords": ["analyze", "analysis", "assess", "evaluate", "review", "audit", "examine", "investigate"],
        "recommended_role": "Senior analyst specializing in {domain}",
        "recommended_structure": ["role", "context", "data", "instructions", "cot", "output_format"],
        "cot_recommended": True,
        "examples_recommended": 1,
        "description": "Analytical task requiring structured reasoning",
    },
    "generation": {
        "keywords": ["write", "create", "generate", "draft", "compose", "produce", "build", "make"],
        "recommended_role": "Expert {domain} writer",
        "recommended_structure": ["role", "context", "instructions", "examples", "constraints", "output_format"],
        "cot_recommended": False,
        "examples_recommended": 2,
        "description": "Content generation task",
    },
    "extraction": {
        "keywords": ["extract", "parse", "pull", "find", "identify", "detect", "locate", "scrape"],
        "recommended_role": "Data extraction specialist",
        "recommended_structure": ["instructions", "data", "examples", "output_format"],
        "cot_recommended": False,
        "examples_recommended": 3,
        "description": "Data extraction task — examples are critical",
    },
    "classification": {
        "keywords": ["classify", "categorize", "label", "tag", "sort", "triage", "bucket", "assign"],
        "recommended_role": "Classification expert for {domain}",
        "recommended_structure": ["role", "categories", "examples", "data", "output_format"],
        "cot_recommended": False,
        "examples_recommended": 3,
        "description": "Classification task — examples of each category needed",
    },
    "reasoning": {
        "keywords": ["decide", "recommend", "choose", "compare", "solve", "reason", "determine", "figure"],
        "recommended_role": "Senior {domain} advisor",
        "recommended_structure": ["role", "context", "instructions", "cot", "output_format"],
        "cot_recommended": True,
        "examples_recommended": 1,
        "description": "Reasoning/decision task — CoT is essential",
    },
    "summarization": {
        "keywords": ["summarize", "summary", "condense", "brief", "recap", "tldr", "overview", "digest"],
        "recommended_role": "Expert summarizer",
        "recommended_structure": ["role", "instructions", "data", "constraints", "output_format"],
        "cot_recommended": False,
        "examples_recommended": 1,
        "description": "Summarization task",
    },
    "code": {
        "keywords": ["code", "function", "implement", "program", "script", "debug", "fix", "refactor"],
        "recommended_role": "Senior software engineer specializing in {domain}",
        "recommended_structure": ["role", "context", "instructions", "examples", "constraints", "output_format"],
        "cot_recommended": True,
        "examples_recommended": 1,
        "description": "Coding task",
    },
    "agent": {
        "keywords": ["agent", "sub-agent", "subagent", "cron", "automated", "skill", "system prompt"],
        "recommended_role": "Specialized {domain} agent",
        "recommended_structure": ["identity", "rules", "context", "tools", "instructions", "constraints", "output_format"],
        "cot_recommended": False,
        "examples_recommended": 0,
        "description": "Agent/system prompt design",
    },
}


def detect_task_type(task: str) -> tuple[str, dict]:
    """Detect the task type from the description."""
    task_lower = task.lower()
    best_match = "generation"  # default
    best_score = 0
    
    for task_type, profile in TASK_PROFILES.items():
        score = sum(1 for kw in profile["keywords"] if kw in task_lower)
        if score > best_score:
            best_score = score
            best_match = task_type
    
    return best_match, TASK_PROFILES[best_match]


def infer_domain(task: str) -> str:
    """Try to infer the domain from the task description."""
    domains = {
        "financial": ["financial", "finance", "revenue", "budget", "accounting", "investment"],
        "legal": ["legal", "contract", "compliance", "regulation", "law", "policy"],
        "technical": ["code", "software", "API", "database", "system", "infrastructure"],
        "marketing": ["marketing", "campaign", "brand", "content", "social media", "SEO"],
        "data": ["data", "analytics", "metrics", "dashboard", "report", "statistics"],
        "medical": ["medical", "health", "clinical", "patient", "diagnosis"],
        "customer": ["customer", "support", "ticket", "feedback", "service"],
    }
    
    task_lower = task.lower()
    for domain, keywords in domains.items():
        if any(kw in task_lower for kw in keywords):
            return domain
    return "the relevant domain"


def build_context(task: str, role: str = None, num_examples: int = None, 
                  use_cot: bool = None, output_format: str = "text") -> dict:
    """Build an optimal context window for the given task."""
    
    task_type, profile = detect_task_type(task)
    domain = infer_domain(task)
    
    # Determine parameters
    if role is None:
        role = profile["recommended_role"].format(domain=domain)
    if num_examples is None:
        num_examples = profile["examples_recommended"]
    if use_cot is None:
        use_cot = profile["cot_recommended"]
    
    # Build the prompt sections
    sections = {}
    
    # System prompt
    sections["system_prompt"] = f"You are a {role}."
    
    # Build user prompt
    user_parts = []
    
    for section in profile["recommended_structure"]:
        if section == "role":
            continue  # handled in system prompt
        elif section == "identity":
            user_parts.append(f"## Identity\nYou are a {role}.\n")
        elif section == "context":
            user_parts.append("## Context\n{{CONTEXT — Add background information: what this is for, "
                            "who the audience is, what workflow this is part of}}\n")
        elif section == "data":
            user_parts.append("<data>\n{{DATA — Insert the data/document to process}}\n</data>\n")
        elif section == "instructions":
            user_parts.append(f"## Task\n{task}\n")
            user_parts.append("## Instructions\n1. {{STEP_1 — First action}}\n"
                            "2. {{STEP_2 — Second action}}\n"
                            "3. {{STEP_3 — Third action}}\n")
        elif section == "categories":
            user_parts.append("## Categories\nClassify into one of: {{CATEGORY_1}}, {{CATEGORY_2}}, {{CATEGORY_3}}\n")
        elif section == "tools":
            user_parts.append("## Available Tools\n- {{TOOL_1}}: {{DESCRIPTION}}\n- {{TOOL_2}}: {{DESCRIPTION}}\n")
        elif section == "rules":
            user_parts.append("## Rules\n- {{RULE_1}}\n- {{RULE_2}}\n- NEVER {{PROHIBITED_ACTION}}\n")
        elif section == "examples":
            if num_examples > 0:
                example_block = "<examples>\n"
                for i in range(1, num_examples + 1):
                    example_block += f"<example>\nInput: {{{{EXAMPLE_{i}_INPUT}}}}\nOutput: {{{{EXAMPLE_{i}_OUTPUT}}}}\n</example>\n"
                example_block += "</examples>\n"
                user_parts.append(example_block)
        elif section == "cot":
            if use_cot:
                user_parts.append("Think step-by-step in <thinking> tags before providing your answer in <answer> tags.\n")
        elif section == "constraints":
            user_parts.append("## Constraints\n- {{CONSTRAINT_1}}\n- {{CONSTRAINT_2}}\n- Do NOT {{PROHIBITION}}\n")
        elif section == "output_format":
            user_parts.append("## Output Format\n{{SPECIFY — exact format, JSON schema, template, or structure}}\n")
    
    user_prompt = "\n".join(user_parts)
    
    # Build recommendations
    recommendations = []
    recommendations.append(f"Task type detected: {task_type} — {profile['description']}")
    recommendations.append(f"Recommended role: {role}")
    if use_cot:
        recommendations.append("✓ Chain-of-thought enabled (recommended for this task type)")
    else:
        recommendations.append("○ Chain-of-thought not needed for this task type (add --cot to enable)")
    recommendations.append(f"Recommended examples: {num_examples} (add --examples N to adjust)")
    
    # Context budget estimation
    budget = {
        "system_prompt": "~50-200 tokens (5%)",
        "user_instructions": "~200-500 tokens (10-15%)",
        "context_data": "~variable (30-50%)",
        "examples": f"~{num_examples * 200} tokens ({num_examples * 5}%)" if num_examples else "0 tokens",
        "thinking_space": "~500-2000 tokens (10-20%)" if use_cot else "N/A",
        "output_reserve": "~20-30% — ALWAYS reserve this",
    }
    
    return {
        "task_type": task_type,
        "domain": domain,
        "system_prompt": sections["system_prompt"],
        "user_prompt": user_prompt,
        "recommendations": recommendations,
        "context_budget": budget,
        "structure": profile["recommended_structure"],
    }


def format_output(result: dict, output_format: str) -> str:
    """Format the result for display."""
    if output_format == "json":
        return json.dumps(result, indent=2)
    
    lines = []
    lines.append("=" * 60)
    lines.append(f"  CONTEXT BUILDER — {result['task_type'].upper()} TASK")
    lines.append("=" * 60)
    lines.append("")
    
    lines.append("  RECOMMENDATIONS:")
    for rec in result["recommendations"]:
        lines.append(f"    • {rec}")
    lines.append("")
    
    lines.append("  CONTEXT BUDGET:")
    for k, v in result["context_budget"].items():
        lines.append(f"    {k}: {v}")
    lines.append("")
    
    lines.append("-" * 60)
    lines.append("  SYSTEM PROMPT:")
    lines.append("-" * 60)
    lines.append(result["system_prompt"])
    lines.append("")
    
    lines.append("-" * 60)
    lines.append("  USER PROMPT TEMPLATE:")
    lines.append("-" * 60)
    lines.append(result["user_prompt"])
    lines.append("")
    lines.append("=" * 60)
    lines.append("  Replace all {{PLACEHOLDERS}} with your actual content.")
    lines.append("=" * 60)
    
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Build optimal context windows for LLM tasks")
    parser.add_argument("task", nargs="?", help="Task description")
    parser.add_argument("--task", "-t", dest="task_flag", help="Task description (alternative)")
    parser.add_argument("--role", "-r", help="Override role/persona")
    parser.add_argument("--examples", "-e", type=int, help="Number of examples to include")
    parser.add_argument("--cot", action="store_true", help="Force chain-of-thought")
    parser.add_argument("--output", "-o", choices=["text", "json"], default="text", help="Output format")
    args = parser.parse_args()
    
    task = args.task or args.task_flag
    if not task:
        parser.print_help()
        sys.exit(1)
    
    result = build_context(task, role=args.role, num_examples=args.examples, 
                          use_cot=args.cot if args.cot else None, output_format=args.output)
    
    print(format_output(result, args.output))


if __name__ == "__main__":
    main()
