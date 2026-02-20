#!/usr/bin/env python3
"""Prompt Optimizer — Analyzes a draft prompt against best practices and suggests improvements.

Usage:
    python prompt_optimizer.py "Your draft prompt here"
    python prompt_optimizer.py --file path/to/prompt.txt
    echo "prompt" | python prompt_optimizer.py --stdin
"""

import sys
import argparse
import re
import json


# Best practices checklist with sources
CHECKS = [
    {
        "id": "clarity",
        "name": "Clarity & Specificity",
        "source": "Anthropic — Be clear and direct",
        "check": lambda p: _check_clarity(p),
        "weight": 2,
    },
    {
        "id": "examples",
        "name": "Few-Shot Examples",
        "source": "Anthropic — Multishot prompting; OpenAI — Provide examples",
        "check": lambda p: _check_examples(p),
        "weight": 1.5,
    },
    {
        "id": "structure",
        "name": "Structural Tags (XML/Markdown)",
        "source": "Anthropic — Use XML tags",
        "check": lambda p: _check_structure(p),
        "weight": 1.5,
    },
    {
        "id": "role",
        "name": "Role / Persona Assignment",
        "source": "Anthropic — System prompts; OpenAI — Adopt a persona",
        "check": lambda p: _check_role(p),
        "weight": 1,
    },
    {
        "id": "cot",
        "name": "Chain of Thought / Thinking",
        "source": "Anthropic — Let Claude think; OpenAI — Give time to think",
        "check": lambda p: _check_cot(p),
        "weight": 1,
    },
    {
        "id": "output_format",
        "name": "Output Format Specification",
        "source": "OpenAI — Specify desired format; Google — Structured output",
        "check": lambda p: _check_output_format(p),
        "weight": 1.5,
    },
    {
        "id": "context",
        "name": "Context Provided",
        "source": "Karpathy — Context Engineering; OpenAI — Reference text",
        "check": lambda p: _check_context(p),
        "weight": 1,
    },
    {
        "id": "constraints",
        "name": "Constraints & Boundaries",
        "source": "Anthropic — Be clear and direct; OpenAI — Clear instructions",
        "check": lambda p: _check_constraints(p),
        "weight": 1,
    },
    {
        "id": "length",
        "name": "Prompt Length Appropriateness",
        "source": "Context Engineering — Token efficiency",
        "check": lambda p: _check_length(p),
        "weight": 0.5,
    },
    {
        "id": "actionable",
        "name": "Actionable Instructions",
        "source": "Anthropic — Sequential steps; OpenAI — Specify steps",
        "check": lambda p: _check_actionable(p),
        "weight": 1,
    },
]


def _check_clarity(prompt: str) -> tuple[float, str, str]:
    """Returns (score 0-1, finding, suggestion)."""
    score = 0.5
    findings = []
    suggestions = []

    # Check for specific details
    if len(prompt) < 50:
        findings.append("Very short prompt — likely lacks specificity")
        suggestions.append("Add details: what the output is for, who the audience is, what success looks like")
        score -= 0.3
    
    # Check for vague language
    vague_words = ["something", "stuff", "things", "whatever", "etc", "somehow", "kind of", "sort of"]
    found_vague = [w for w in vague_words if w in prompt.lower()]
    if found_vague:
        findings.append(f"Vague language detected: {', '.join(found_vague)}")
        suggestions.append("Replace vague terms with specific descriptions")
        score -= 0.1 * len(found_vague)

    # Check for specific task verbs
    task_verbs = ["analyze", "generate", "write", "create", "extract", "classify", "summarize",
                  "compare", "evaluate", "translate", "convert", "list", "explain", "review"]
    has_verb = any(v in prompt.lower() for v in task_verbs)
    if has_verb:
        score += 0.2
    else:
        findings.append("No clear task verb found")
        suggestions.append("Start with a clear action verb: Analyze, Generate, Write, Extract, etc.")

    # Check for audience/purpose
    purpose_words = ["for", "purpose", "audience", "goal", "used for", "meant for", "intended"]
    if any(w in prompt.lower() for w in purpose_words):
        score += 0.2
        findings.append("✓ Purpose/audience specified")
    else:
        suggestions.append("Specify who the output is for and what it will be used for")

    score = max(0, min(1, score))
    return score, "; ".join(findings) if findings else "Reasonably clear", "; ".join(suggestions) if suggestions else "Good"


def _check_examples(prompt: str) -> tuple[float, str, str]:
    example_indicators = ["example", "<example", "for instance", "e.g.", "such as", "like this:", 
                          "input:", "output:", "sample"]
    count = sum(1 for ind in example_indicators if ind.lower() in prompt.lower())
    if count >= 3:
        return 1.0, "✓ Multiple examples detected", "Good"
    elif count >= 1:
        return 0.5, "Some example indicators found", "Add 2-3 more diverse examples for better consistency"
    return 0.0, "No examples found", "Add 3-5 diverse examples showing expected input→output pairs wrapped in <example> tags"


def _check_structure(prompt: str) -> tuple[float, str, str]:
    xml_tags = len(re.findall(r'<\w+>', prompt))
    has_headers = bool(re.search(r'^#{1,3}\s', prompt, re.MULTILINE))
    has_numbered = bool(re.search(r'^\d+[\.\)]\s', prompt, re.MULTILINE))
    has_bullets = bool(re.search(r'^[-*]\s', prompt, re.MULTILINE))
    
    structure_score = min(1.0, (xml_tags * 0.15) + (0.3 if has_headers else 0) + 
                         (0.2 if has_numbered else 0) + (0.2 if has_bullets else 0))
    
    findings = []
    suggestions = []
    if xml_tags > 0: findings.append(f"✓ {xml_tags} XML tags found")
    if has_headers: findings.append("✓ Headers used")
    if has_numbered: findings.append("✓ Numbered steps")
    
    if structure_score < 0.5:
        suggestions.append("Add XML tags (<instructions>, <context>, <examples>) to separate prompt components")
        suggestions.append("Use numbered steps for sequential instructions")
    
    return structure_score, "; ".join(findings) or "No structure", "; ".join(suggestions) or "Good"


def _check_role(prompt: str) -> tuple[float, str, str]:
    role_patterns = [r"you are a", r"you're a", r"act as", r"role:", r"persona:", r"as a \w+ expert",
                     r"you are an", r"system:", r"## role"]
    if any(re.search(p, prompt, re.IGNORECASE) for p in role_patterns):
        return 1.0, "✓ Role/persona assigned", "Good"
    return 0.0, "No role assigned", "Add a role: 'You are a [specific expert] specializing in [domain]'"


def _check_cot(prompt: str) -> tuple[float, str, str]:
    cot_patterns = [r"step.by.step", r"think", r"<thinking>", r"reason", r"chain of thought",
                    r"work through", r"break down", r"analyze.*before"]
    if any(re.search(p, prompt, re.IGNORECASE) for p in cot_patterns):
        return 1.0, "✓ Chain-of-thought prompting detected", "Good"
    if len(prompt) > 500:  # Complex prompts benefit more from CoT
        return 0.0, "No CoT for complex prompt", "Add 'Think step-by-step' or use <thinking>/<answer> tags for complex reasoning"
    return 0.3, "Simple prompt — CoT may not be needed", "Consider adding CoT if task requires reasoning"


def _check_output_format(prompt: str) -> tuple[float, str, str]:
    format_patterns = [r"output format", r"format:", r"respond with", r"output.*json", r"return.*as",
                       r"structure:", r"template:", r"respond in", r"output only", r"## output",
                       r"```", r"table", r"\bJSON\b", r"markdown", r"bullet"]
    matches = sum(1 for p in format_patterns if re.search(p, prompt, re.IGNORECASE))
    if matches >= 2:
        return 1.0, "✓ Output format well-specified", "Good"
    elif matches == 1:
        return 0.5, "Partial format specification", "Be more explicit: show exact output structure or provide a template"
    return 0.0, "No output format specified", "Add explicit output format: JSON schema, template, or 'Output format:' section"


def _check_context(prompt: str) -> tuple[float, str, str]:
    context_patterns = [r"<context", r"background:", r"context:", r"given that", r"based on",
                        r"here is", r"the following", r"## context", r"<data", r"<document"]
    matches = sum(1 for p in context_patterns if re.search(p, prompt, re.IGNORECASE))
    if matches >= 2:
        return 1.0, "✓ Context provided", "Good"
    elif matches == 1:
        return 0.5, "Some context provided", "Add more background: what workflow this is part of, relevant constraints"
    return 0.0, "No explicit context", "Add context about the task's purpose, audience, and environment"


def _check_constraints(prompt: str) -> tuple[float, str, str]:
    constraint_patterns = [r"do not", r"don't", r"never", r"must not", r"avoid", r"constraint",
                           r"requirement", r"rule:", r"must:", r"## rules", r"## constraints",
                           r"important:", r"critical:"]
    matches = sum(1 for p in constraint_patterns if re.search(p, prompt, re.IGNORECASE))
    if matches >= 2:
        return 1.0, "✓ Constraints defined", "Good"
    elif matches == 1:
        return 0.5, "Minimal constraints", "Add explicit constraints: what to avoid, boundaries, must-haves"
    return 0.2, "No constraints found", "Consider adding constraints: 'Do NOT...', 'MUST include...', 'NEVER...'"


def _check_length(prompt: str) -> tuple[float, str, str]:
    words = len(prompt.split())
    if words < 10:
        return 0.2, f"Very short ({words} words)", "Expand with context, examples, and format specification"
    elif words < 30:
        return 0.5, f"Brief ({words} words)", "Consider adding more detail unless task is simple"
    elif words < 500:
        return 1.0, f"✓ Good length ({words} words)", "Good"
    elif words < 2000:
        return 0.8, f"Long ({words} words)", "Ensure all content is necessary; consider if any can be moved to retrieval"
    return 0.5, f"Very long ({words} words)", "Risk of diluting key instructions; consider chunking or prompt chaining"


def _check_actionable(prompt: str) -> tuple[float, str, str]:
    numbered = len(re.findall(r'^\d+[\.\)]\s', prompt, re.MULTILINE))
    bullets = len(re.findall(r'^[-*]\s', prompt, re.MULTILINE))
    imperatives = ["analyze", "write", "create", "generate", "list", "extract", "identify",
                   "compare", "evaluate", "summarize", "classify", "review", "explain"]
    imp_count = sum(1 for imp in imperatives if imp in prompt.lower())
    
    score = min(1.0, (numbered * 0.15) + (bullets * 0.1) + (imp_count * 0.15))
    if score >= 0.7:
        return score, f"✓ Actionable ({numbered} numbered steps, {imp_count} action verbs)", "Good"
    elif score >= 0.3:
        return score, f"Partially actionable ({numbered} steps)", "Add more numbered steps or explicit action verbs"
    return score, "Not actionable", "Use numbered steps: 1. Do X, 2. Then Y, 3. Output Z"


def analyze_prompt(prompt: str) -> dict:
    """Analyze a prompt against all best practices."""
    results = []
    total_score = 0
    total_weight = 0
    
    for check in CHECKS:
        score, finding, suggestion = check["check"](prompt)
        results.append({
            "id": check["id"],
            "name": check["name"],
            "source": check["source"],
            "score": round(score, 2),
            "weight": check["weight"],
            "finding": finding,
            "suggestion": suggestion,
        })
        total_score += score * check["weight"]
        total_weight += check["weight"]
    
    overall = round(total_score / total_weight * 10, 1)
    
    # Sort by score ascending (worst first)
    results.sort(key=lambda r: r["score"])
    
    return {
        "overall_score": overall,
        "max_score": 10.0,
        "grade": _grade(overall),
        "checks": results,
        "top_improvements": [r for r in results if r["score"] < 0.7][:3],
        "prompt_length_words": len(prompt.split()),
        "prompt_length_chars": len(prompt),
    }


def _grade(score: float) -> str:
    if score >= 9: return "A+"
    if score >= 8: return "A"
    if score >= 7: return "B+"
    if score >= 6: return "B"
    if score >= 5: return "C+"
    if score >= 4: return "C"
    if score >= 3: return "D"
    return "F"


def format_report(analysis: dict, prompt: str) -> str:
    """Format analysis as a readable report."""
    lines = []
    lines.append("=" * 60)
    lines.append(f"  PROMPT ANALYSIS REPORT — Score: {analysis['overall_score']}/10 ({analysis['grade']})")
    lines.append("=" * 60)
    lines.append(f"  Length: {analysis['prompt_length_words']} words / {analysis['prompt_length_chars']} chars")
    lines.append("")
    
    lines.append("  DETAILED SCORES:")
    lines.append("  " + "-" * 56)
    for check in analysis["checks"]:
        bar = "█" * int(check["score"] * 10) + "░" * (10 - int(check["score"] * 10))
        lines.append(f"  {bar} {check['score']:.1f}  {check['name']}")
        lines.append(f"         → {check['finding']}")
        if check["score"] < 0.7:
            lines.append(f"         ★ {check['suggestion']}")
        lines.append("")
    
    if analysis["top_improvements"]:
        lines.append("  TOP 3 IMPROVEMENTS:")
        lines.append("  " + "-" * 56)
        for i, imp in enumerate(analysis["top_improvements"], 1):
            lines.append(f"  {i}. [{imp['name']}] {imp['suggestion']}")
            lines.append(f"     Source: {imp['source']}")
            lines.append("")
    
    lines.append("=" * 60)
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Analyze and optimize prompts against best practices")
    parser.add_argument("prompt", nargs="?", help="Draft prompt to analyze")
    parser.add_argument("--file", "-f", help="Read prompt from file")
    parser.add_argument("--stdin", action="store_true", help="Read prompt from stdin")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    args = parser.parse_args()
    
    if args.file:
        with open(args.file) as f:
            prompt = f.read()
    elif args.stdin or not sys.stdin.isatty():
        prompt = sys.stdin.read()
    elif args.prompt:
        prompt = args.prompt
    else:
        parser.print_help()
        sys.exit(1)
    
    prompt = prompt.strip()
    if not prompt:
        print("Error: Empty prompt")
        sys.exit(1)
    
    analysis = analyze_prompt(prompt)
    
    if args.json:
        print(json.dumps(analysis, indent=2))
    else:
        print(format_report(analysis, prompt))


if __name__ == "__main__":
    main()
