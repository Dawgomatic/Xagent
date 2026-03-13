#!/usr/bin/env python3
"""
Supabase 테이블 자동 생성 스크립트

실행:
    python3 scripts/setup_supabase.py

인증:
    SUPABASE_ACCESS_TOKEN  — ~/.openclaw/.env에서 로드
    SUPABASE_URL           — project-ref 추출용

Management API:
    POST https://api.supabase.com/v1/projects/{ref}/database/query
    Authorization: Bearer {SUPABASE_ACCESS_TOKEN}

SaaS 모드 (RAON_API_URL 설정):
    → 이 스크립트 불필요. k-startup.ai 서버가 Supabase를 직접 관리.
"""

import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path


# ─── .env 로드 ────────────────────────────────────────────────────────────────
def _load_env():
    for env_path in [
        Path.home() / ".openclaw" / ".env",
        Path(__file__).parent.parent / ".env",
    ]:
        if not env_path.exists():
            continue
        for line in env_path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, val = line.partition("=")
            key = key.strip()
            val = val.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = val


_load_env()

SUPABASE_URL    = os.environ.get("SUPABASE_URL", "").rstrip("/")
SERVICE_KEY     = os.environ.get("SUPABASE_SERVICE_KEY", "")
ACCESS_TOKEN    = os.environ.get("SUPABASE_ACCESS_TOKEN", "")
RAON_API_URL    = os.environ.get("RAON_API_URL", "")

# ─── 사전 검사 ───────────────────────────────────────────────────────────────
if RAON_API_URL:
    print("  RAON_API_URL 설정됨 (SaaS 모드)")
    print(f"   Supabase는 {RAON_API_URL} 서버에서 중앙 관리됩니다.")
    print("   이 스크립트는 로컬 모드 전용입니다.")
    sys.exit(0)

if not SUPABASE_URL:
    print(" SUPABASE_URL 미설정 (~/.openclaw/.env 확인)")
    sys.exit(1)

if not ACCESS_TOKEN:
    print(" SUPABASE_ACCESS_TOKEN 미설정")
    print("   ~/.openclaw/.env에 추가: SUPABASE_ACCESS_TOKEN=sbp_xxx...")
    sys.exit(1)

# project-ref: https://{ref}.supabase.co → ref 추출
PROJECT_REF = SUPABASE_URL.split("//")[-1].split(".")[0]

print(f" Supabase 프로젝트: {PROJECT_REF}")
print(f"   URL: {SUPABASE_URL}")
print()


# ─── Management API 요청 헬퍼 ────────────────────────────────────────────────
def _mgmt_request(method: str, path: str, body=None):
    """
    Supabase Management API 호출.
    urllib + User-Agent 헤더로 Cloudflare 우회.
    """
    url = f"https://api.supabase.com/v1/{path.lstrip('/')}"
    headers = {
        "Authorization": f"Bearer {ACCESS_TOKEN}",
        "Content-Type": "application/json",
        # Cloudflare가 Python urllib를 차단하므로 curl User-Agent 사용
        "User-Agent": "curl/8.7.1",
        "Accept": "*/*",
    }
    data = json.dumps(body, ensure_ascii=False).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            resp_body = r.read().decode("utf-8", errors="replace")
            return r.status, resp_body
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8", errors="replace")
        return e.code, err_body


# ─── REST API 헬퍼 (테이블 존재 확인용) ──────────────────────────────────────
def _rest_request(method: str, table: str):
    """PostgREST로 테이블 존재 여부 확인 (service_role key 사용)"""
    url = f"{SUPABASE_URL}/rest/v1/{table}?limit=1"
    headers = {
        "apikey": SERVICE_KEY,
        "Authorization": f"Bearer {SERVICE_KEY}",
        "User-Agent": "curl/8.7.1",
    }
    req = urllib.request.Request(url, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=8) as r:
            r.read()
            return True   # 200 = 테이블 존재
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        if "does not exist" in body or "PGRST205" in body or e.code == 404:
            return False
        return True       # 다른 에러는 존재로 간주
    except Exception:
        return False


# ─── 테이블 존재 확인 ─────────────────────────────────────────────────────────
def check_tables() -> tuple:
    """(raon_evaluations_exists, raon_feedback_exists) 반환"""
    if not SERVICE_KEY:
        # service_key 없으면 Management API로 확인
        status, body = _mgmt_request(
            "GET", f"projects/{PROJECT_REF}/database/tables?schema=public"
        )
        if status == 200:
            tables = [t.get("name", "") for t in json.loads(body)]
            return "raon_evaluations" in tables, "raon_feedback" in tables
        return False, False
    return _rest_request("GET", "raon_evaluations"), _rest_request("GET", "raon_feedback")


# ─── SQL 실행 (Management API) ───────────────────────────────────────────────
def execute_sql(sql: str) -> tuple:
    """
    POST https://api.supabase.com/v1/projects/{ref}/database/query
    Body: {"query": "<SQL>"}
    Returns: (success: bool, message: str)
    """
    status, body = _mgmt_request(
        "POST",
        f"projects/{PROJECT_REF}/database/query",
        {"query": sql},
    )
    if status in (200, 201):
        return True, body
    return False, f"HTTP {status}: {body[:300]}"


# ─── 메인 실행 ───────────────────────────────────────────────────────────────
CREATE_SQL = """
CREATE TABLE IF NOT EXISTS raon_evaluations (
  id            uuid         DEFAULT gen_random_uuid() PRIMARY KEY,
  session_id    text,
  mode          text,
  input_text    text,
  result_text   text,
  score         jsonb,
  duration_sec  float,
  model         text,
  created_at    timestamptz  DEFAULT now()
);

CREATE TABLE IF NOT EXISTS raon_feedback (
  id             uuid   DEFAULT gen_random_uuid() PRIMARY KEY,
  evaluation_id  uuid   REFERENCES raon_evaluations(id),
  rating         int,
  comment        text,
  created_at     timestamptz DEFAULT now()
);

ALTER TABLE raon_evaluations ENABLE ROW LEVEL SECURITY;
ALTER TABLE raon_feedback    ENABLE ROW LEVEL SECURITY;
""".strip()

print(" 테이블 존재 여부 확인...")
eval_ok, fb_ok = check_tables()
print(f"   raon_evaluations : {' 존재' if eval_ok else ' 없음'}")
print(f"   raon_feedback    : {' 존재' if fb_ok else ' 없음'}")

if eval_ok and fb_ok:
    print()
    print(" 모든 테이블이 이미 존재합니다. 파이프라인 준비 완료!")
    sys.exit(0)

print()
print(" Management API로 테이블 생성 중...")
success, msg = execute_sql(CREATE_SQL)

if success:
    print(f"    SQL 실행 성공")
else:
    print(f"    실패: {msg}")
    sys.exit(1)

# ─── 최종 확인 ───────────────────────────────────────────────────────────────
print()
print(" 최종 확인...")
eval_ok, fb_ok = check_tables()
print(f"   raon_evaluations : {' 존재' if eval_ok else '  확인 필요'}")
print(f"   raon_feedback    : {' 존재' if fb_ok else '  확인 필요'}")
print()

if eval_ok and fb_ok:
    print(" 피드백 수집 파이프라인 준비 완료!")
    print()
    print("   다음 단계:")
    print("   1. 서버 재시작: launchctl stop/start com.yeomyeonggeori.raon-os")
    print("   2. 평가 실행: curl -X POST http://localhost:8400/v1/evaluate ...")
    print("   3. 피드백 전송: curl -X POST http://localhost:8400/v1/feedback ...")
    sys.exit(0)
else:
    print("  테이블이 생성됐으나 REST API로 아직 확인되지 않습니다.")
    print("   잠시 후 다시 실행해보세요 (Supabase cache refresh 필요할 수 있음).")
    sys.exit(0)
