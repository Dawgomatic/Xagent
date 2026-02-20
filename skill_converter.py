#!/usr/bin/env python3
"""
OpenClaw → Xagent Skill Converter

Converts skills from the skills archive into Xagent-compatible
format and installs them into the Xagent workspace skills directory.

Usage:
    # List all available skills
    python3 skill_converter.py list

    # Search skills by keyword
    python3 skill_converter.py search "docker"

    # Convert + install a single skill
    python3 skill_converter.py install <owner>/<slug>

    # Convert + install all skills from a publisher
    python3 skill_converter.py install-publisher <owner>

    # Bulk convert top skills (by name relevance)
    python3 skill_converter.py install-curated

    # Show info about a skill before installing
    python3 skill_converter.py info <owner>/<slug>
"""

import argparse
import json
import os
import platform
import re
import shutil
import subprocess
import sys
from pathlib import Path

OPENCLAW_SKILLS_DIR = Path(__file__).parent / "skills" / "skills"
XAGENT_WORKSPACE_SKILLS = Path(__file__).parent / "workspace" / "skills"
XAGENT_GLOBAL_SKILLS = Path.home() / ".xagent" / "skills"

SKIP_FILES = {"_meta.json", ".clawhub", ".clawdhub", ".clawhubignore", ".clawdhubignore"}

OPENCLAW_METADATA_KEYS = ("openclaw", "clawdbot", "clawdis", "nanobot")
STRIP_FRONTMATTER_FIELDS = {
    "allowed-tools", "user-invocable", "argument-hint",
    "version", "license", "compatibility",
}

CURATED_SKILLS = [
    # Search & information
    "10e9928a/duckduckgo-search",
    "10e9928a/task-decomposer",
    # Security
    "0xbeekeeper/security",
    # SSH & remote access
    "arnarsson/ssh-essentials",
    # Docker containers
    "arnarsson/docker-essentials",
    # Edge/IoT hardware
    "ivangdavila/raspberry",
    "josunlp/pi-health",
    "thesethrose/pi-admin",
    "ivangdavila/mqtt",
    "noahseeger/dht11-temp",
    # Linux system
    "ivangdavila/kernel",
    "1999azzar/file-organizer-skill",
    "1999azzar/pihole-ctl",
    "1999azzar/node-red-manager",
]


############################################################
# Chinese service detection engine
############################################################

CN_DOMAIN_PATTERNS = [
    re.compile(r"\.cn[/\s\"'\)]", re.IGNORECASE),
    re.compile(r"weixin\.qq\.com", re.IGNORECASE),
    re.compile(r"api\.weixin", re.IGNORECASE),
    re.compile(r"qyapi\.weixin", re.IGNORECASE),
    re.compile(r"oapi\.dingtalk", re.IGNORECASE),
    re.compile(r"open\.dingtalk", re.IGNORECASE),
    re.compile(r"open\.feishu", re.IGNORECASE),
    re.compile(r"larksuite", re.IGNORECASE),
    re.compile(r"dashscope\.aliyuncs", re.IGNORECASE),
    re.compile(r"aliyuncs\.com", re.IGNORECASE),
    re.compile(r"aliyun\.com", re.IGNORECASE),
    re.compile(r"cloud\.tencent", re.IGNORECASE),
    re.compile(r"tencentyun", re.IGNORECASE),
    re.compile(r"bce\.baidu", re.IGNORECASE),
    re.compile(r"baidu\.com", re.IGNORECASE),
    re.compile(r"huaweicloud", re.IGNORECASE),
    re.compile(r"api\.deepseek\.com", re.IGNORECASE),
    re.compile(r"api\.moonshot", re.IGNORECASE),
    re.compile(r"bigmodel\.cn", re.IGNORECASE),
    re.compile(r"volces\.com", re.IGNORECASE),
    re.compile(r"xfyun\.cn", re.IGNORECASE),
    re.compile(r"xiaohongshu", re.IGNORECASE),
    re.compile(r"douyin\.com", re.IGNORECASE),
    re.compile(r"weibo\.com", re.IGNORECASE),
    re.compile(r"tencent-connect", re.IGNORECASE),
    re.compile(r"dingtalk-stream", re.IGNORECASE),
    re.compile(r"shengsuanyun", re.IGNORECASE),
]

CN_KEYWORD_PATTERNS = [
    re.compile(r"\bwechat\b", re.IGNORECASE),
    re.compile(r"\bweixin\b", re.IGNORECASE),
    re.compile(r"\bdingtalk\b", re.IGNORECASE),
    re.compile(r"\bfeishu\b", re.IGNORECASE),
    re.compile(r"\baliyun\b", re.IGNORECASE),
    re.compile(r"\balibaba\s+cloud\b", re.IGNORECASE),
    re.compile(r"\btencent\s+cloud\b", re.IGNORECASE),
    re.compile(r"\bbaidu\s+cloud\b", re.IGNORECASE),
    re.compile(r"\bhuawei\s+cloud\b", re.IGNORECASE),
    re.compile(r"\bxiaohongshu\b", re.IGNORECASE),
]


def scan_chinese_services(skill_dir: Path) -> list[str]:
    """
    Scan a skill directory for references to Chinese services/endpoints.
    Returns a list of matched indicators (empty = clean).
    """
    hits = []
    seen = set()

    for f in skill_dir.rglob("*"):
        if not f.is_file():
            continue
        if f.suffix.lower() in (".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".ttf"):
            continue
        if ".git" in f.parts or "node_modules" in f.parts:
            continue

        try:
            content = f.read_text(encoding="utf-8", errors="replace")
        except Exception:
            continue

        rel = str(f.relative_to(skill_dir))

        for pattern in CN_DOMAIN_PATTERNS:
            for m in pattern.finditer(content):
                key = (rel, m.group())
                if key not in seen:
                    seen.add(key)
                    hits.append(f"{rel}: {m.group().strip()}")

        for pattern in CN_KEYWORD_PATTERNS:
            for m in pattern.finditer(content):
                key = (rel, m.group())
                if key not in seen:
                    seen.add(key)
                    hits.append(f"{rel}: {m.group().strip()}")

    return hits


############################################################
# Dependency detection engine
############################################################

# Patterns to extract install commands from markdown body
INSTALL_PATTERNS = [
    (re.compile(r"(?:pip|pip3|uv pip)\s+install\s+([a-zA-Z0-9_-]+(?:\[[\w,]+\])?)"), "pip"),
    (re.compile(r"npm\s+install\s+(?:-g\s+)?([a-zA-Z0-9@/_-]+)"), "npm"),
    (re.compile(r"brew\s+install\s+([a-zA-Z0-9_/-]+)"), "brew"),
    (re.compile(r"apt(?:-get)?\s+install\s+(?:-y\s+)?([a-zA-Z0-9_.-]+)"), "apt"),
    (re.compile(r"go\s+install\s+([a-zA-Z0-9./_@-]+)"), "go"),
    (re.compile(r"cargo\s+install\s+([a-zA-Z0-9_-]+)"), "cargo"),
]

# Common binaries we know how to install
BIN_INSTALL_HINTS = {
    "curl": "apt install curl",
    "wget": "apt install wget",
    "jq": "apt install jq",
    "git": "apt install git",
    "docker": "curl -fsSL https://get.docker.com | sh",
    "node": "apt install nodejs",
    "npm": "apt install npm",
    "python3": "apt install python3",
    "pip": "apt install python3-pip",
    "pip3": "apt install python3-pip",
    "go": "see start.sh (auto-installed)",
    "gh": "apt install gh  # or: https://cli.github.com",
    "tmux": "apt install tmux",
    "ffmpeg": "apt install ffmpeg",
    "ssh": "apt install openssh-client",
    "rsync": "apt install rsync",
    "mosquitto_pub": "apt install mosquitto-clients",
    "mosquitto_sub": "apt install mosquitto-clients",
    "vcgencmd": "raspi-config (Raspberry Pi only)",
    "pihole": "curl -sSL https://install.pi-hole.net | bash",
    "ollama": "see start.sh (auto-installed)",
}

_bin_cache: dict[str, bool] = {}
_pip_cache: set[str] | None = None


def check_bin(name: str) -> bool:
    """Check if a binary is available on PATH."""
    if name in _bin_cache:
        return _bin_cache[name]
    found = shutil.which(name) is not None
    _bin_cache[name] = found
    return found


def get_installed_pip_packages() -> set[str]:
    """Get set of installed pip package names (lowercase, normalized)."""
    global _pip_cache
    if _pip_cache is not None:
        return _pip_cache
    _pip_cache = set()
    try:
        out = subprocess.run(
            [sys.executable, "-m", "pip", "list", "--format=json"],
            capture_output=True, text=True, timeout=10,
        )
        if out.returncode == 0:
            for pkg in json.loads(out.stdout):
                _pip_cache.add(pkg["name"].lower().replace("-", "_"))
    except Exception:
        pass
    return _pip_cache


def get_system_info() -> dict:
    """Gather basic system info for OS-compatibility checks."""
    info = {
        "os": sys.platform,
        "arch": platform.machine(),
        "python": platform.python_version(),
    }
    if sys.platform == "linux":
        try:
            with open("/etc/os-release") as f:
                for line in f:
                    if line.startswith("ID="):
                        info["distro"] = line.strip().split("=", 1)[1].strip('"')
                    elif line.startswith("VERSION_ID="):
                        info["distro_version"] = line.strip().split("=", 1)[1].strip('"')
        except OSError:
            pass
    return info


def extract_deps_from_frontmatter(fm: dict) -> dict:
    """
    Extract declared dependencies from frontmatter metadata.
    Returns {"bins": [...], "env": [...], "pip": [...], "npm": [...], "os": [...]}.
    """
    deps = {"bins": [], "env": [], "pip": [], "npm": [], "os": [], "any_bins": []}

    md = fm.get("metadata")
    if md is None:
        return deps

    if isinstance(md, str):
        try:
            md = json.loads(md)
        except json.JSONDecodeError:
            return deps

    if not isinstance(md, dict):
        return deps

    source = None
    for key in OPENCLAW_METADATA_KEYS:
        if key in md:
            source = md[key]
            break
    if source is None and any(k in md for k in ("requires", "emoji", "os", "install")):
        source = md
    if source is None:
        return deps

    if not isinstance(source, dict):
        return deps

    req = source.get("requires", {})
    if isinstance(req, dict):
        deps["bins"] = req.get("bins", [])
        deps["any_bins"] = req.get("anyBins", [])
        deps["env"] = req.get("env", [])

    deps["os"] = source.get("os", [])

    for inst in source.get("install", []):
        if isinstance(inst, dict):
            kind = inst.get("kind", "")
            if kind == "node":
                pkg = inst.get("package", "")
                if pkg:
                    deps["npm"].append(pkg)
            elif kind in ("uv", "pip"):
                pkg = inst.get("package", "")
                if pkg:
                    deps["pip"].append(pkg)

    if source.get("npm"):
        deps["npm"].append(source["npm"])

    return deps


def extract_deps_from_body(body: str) -> dict:
    """
    Heuristically extract dependencies from SKILL.md markdown body by
    scanning for install commands inside code blocks.
    """
    deps = {"pip": [], "npm": [], "brew": [], "apt": [], "go": [], "cargo": []}
    seen = set()

    code_blocks = re.findall(r"```(?:bash|sh|shell|zsh)?\n(.*?)```", body, re.DOTALL)
    scan_text = "\n".join(code_blocks) if code_blocks else body

    for pattern, pkg_type in INSTALL_PATTERNS:
        for m in pattern.finditer(scan_text):
            pkg = m.group(1).strip()
            if pkg.startswith("-") or pkg.startswith("$"):
                continue
            key = (pkg_type, pkg.lower())
            if key not in seen:
                seen.add(key)
                deps[pkg_type].append(pkg)

    return deps


def extract_script_types(skill_dir: Path) -> list[str]:
    """Detect what runtimes the skill's scripts need."""
    runtimes = set()
    scripts_dir = skill_dir / "scripts"
    if not scripts_dir.is_dir():
        return []

    for f in scripts_dir.rglob("*"):
        if not f.is_file():
            continue
        suffix = f.suffix.lower()
        if suffix in (".ts", ".js", ".mjs", ".cjs"):
            runtimes.add("node")
        elif suffix in (".py",):
            runtimes.add("python3")
        elif suffix in (".sh", ".bash"):
            runtimes.add("bash")
        elif suffix in (".rb",):
            runtimes.add("ruby")
        elif suffix in (".go",):
            runtimes.add("go")
        elif suffix in (".rs",):
            runtimes.add("cargo")

    if (scripts_dir / "package.json").exists():
        runtimes.add("npm")

    return sorted(runtimes)


class DepReport:
    """Dependency analysis report for a single skill."""

    def __init__(self, owner: str, slug: str):
        self.owner = owner
        self.slug = slug
        self.bins_ok: list[str] = []
        self.bins_missing: list[str] = []
        self.any_bins_ok: list[str] = []
        self.any_bins_missing: bool = False
        self.env_ok: list[str] = []
        self.env_missing: list[str] = []
        self.pip_ok: list[str] = []
        self.pip_missing: list[str] = []
        self.npm_packages: list[str] = []
        self.other_packages: dict[str, list[str]] = {}
        self.script_runtimes: list[str] = []
        self.runtime_missing: list[str] = []
        self.os_restriction: list[str] = []
        self.os_compatible: bool = True
        self.is_pure_markdown: bool = True

    @property
    def ready(self) -> bool:
        return (
            not self.bins_missing
            and not self.any_bins_missing
            and not self.env_missing
            and not self.pip_missing
            and not self.runtime_missing
            and self.os_compatible
        )

    @property
    def status_icon(self) -> str:
        if self.ready:
            return "READY"
        if self.bins_missing or self.runtime_missing or not self.os_compatible:
            return "NEEDS"
        return "PARTIAL"

    def format_short(self) -> str:
        if self.ready:
            return "ready"
        parts = []
        if not self.os_compatible:
            parts.append(f"os:{','.join(self.os_restriction)}")
        if self.bins_missing:
            parts.append(f"bins:{','.join(self.bins_missing)}")
        if self.env_missing:
            parts.append(f"env:{','.join(self.env_missing)}")
        if self.pip_missing:
            parts.append(f"pip:{','.join(self.pip_missing)}")
        if self.runtime_missing:
            parts.append(f"runtime:{','.join(self.runtime_missing)}")
        if self.npm_packages:
            parts.append(f"npm:{','.join(self.npm_packages)}")
        return " | ".join(parts) if parts else "ready"

    def format_full(self) -> str:
        lines = []
        lines.append(f"Dependency Report: {self.owner}/{self.slug}")
        lines.append("=" * 50)

        if self.os_restriction:
            compat = "YES" if self.os_compatible else "NO"
            lines.append(f"  OS Restriction: {', '.join(self.os_restriction)} (compatible: {compat})")

        if self.is_pure_markdown and not self.bins_missing and not self.pip_missing:
            lines.append("  Type: Pure markdown (no external dependencies)")
            lines.append(f"  Status: READY")
            return "\n".join(lines)

        if self.bins_ok:
            lines.append(f"  Binaries (installed):    {', '.join(self.bins_ok)}")
        if self.bins_missing:
            lines.append(f"  Binaries (MISSING):      {', '.join(self.bins_missing)}")
            for b in self.bins_missing:
                hint = BIN_INSTALL_HINTS.get(b, f"install '{b}' manually")
                lines.append(f"    -> {b}: {hint}")

        if self.any_bins_ok:
            lines.append(f"  Alt binaries (have one): {', '.join(self.any_bins_ok)}")
        if self.any_bins_missing:
            lines.append(f"  Alt binaries (NEED ONE): install any of the above")

        if self.env_ok:
            lines.append(f"  Env vars (set):          {', '.join(self.env_ok)}")
        if self.env_missing:
            lines.append(f"  Env vars (MISSING):      {', '.join(self.env_missing)}")
            for e in self.env_missing:
                lines.append(f"    -> export {e}=<your-value>")

        if self.pip_ok:
            lines.append(f"  Python packages (have):  {', '.join(self.pip_ok)}")
        if self.pip_missing:
            lines.append(f"  Python packages (NEED):  {', '.join(self.pip_missing)}")
            lines.append(f"    -> pip install {' '.join(self.pip_missing)}")

        if self.npm_packages:
            lines.append(f"  Node packages (NEED):    {', '.join(self.npm_packages)}")
            lines.append(f"    -> npm install -g {' '.join(self.npm_packages)}")

        for pkg_type, pkgs in self.other_packages.items():
            if pkgs:
                lines.append(f"  {pkg_type} packages (NEED): {', '.join(pkgs)}")

        if self.script_runtimes:
            lines.append(f"  Script runtimes needed:  {', '.join(self.script_runtimes)}")
        if self.runtime_missing:
            lines.append(f"  Runtimes (MISSING):      {', '.join(self.runtime_missing)}")
            for r in self.runtime_missing:
                hint = BIN_INSTALL_HINTS.get(r, f"install '{r}'")
                lines.append(f"    -> {r}: {hint}")

        status = "READY" if self.ready else "NEEDS DEPENDENCIES"
        lines.append(f"\n  Status: {status}")

        return "\n".join(lines)


def analyze_skill_deps(owner: str, slug: str, skill_dir: Path) -> DepReport:
    """Run full dependency analysis on a skill."""
    report = DepReport(owner, slug)

    skill_md = skill_dir / "SKILL.md"
    if not skill_md.exists():
        skill_md = skill_dir / "skill.md"
    if not skill_md.exists():
        return report

    content = skill_md.read_text(encoding="utf-8", errors="replace")
    fm, body = parse_frontmatter(content)

    fm_deps = extract_deps_from_frontmatter(fm or {})
    body_deps = extract_deps_from_body(body)
    runtimes = extract_script_types(skill_dir)

    if runtimes or fm_deps["bins"] or fm_deps["npm"] or body_deps["npm"]:
        report.is_pure_markdown = False

    # OS compatibility
    sysinfo = get_system_info()
    os_map = {"darwin": "macos", "linux": "linux", "win32": "windows"}
    current_os = os_map.get(sysinfo["os"], sysinfo["os"])
    if fm_deps["os"]:
        report.os_restriction = fm_deps["os"]
        if current_os not in fm_deps["os"] and sysinfo["os"] not in fm_deps["os"]:
            report.os_compatible = False

    # Required binaries
    for b in fm_deps["bins"]:
        if check_bin(b):
            report.bins_ok.append(b)
        else:
            report.bins_missing.append(b)

    # Any-of binaries
    if fm_deps["any_bins"]:
        found_any = False
        for b in fm_deps["any_bins"]:
            if check_bin(b):
                report.any_bins_ok.append(b)
                found_any = True
        if not found_any:
            report.any_bins_missing = True

    # Env vars
    for e in fm_deps["env"]:
        if os.environ.get(e):
            report.env_ok.append(e)
        else:
            report.env_missing.append(e)

    # Pip packages (from frontmatter)
    installed_pip = get_installed_pip_packages()
    all_pip = set()
    for p in fm_deps["pip"] + body_deps.get("pip", []):
        all_pip.add(p)
    for p in all_pip:
        normalized = p.lower().replace("-", "_").split("[")[0]
        if normalized in installed_pip:
            report.pip_ok.append(p)
        else:
            report.pip_missing.append(p)

    # npm packages
    all_npm = set(fm_deps["npm"] + body_deps.get("npm", []))
    report.npm_packages = sorted(all_npm)

    # Other package managers from body
    for pkg_type in ("brew", "apt", "go", "cargo"):
        pkgs = body_deps.get(pkg_type, [])
        if pkgs:
            report.other_packages[pkg_type] = pkgs
            report.is_pure_markdown = False

    # Script runtimes
    report.script_runtimes = runtimes
    for rt in runtimes:
        bin_name = rt
        if rt == "npm":
            bin_name = "npm"
        if not check_bin(bin_name):
            report.runtime_missing.append(rt)

    return report


def parse_frontmatter(content: str) -> tuple[dict | None, str]:
    """Extract YAML frontmatter and body from SKILL.md content."""
    match = re.match(r"^---\n(.*?)\n---\n?(.*)", content, re.DOTALL)
    if not match:
        return None, content
    raw_fm = match.group(1)
    body = match.group(2)
    fm = parse_yaml_ish(raw_fm)
    return fm, body


def parse_yaml_ish(text: str) -> dict:
    """
    Parse the frontmatter text. Handles both simple YAML key: value lines
    and inline JSON in the metadata field (which is common in OpenClaw skills).
    """
    result = {}
    lines = text.split("\n")
    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            i += 1
            continue

        colon_idx = stripped.find(":")
        if colon_idx < 0:
            i += 1
            continue

        key = stripped[:colon_idx].strip()
        value = stripped[colon_idx + 1:].strip()

        if value.startswith("{") or value.startswith("["):
            try:
                result[key] = json.loads(value)
            except json.JSONDecodeError:
                json_str = value
                i += 1
                while i < len(lines):
                    json_str += "\n" + lines[i]
                    try:
                        result[key] = json.loads(json_str)
                        break
                    except json.JSONDecodeError:
                        i += 1
                        continue
        elif value.startswith('"') and value.endswith('"'):
            result[key] = value[1:-1]
        elif value.startswith("'") and value.endswith("'"):
            result[key] = value[1:-1]
        elif value == "true":
            result[key] = True
        elif value == "false":
            result[key] = False
        elif value:
            result[key] = value
        else:
            block = {}
            i += 1
            while i < len(lines) and lines[i].startswith("  "):
                sub = lines[i].strip()
                sc = sub.find(":")
                if sc > 0:
                    sk = sub[:sc].strip()
                    sv = sub[sc + 1:].strip()
                    if sv.startswith('"') and sv.endswith('"'):
                        sv = sv[1:-1]
                    block[sk] = sv
                i += 1
            if block:
                result[key] = block
            continue

        i += 1

    return result


def convert_metadata(fm: dict) -> dict:
    """
    Convert OpenClaw metadata keys to Xagent's nanobot format.
    metadata.openclaw / metadata.clawdbot / metadata.clawdis -> metadata.nanobot
    """
    md = fm.get("metadata")
    if md is None:
        return fm

    if isinstance(md, str):
        try:
            md = json.loads(md)
        except json.JSONDecodeError:
            return fm

    if not isinstance(md, dict):
        return fm

    source_data = None
    for key in OPENCLAW_METADATA_KEYS:
        if key in md:
            source_data = md[key]
            break

    if source_data is None:
        if any(k in md for k in ("requires", "emoji", "os", "install", "primaryEnv")):
            source_data = md
        else:
            return fm

    nano = {}
    if "emoji" in source_data:
        nano["emoji"] = source_data["emoji"]
    if "requires" in source_data:
        req = source_data["requires"]
        nano_req = {}
        for rk in ("bins", "anyBins", "env", "config", "tools"):
            if rk in req:
                nano_req[rk] = req[rk]
        if nano_req:
            nano["requires"] = nano_req
    if "os" in source_data:
        nano["os"] = source_data["os"]
    if "install" in source_data:
        nano["install"] = source_data["install"]

    fm["metadata"] = json.dumps({"nanobot": nano})
    return fm


def build_frontmatter_string(fm: dict) -> str:
    """Serialize frontmatter dict back to YAML-ish string."""
    lines = []
    lines.append("---")

    if "name" in fm:
        lines.append(f"name: {fm['name']}")
    if "description" in fm:
        desc = fm["description"]
        if ":" in desc or '"' in desc:
            desc = desc.replace('"', '\\"')
            lines.append(f'description: "{desc}"')
        else:
            lines.append(f"description: {desc}")
    if "homepage" in fm:
        lines.append(f"homepage: {fm['homepage']}")
    if "metadata" in fm:
        md = fm["metadata"]
        if isinstance(md, str):
            lines.append(f"metadata: {md}")
        elif isinstance(md, dict):
            lines.append(f"metadata: {json.dumps(md)}")

    lines.append("---")
    return "\n".join(lines)


def convert_skill(owner: str, slug: str, skill_dir: Path, dest_dir: Path, force: bool = False) -> bool:
    """Convert a single OpenClaw skill to Xagent format."""
    skill_md = skill_dir / "SKILL.md"
    if not skill_md.exists():
        skill_md = skill_dir / "skill.md"
    if not skill_md.exists():
        print(f"  SKIP {owner}/{slug}: no SKILL.md found")
        return False

    out_dir = dest_dir / slug
    if out_dir.exists() and not force:
        print(f"  SKIP {owner}/{slug}: already installed (use --force to overwrite)")
        return False

    content = skill_md.read_text(encoding="utf-8", errors="replace")
    fm, body = parse_frontmatter(content)

    if fm is None:
        fm = {"name": slug, "description": f"Converted from openclaw {owner}/{slug}"}

    for field in STRIP_FRONTMATTER_FIELDS:
        fm.pop(field, None)

    if "name" not in fm:
        fm["name"] = slug

    fm = convert_metadata(fm)

    new_content = build_frontmatter_string(fm) + "\n" + body

    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "SKILL.md").write_text(new_content, encoding="utf-8")

    for item in skill_dir.iterdir():
        if item.name in SKIP_FILES:
            continue
        if item.name.startswith("."):
            continue
        if item.name.lower() in ("skill.md",):
            continue
        dst = out_dir / item.name
        if item.is_dir():
            if item.name == "node_modules":
                continue
            if dst.exists():
                shutil.rmtree(dst)
            shutil.copytree(item, dst, ignore=shutil.ignore_patterns(
                "node_modules", "__pycache__", "*.pyc", ".git",
            ))
        else:
            shutil.copy2(item, dst)

    print(f"  OK   {owner}/{slug} -> {out_dir.relative_to(dest_dir.parent.parent)}")
    return True


def find_all_skills() -> list[dict]:
    """Scan the skills archive and return a list of all skills."""
    skills = []
    if not OPENCLAW_SKILLS_DIR.exists():
        return skills

    for owner_dir in sorted(OPENCLAW_SKILLS_DIR.iterdir()):
        if not owner_dir.is_dir():
            continue
        owner = owner_dir.name
        for skill_dir in sorted(owner_dir.iterdir()):
            if not skill_dir.is_dir():
                continue
            slug = skill_dir.name
            skill_md = skill_dir / "SKILL.md"
            if not skill_md.exists():
                skill_md = skill_dir / "skill.md"
            if not skill_md.exists():
                continue

            meta_file = skill_dir / "_meta.json"
            meta = {}
            if meta_file.exists():
                try:
                    meta = json.loads(meta_file.read_text())
                except (json.JSONDecodeError, OSError):
                    pass

            content = skill_md.read_text(encoding="utf-8", errors="replace")
            fm, _ = parse_frontmatter(content)
            desc = ""
            if fm:
                desc = fm.get("description", "")

            has_scripts = (skill_dir / "scripts").is_dir()
            has_refs = (skill_dir / "references").is_dir()
            has_assets = (skill_dir / "assets").is_dir()

            skills.append({
                "owner": owner,
                "slug": slug,
                "path": str(skill_dir),
                "description": desc[:200] if desc else "",
                "version": meta.get("latest", {}).get("version", ""),
                "has_scripts": has_scripts,
                "has_references": has_refs,
                "has_assets": has_assets,
            })

    return skills


def cmd_list(args):
    """List all available skills."""
    skills = find_all_skills()
    print(f"Found {len(skills)} skills in skills archive\n")

    limit = args.limit or 50
    for i, s in enumerate(skills[:limit]):
        extras = []
        if s["has_scripts"]:
            extras.append("scripts")
        if s["has_references"]:
            extras.append("refs")
        if s["has_assets"]:
            extras.append("assets")
        extra_str = f" [{', '.join(extras)}]" if extras else ""
        desc = s["description"][:80] + "..." if len(s["description"]) > 80 else s["description"]
        print(f"  {s['owner']}/{s['slug']}{extra_str}")
        if desc:
            print(f"    {desc}")

    if len(skills) > limit:
        print(f"\n  ... and {len(skills) - limit} more. Use --limit N or 'search' to filter.")


def cmd_search(args):
    """Search skills by keyword, optionally showing dependency status."""
    query = args.query.lower()
    skills = find_all_skills()
    matches = [
        s for s in skills
        if query in s["slug"].lower()
        or query in s["description"].lower()
        or query in s["owner"].lower()
    ]

    if getattr(args, "safe", False):
        print(f"Scanning {len(matches)} results for Chinese service references...")
        safe_matches = []
        for s in matches:
            cn_hits = scan_chinese_services(Path(s["path"]))
            if not cn_hits:
                safe_matches.append(s)
        print(f"Found {len(safe_matches)} clean skills (filtered {len(matches) - len(safe_matches)} with Chinese refs)\n")
        matches = safe_matches
    else:
        print(f"Found {len(matches)} skills matching '{args.query}'")
        if args.deps:
            print("(checking dependencies...)")
        print()

    for s in matches[:args.limit or 50]:
        desc = s["description"][:80] + "..." if len(s["description"]) > 80 else s["description"]

        if args.deps:
            report = analyze_skill_deps(s["owner"], s["slug"], Path(s["path"]))
            icon = report.status_icon
            dep_info = report.format_short()
            print(f"  [{icon:7s}] {s['owner']}/{s['slug']}")
            if not report.ready:
                print(f"            deps: {dep_info}")
        else:
            print(f"  {s['owner']}/{s['slug']}")

        if desc:
            indent = "            " if args.deps else "    "
            print(f"{indent}{desc}")


def cmd_info(args):
    """Show detailed info about a skill including dependency analysis."""
    parts = args.skill.split("/", 1)
    if len(parts) != 2:
        print("Usage: info <owner>/<slug>")
        return

    owner, slug = parts
    skill_dir = OPENCLAW_SKILLS_DIR / owner / slug
    if not skill_dir.exists():
        print(f"Skill '{args.skill}' not found in archive")
        return

    skill_md = skill_dir / "SKILL.md"
    if not skill_md.exists():
        skill_md = skill_dir / "skill.md"

    print(f"Skill: {owner}/{slug}")
    print(f"Path:  {skill_dir}")
    print(f"Files:")
    for item in sorted(skill_dir.rglob("*")):
        if ".git" in item.parts or "node_modules" in item.parts:
            continue
        if item.is_file():
            rel = item.relative_to(skill_dir)
            size = item.stat().st_size
            print(f"  {rel} ({size} bytes)")

    if skill_md.exists():
        content = skill_md.read_text(encoding="utf-8", errors="replace")
        fm, _ = parse_frontmatter(content)
        if fm:
            print(f"\nFrontmatter:")
            for k, v in fm.items():
                val = str(v)[:120]
                print(f"  {k}: {val}")

    cn_hits = scan_chinese_services(skill_dir)
    if cn_hits:
        print(f"\nChinese Service Scan: FLAGGED ({len(cn_hits)} references)")
        for h in cn_hits[:20]:
            print(f"  - {h}")
        if len(cn_hits) > 20:
            print(f"  ... and {len(cn_hits) - 20} more")
        print("  This skill will be BLOCKED during install (use --force to override).")
    else:
        print("\nChinese Service Scan: CLEAN")

    print()
    report = analyze_skill_deps(owner, slug, skill_dir)
    print(report.format_full())


def cmd_install(args):
    """Convert and install a single skill with dependency report."""
    parts = args.skill.split("/", 1)
    if len(parts) != 2:
        print("Usage: install <owner>/<slug>")
        return

    owner, slug = parts
    skill_dir = OPENCLAW_SKILLS_DIR / owner / slug
    if not skill_dir.exists():
        print(f"Skill '{args.skill}' not found in archive")
        return

    cn_hits = scan_chinese_services(skill_dir)
    if cn_hits and not args.force:
        print(f"  BLOCKED {owner}/{slug}: Chinese service references detected:")
        for h in cn_hits[:10]:
            print(f"    - {h}")
        if len(cn_hits) > 10:
            print(f"    ... and {len(cn_hits) - 10} more")
        print(f"  Use --force to install anyway.")
        return
    elif cn_hits:
        print(f"  WARNING: {owner}/{slug} references {len(cn_hits)} Chinese service(s) (--force used)")

    report = analyze_skill_deps(owner, slug, skill_dir)

    if not report.os_compatible and not args.force:
        print(f"  SKIP {owner}/{slug}: OS restriction {report.os_restriction} "
              f"(current: {sys.platform}). Use --force to install anyway.")
        return

    dest = Path(args.dest) if args.dest else XAGENT_GLOBAL_SKILLS
    dest.mkdir(parents=True, exist_ok=True)

    print(f"Converting {owner}/{slug} -> Xagent format")
    ok = convert_skill(owner, slug, skill_dir, dest, force=args.force)
    if ok:
        print(f"\nInstalled to {dest / slug}")
        if report.ready:
            print(f"Status: READY -- all dependencies satisfied")
        else:
            print(f"\nDependencies needed to use this skill:")
            if report.bins_missing:
                print(f"  Missing binaries:")
                for b in report.bins_missing:
                    hint = BIN_INSTALL_HINTS.get(b, f"install '{b}' manually")
                    print(f"    {b} -> {hint}")
            if report.env_missing:
                print(f"  Missing environment variables:")
                for e in report.env_missing:
                    print(f"    export {e}=<your-value>")
            if report.pip_missing:
                print(f"  Missing Python packages:")
                print(f"    pip install {' '.join(report.pip_missing)}")
            if report.npm_packages:
                print(f"  Node.js packages needed:")
                print(f"    npm install -g {' '.join(report.npm_packages)}")
            if report.runtime_missing:
                print(f"  Missing script runtimes:")
                for r in report.runtime_missing:
                    hint = BIN_INSTALL_HINTS.get(r, f"install '{r}'")
                    print(f"    {r} -> {hint}")
            for pkg_type, pkgs in report.other_packages.items():
                if pkgs:
                    print(f"  {pkg_type} packages:")
                    print(f"    {pkg_type} install {' '.join(pkgs)}")

        print(f"\nXagent will auto-discover it on next startup.")


def cmd_install_publisher(args):
    """Install all skills from a publisher."""
    owner = args.owner
    owner_dir = OPENCLAW_SKILLS_DIR / owner
    if not owner_dir.exists():
        print(f"Publisher '{owner}' not found in archive")
        return

    dest = Path(args.dest) if args.dest else XAGENT_GLOBAL_SKILLS
    dest.mkdir(parents=True, exist_ok=True)

    count = 0
    for skill_dir in sorted(owner_dir.iterdir()):
        if not skill_dir.is_dir():
            continue
        slug = skill_dir.name
        if convert_skill(owner, slug, skill_dir, dest, force=args.force):
            count += 1

    print(f"\nInstalled {count} skill(s) from {owner}")


def cmd_install_curated(args):
    """Install a curated set of high-quality skills."""
    dest = Path(args.dest) if args.dest else XAGENT_GLOBAL_SKILLS
    dest.mkdir(parents=True, exist_ok=True)

    print("Installing curated skill set...\n")
    count = 0
    blocked = 0
    for spec in CURATED_SKILLS:
        owner, slug = spec.split("/", 1)
        skill_dir = OPENCLAW_SKILLS_DIR / owner / slug
        if not skill_dir.exists():
            print(f"  MISS {spec}: not found in archive")
            continue
        cn_hits = scan_chinese_services(skill_dir)
        if cn_hits and not args.force:
            print(f"  BLOCK {spec}: {len(cn_hits)} Chinese service reference(s)")
            blocked += 1
            continue
        if convert_skill(owner, slug, skill_dir, dest, force=args.force):
            count += 1

    print(f"\nInstalled {count}/{len(CURATED_SKILLS)} curated skills")
    if blocked:
        print(f"Blocked {blocked} skill(s) with Chinese service references")


def cmd_check(args):
    """Check dependencies for a specific skill without installing."""
    parts = args.skill.split("/", 1)
    if len(parts) != 2:
        print("Usage: check <owner>/<slug>")
        return

    owner, slug = parts
    skill_dir = OPENCLAW_SKILLS_DIR / owner / slug
    if not skill_dir.exists():
        print(f"Skill '{args.skill}' not found in archive")
        return

    report = analyze_skill_deps(owner, slug, skill_dir)
    print(report.format_full())


def cmd_check_installed(args):
    """Check dependencies for all currently installed skills."""
    search_dirs = []
    ws = XAGENT_WORKSPACE_SKILLS
    gs = XAGENT_GLOBAL_SKILLS

    if args.dest:
        search_dirs.append(Path(args.dest))
    else:
        if ws.exists():
            search_dirs.append(ws)
        if gs.exists():
            search_dirs.append(gs)

    if not search_dirs:
        print("No installed skills found.")
        return

    ready_count = 0
    needs_count = 0
    total = 0

    for sdir in search_dirs:
        print(f"\nScanning: {sdir}\n")
        for skill_dir in sorted(sdir.iterdir()):
            if not skill_dir.is_dir():
                continue
            skill_md = skill_dir / "SKILL.md"
            if not skill_md.exists():
                continue

            slug = skill_dir.name
            total += 1

            report = analyze_skill_deps("installed", slug, skill_dir)

            if report.ready:
                ready_count += 1
                icon = "READY"
            else:
                needs_count += 1
                icon = "NEEDS"

            dep_summary = report.format_short()
            print(f"  [{icon:5s}] {slug}")
            if not report.ready:
                print(f"           {dep_summary}")

    print(f"\n{'=' * 50}")
    print(f"Total: {total} skills | Ready: {ready_count} | Need deps: {needs_count}")

    if needs_count > 0:
        print(f"\nRun 'info <owner>/<slug>' or 'check <owner>/<slug>' for install instructions.")


def cmd_ready(args):
    """List only skills from the archive that are ready to use right now."""
    skills = find_all_skills()
    query = args.query.lower() if args.query else None

    if query:
        skills = [
            s for s in skills
            if query in s["slug"].lower()
            or query in s["description"].lower()
        ]

    print(f"Scanning {len(skills)} skills for compatibility...\n")
    ready = []

    for s in skills:
        skill_dir = Path(s["path"])
        report = analyze_skill_deps(s["owner"], s["slug"], skill_dir)
        if report.ready:
            ready.append((s, report))

    limit = args.limit or 50
    print(f"Found {len(ready)} ready-to-use skills\n")
    for s, r in ready[:limit]:
        desc = s["description"][:80] + "..." if len(s["description"]) > 80 else s["description"]
        marker = "markdown" if r.is_pure_markdown else "has deps (all satisfied)"
        print(f"  {s['owner']}/{s['slug']}  [{marker}]")
        if desc:
            print(f"    {desc}")

    if len(ready) > limit:
        print(f"\n  ... and {len(ready) - limit} more. Use --limit N to show more.")


def cmd_bulk(args):
    """Bulk convert all skills (or filtered subset)."""
    dest = Path(args.dest) if args.dest else XAGENT_GLOBAL_SKILLS
    dest.mkdir(parents=True, exist_ok=True)

    skills = find_all_skills()
    if args.filter:
        filt = args.filter.lower()
        skills = [s for s in skills if filt in s["slug"].lower() or filt in s["description"].lower()]

    safe_mode = getattr(args, "safe", True)

    print(f"Converting {len(skills)} skills...\n")
    count = 0
    errors = 0
    blocked = 0
    for s in skills:
        skill_dir = Path(s["path"])
        if safe_mode and not args.force:
            cn_hits = scan_chinese_services(skill_dir)
            if cn_hits:
                blocked += 1
                continue
        try:
            if convert_skill(s["owner"], s["slug"], skill_dir, dest, force=args.force):
                count += 1
        except Exception as e:
            print(f"  ERR  {s['owner']}/{s['slug']}: {e}")
            errors += 1

    print(f"\nConverted {count} skills ({errors} errors)")
    if blocked:
        print(f"Blocked {blocked} skill(s) with Chinese service references (use --force to override)")


def main():
    parser = argparse.ArgumentParser(
        description="OpenClaw -> Xagent Skill Converter",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    sub = parser.add_subparsers(dest="command")

    p_list = sub.add_parser("list", help="List all available skills")
    p_list.add_argument("--limit", type=int, default=50, help="Max results to show")

    p_search = sub.add_parser("search", help="Search skills by keyword")
    p_search.add_argument("query", help="Search keyword")
    p_search.add_argument("--limit", type=int, default=50, help="Max results to show")
    p_search.add_argument("--deps", action="store_true",
                          help="Show dependency status for each result")
    p_search.add_argument("--safe", action="store_true",
                          help="Filter out skills with Chinese service references")

    p_info = sub.add_parser("info", help="Show skill details")
    p_info.add_argument("skill", help="Skill in owner/slug format")

    p_install = sub.add_parser("install", help="Convert and install a skill")
    p_install.add_argument("skill", help="Skill in owner/slug format")
    p_install.add_argument("--dest", help="Destination directory (default: ~/.xagent/skills)")
    p_install.add_argument("--force", action="store_true", help="Overwrite existing")

    p_pub = sub.add_parser("install-publisher", help="Install all skills from a publisher")
    p_pub.add_argument("owner", help="Publisher username")
    p_pub.add_argument("--dest", help="Destination directory")
    p_pub.add_argument("--force", action="store_true", help="Overwrite existing")

    p_curated = sub.add_parser("install-curated", help="Install curated skill set")
    p_curated.add_argument("--dest", help="Destination directory")
    p_curated.add_argument("--force", action="store_true", help="Overwrite existing")

    p_check = sub.add_parser("check", help="Check dependencies for a skill")
    p_check.add_argument("skill", help="Skill in owner/slug format")

    p_check_inst = sub.add_parser("check-installed",
                                  help="Check deps for all installed skills")
    p_check_inst.add_argument("--dest", help="Override skills directory to scan")

    p_ready = sub.add_parser("ready", help="List skills ready to use on this device")
    p_ready.add_argument("query", nargs="?", help="Optional keyword filter")
    p_ready.add_argument("--limit", type=int, default=50, help="Max results to show")

    p_bulk = sub.add_parser("bulk", help="Bulk convert skills")
    p_bulk.add_argument("--dest", help="Destination directory")
    p_bulk.add_argument("--force", action="store_true", help="Overwrite existing")
    p_bulk.add_argument("--filter", help="Only convert skills matching keyword")
    p_bulk.add_argument("--safe", action="store_true", default=True,
                        help="Block skills with Chinese service references (default: on)")

    args = parser.parse_args()

    if not OPENCLAW_SKILLS_DIR.exists():
        print(f"Error: skills archive not found at {OPENCLAW_SKILLS_DIR}")
        print("Run: git clone https://github.com/moltbot/skills.git skills")
        sys.exit(1)

    commands = {
        "list": cmd_list,
        "search": cmd_search,
        "info": cmd_info,
        "check": cmd_check,
        "check-installed": cmd_check_installed,
        "ready": cmd_ready,
        "install": cmd_install,
        "install-publisher": cmd_install_publisher,
        "install-curated": cmd_install_curated,
        "bulk": cmd_bulk,
    }

    if args.command in commands:
        commands[args.command](args)
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
