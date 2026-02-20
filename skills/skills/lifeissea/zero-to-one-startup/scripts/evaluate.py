#!/usr/bin/env python3
"""
Raon OS — 사업계획서 평가 엔진
OpenRouter / Gemini / Anthropic / OpenAI / Ollama 자동감지 (raon_llm)

Track A: 기술창업/벤처 (TIPS 기준 100점)
Track B: 소상공인/로컬 (5항목 100점 — 입지/경험/차별화/자금/지역)
"""
from __future__ import annotations

import sys
import json
import os
import subprocess
import urllib.request
import urllib.error
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BASE_DIR = os.path.dirname(SCRIPT_DIR)
REF_DIR = os.path.join(BASE_DIR, "references")

try:
    from gamification import add_xp, format_xp_gain
    HAS_GAMIFICATION = True
except ImportError:
    HAS_GAMIFICATION = False

# TrackClassifier: 트랙 자동 감지
try:
    from track_classifier import TrackClassifier as _TrackClassifier
    _HAS_TRACK_CLASSIFIER = True
except ImportError:
    _HAS_TRACK_CLASSIFIER = False

# raon_llm: 범용 LLM 클라이언트 (.env 자동로드 포함)
from raon_llm import (
    chat as _raon_chat,
    detect_provider as _detect_provider,
    prompt_to_messages,
)

# --- Config ---
OLLAMA_MODEL = os.environ.get("OLLAMA_MODEL", "qwen3:8b")   # 하위 호환용
RAON_API_URL = os.environ.get("RAON_API_URL", "")
RAON_API_KEY = os.environ.get("RAON_API_KEY", "")


def read_ref(filename):
    path = os.path.join(REF_DIR, filename)
    try:
        with open(path, "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        return ""


def extract_pdf_text(filepath):
    """PDF에서 텍스트 추출"""
    # pdftotext
    try:
        result = subprocess.run(
            ["pdftotext", filepath, "-"],
            capture_output=True, text=True, timeout=30
        )
        if result.returncode == 0 and result.stdout.strip():
            return result.stdout
    except (FileNotFoundError, subprocess.TimeoutExpired):
        pass

    # pypdf (formerly PyPDF2)
    try:
        from pypdf import PdfReader
        with open(filepath, "rb") as f:
            reader = PdfReader(f)
            return "\n".join(page.extract_text() or "" for page in reader.pages)
    except ImportError:
        pass

    # PyPDF2 (legacy fallback)
    try:
        import PyPDF2
        with open(filepath, "rb") as f:
            reader = PyPDF2.PdfReader(f)
            return "\n".join(page.extract_text() or "" for page in reader.pages)
    except ImportError:
        pass

    # pdfplumber
    try:
        import pdfplumber
        with pdfplumber.open(filepath) as pdf:
            return "\n".join(page.extract_text() or "" for page in pdf.pages)
    except ImportError:
        pass

    print("❌ PDF 파싱 도구가 필요합니다: pdftotext, pypdf, PyPDF2, 또는 pdfplumber", file=sys.stderr)
    sys.exit(1)


def build_prompt(text, mode="evaluate", **kwargs):
    tips_criteria = read_ref("tips-criteria.md")
    
    if mode == "evaluate":
        return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서이자 참모.
사업계획서를 평가할 때는 직접적이고 솔직하게, 근데 건설적으로 피드백해.
빈말 없이 핵심만. 잘된 건 칭찬하고, 부족한 건 구체적으로 어떻게 고치면 되는지 알려줘.

## 평가 기준 (TIPS 심사 기준 기반)

{tips_criteria}

## 평가 대상 사업계획서

{text[:8000]}

## 지시사항

위 사업계획서를 실제 TIPS 심사 배점 기준으로 평가해줘.

⚠️ 중요 규칙 (반드시 준수):
1. 종합 점수 = 기술성 + 시장성 + 사업성 + 팀역량의 합계 (직접 더해서 계산)
2. 세부항목 점수는 절대 만점을 초과할 수 없음 (예: __/5 항목은 최대 5점, __/10 항목은 최대 10점)
3. 영역 점수도 배점 상한을 초과할 수 없음 (기술성 최대 30, 시장성 최대 25, 사업성 최대 25, 팀역량 최대 20)

### 🌅 라온의 사업계획서 평가

**종합 점수: __/100** (= 기술성 + 시장성 + 사업성 + 팀역량의 합)

#### 1. 기술성 (__/30)
- 기술 혁신성/차별성 (__/15):
- 기술 완성도 및 개발 역량 (__/10):
- 지식재산권 확보 가능성 (__/5):
- 💡 제안:

#### 2. 시장성 (__/25)
- 목표 시장 규모 및 성장성 (__/10):
- 경쟁 현황 및 진입 전략 (__/10):
- 고객 검증 (PoC/LOI/매출) (__/5):
- 💡 제안:

#### 3. 사업성 (__/25)
- 비즈니스 모델 타당성 (__/10):
- 수익 구조 및 확장성 (__/10):
- 재무 계획 합리성 (__/5):
- 💡 제안:

#### 4. 대표자/팀 역량 (__/20)
- 대표자 경력 및 도메인 전문성 (__/10):
- 팀 구성 적합성 (__/5):
- 실행 역량 (이전 창업/프로젝트 성과) (__/5):
- 💡 제안:

---

### 📋 핵심 요약
- **강점 TOP 3:**
- **즉시 개선 필요 TOP 3:**
- **TIPS 합격 가능성:** (높음/보통/낮음) + 이유 한줄

### 🎯 다음 스텝
1.
2.
3.

톤은 스타트업 참모처럼. 솔직하되 응원하는 느낌으로."""

    elif mode == "improve":
        return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서.
아래 사업계획서의 개선된 버전을 작성해줘.

## 평가 기준

{tips_criteria}

## 원본 사업계획서

{text[:8000]}

## 지시사항

원본의 구조를 유지하면서 다음을 개선해:
1. 문제 정의를 더 날카롭게
2. 시장 규모에 TAM-SAM-SOM 적용
3. 비즈니스 모델에 수익 구조 명확화
4. 경쟁 우위를 정량적으로
5. 재무 계획에 현실적 근거 추가

## 출력 형식

각 섹션을 아래 형식으로 작성해:

### [섹션명] [✏️ 수정됨] or [✅ 유지]

> **변경 요약:** 원본에서 뭘 왜 바꿨는지 한줄 설명
> **원본:** "원본의 핵심 문장/수치를 인용" (2-3줄)
> **변경:** "수정된 핵심 문장/수치" (2-3줄)

[개선된 전체 섹션 내용]

---

마지막에 변경 요약 테이블을 추가해:

### 📊 변경 요약

| 섹션 | 변경 수준 | 핵심 변경 |
|------|-----------|-----------|
| 문제정의 | 🔴대폭/🟡일부/🟢유지 | 한줄 설명 |
| ... | ... | ... |

톤은 비서 라온답게 — 직접적이고 실용적으로."""

    elif mode == "match":
        gov_programs = read_ref("gov-programs.md")
        return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서.
아래 사업계획서를 분석해서 최적의 정부 지원사업을 추천해줘.

## 정부 지원사업 목록

{gov_programs}

## 사업계획서

{text[:8000]}

## 매칭 평가 기준 (가중치)

각 프로그램별로 아래 5개 항목을 0~100점으로 채점하고, 가중 평균으로 최종 매칭도를 산출해:

| 항목 | 가중치 | 설명 |
|------|--------|------|
| 산업 적합성 | 25% | 프로그램 지원 대상 산업/기술 분야와 사업계획서 일치도 |
| 성장 단계 | 25% | 프로그램이 요구하는 기업 단계(예비창업/초기/성장)와 현재 단계 일치 |
| 매출/규모 조건 | 20% | 매출, 직원 수, 업력 등 정량 조건 충족 여부 |
| 기술 혁신성 | 15% | R&D 요소, 기술 차별화, 특허/IP 보유 등 |
| 정책 우선순위 | 15% | 올해 정부 정책 방향(AI, 디지털전환 등)과 부합도 |

**최종 매칭도 = (산업×0.25) + (단계×0.25) + (매출×0.20) + (기술×0.15) + (정책×0.15)**

## 지시사항

### 🌅 라온의 정부 지원사업 매칭 결과

아래 형식으로 추천해줘 (상위 3개):

**🥇 1순위: [프로그램명]**
- 최종 매칭도: __%
- 세부 점수: 산업(__) | 단계(__) | 매출(__) | 기술(__) | 정책(__)
- 이유: 
- 예상 지원 금액:
- 준비 사항:

**🥈 2순위: [프로그램명]**
- (동일 형식)

**🥉 3순위: [프로그램명]**
- (동일 형식)

### ⚠️ 지원 전 체크리스트
- 

### 📅 추천 지원 타임라인
-

실제 심사위원 관점에서 솔직하게 매칭도를 평가해줘. 점수는 근거를 들어 정확히 산출하고, 80% 이상이면 강력 추천."""

    elif mode == "checklist":
        gov_programs = read_ref("gov-programs.md")
        program = kwargs.get("program", "TIPS")
        return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서이자 참모.
아래 사업계획서를 분석해서 '{program}' 지원 전 준비 상태를 체크리스트로 점검해줘.

## 프로그램 정보

{gov_programs}

## TIPS 심사 기준 (참고)

{tips_criteria}

## 사업계획서

{text[:8000]}

## 지시사항

### 🌅 라온의 {program} 지원 준비 체크리스트

사업계획서에서 확인 가능한 항목은 ✅, 누락/부족한 항목은 ❌, 부분적인 항목은 ⚠️로 표시해:

#### 📋 필수 서류
| 항목 | 상태 | 비고 |
|------|------|------|
| 사업계획서 | ✅/❌/⚠️ | |
| 법인등기부등본 | ✅/❌/⚠️ | 사업계획서에서 확인 불가 시 "[확인 필요]" |
| 재무제표 | ✅/❌/⚠️ | |
| 대표자 이력서 | ✅/❌/⚠️ | |
| 기술설명서/특허 | ✅/❌/⚠️ | |

#### 📊 사업계획서 내용 점검
| 항목 | 상태 | 현재 수준 | 개선 방향 |
|------|------|-----------|-----------|
| 문제 정의 | ✅/❌/⚠️ | | |
| TAM-SAM-SOM | ✅/❌/⚠️ | | |
| 경쟁 분석 | ✅/❌/⚠️ | | |
| 비즈니스 모델 | ✅/❌/⚠️ | | |
| 수익 구조 | ✅/❌/⚠️ | | |
| 고객 검증 (PoC/LOI/매출) | ✅/❌/⚠️ | | |
| 팀 역량 | ✅/❌/⚠️ | | |
| 재무 계획 | ✅/❌/⚠️ | | |
| R&D 계획 | ✅/❌/⚠️ | | |
| 마일스톤/일정 | ✅/❌/⚠️ | | |
| 지식재산권 | ✅/❌/⚠️ | | |

#### 🎯 {program} 특화 요건
- (프로그램별 특수 요건을 체크)

### 📊 준비도 요약
- **준비 완료:** __개 / 전체 __개
- **즉시 보완 필요 (❌):** 
- **부분 보완 필요 (⚠️):**
- **예상 준비 기간:** 

### 🚀 우선 조치 사항 (중요도순)
1.
2.
3.

솔직하게 평가해줘. 부족한 건 부족하다고 하고, 어떻게 보완하면 되는지 구체적으로 알려줘."""

    elif mode == "draft":
        gov_programs = read_ref("gov-programs.md")
        tips_criteria = read_ref("tips-criteria.md")
        program = kwargs.get("program", "TIPS")
        return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서이자 참모.
아래 사업계획서를 기반으로 '{program}' 지원서 초안을 작성해줘.

## 프로그램 정보

{gov_programs}

## TIPS 심사 기준 (참고)

{tips_criteria}

## 원본 사업계획서

{text[:8000]}

## 지시사항

'{program}' 지원서에 필요한 주요 섹션을 작성해줘:

### 🌅 라온의 {program} 지원서 초안

#### 1. 사업 개요
- 기업명/대표자:
- 주요 제품/서비스:
- 핵심 기술:
- 창업일/업력:

#### 2. 기술 혁신성
- 핵심 기술 설명 (기존 대비 차별점):
- 기술 완성도 (TRL 단계):
- 지식재산권 현황/계획:

#### 3. 시장 분석
- TAM/SAM/SOM:
- 목표 시장 및 성장률:
- 경쟁 현황 및 차별화 전략:

#### 4. 사업화 전략
- 비즈니스 모델:
- 수익 구조:
- 고객 확보 전략:
- 마일스톤 (6개월/12개월/24개월):

#### 5. 팀 역량
- 대표자 이력 및 도메인 전문성:
- 핵심 팀원:
- 외부 자문/멘토:

#### 6. 재무 계획
- 소요 자금 및 용도:
- 매출 전망 (1~3년):
- 손익분기점:

#### 7. R&D 계획 (해당 시)
- 연구 목표:
- 추진 일정:
- 예상 산출물:

---

### ⚠️ 작성 시 주의사항
- 원본에 없는 수치는 "[확인 필요]"로 표시
- 과장하지 말고 근거 있는 서술만
- 심사위원이 중요하게 보는 포인트를 강조

### 📋 추가 준비 서류 체크리스트
- 

톤은 공식 지원서답게 격식체로. 단, 내용은 실질적이고 구체적으로."""

    elif mode == "investor":
        return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서.
아래 사업계획서를 분석하여 투자자(VC/AC/Angel) 관점에서 매력적인 투자 프로필을 작성해줘.

## 사업계획서

{text[:8000]}

## 지시사항

투자자가 3분 안에 이 스타트업을 파악할 수 있도록 핵심 정보를 요약하고, 적합한 투자자 유형을 추천해줘.

### 🌅 라온의 투자자 매칭 프로필

#### 1. 📋 Deal Summary (투자자용 1분 요약)
- **Problem:** (한 문장)
- **Solution:** (한 문장)
- **Traction:** (핵심 지표: 매출, 유저, 특허 등)
- **Ask:** (투자 유치 희망 금액 및 밸류에이션 추정 - 정보 없으면 '[정보 필요]')

#### 2. 🎯 타겟 투자자 유형
- **단계:** (Seed / Pre-A / Series A)
- **섹터:** (SaaS / AI / Bio / Commerce 등)
- **추천 투자자 유형:** (AC / Micro VC / CVC / 전략적 투자자)
- **이유:** (왜 이 유형의 투자자가 관심을 가질지)

#### 3. 💎 Investment Highlights (투자 포인트)
1.
2.
3.

#### 4. 🚩 Red Flags (투자자가 우려할 점 & 방어 논리)
- **우려점:** 
  - **방어 논리:** 
- **우려점:** 
  - **방어 논리:**

#### 5. 🗣️ 피칭 팁
- IR 미팅 시 강조할 점:
- 보완이 필요한 자료:

톤은 전문적인 투자 심사역 보고서처럼 객관적이고 날카롭게."""



def call_ollama(prompt, model=None):
    """
    LLM 호출. raon_llm.chat()으로 위임.
    프로바이더 자동감지: OpenRouter → Gemini → Anthropic → OpenAI → Ollama
    model 인수는 RAON_MODEL 환경변수로 override 가능.
    """
    return _raon_chat(prompt_to_messages(prompt), model=model or None)


def call_raon_api(text, mode):
    """K-Startup AI API 호출"""
    data = json.dumps({
        "action": mode,
        "text": text[:10000]
    }).encode("utf-8")

    req = urllib.request.Request(
        f"{RAON_API_URL}/v1/biz-plan/{mode}",
        data=data,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {RAON_API_KEY}"
        }
    )

    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return resp.read().decode("utf-8")
    except Exception:
        return None


def build_comparison_prompt(docs, mode):
    """다중 문서 비교 프롬프트 생성"""
    docs_text = ""
    for i, doc in enumerate(docs):
        # 텍스트 길이 제한 (문서당 5000자)
        docs_text += f"\n\n### 문서 {i+1}: {doc['name']}\n{doc['text'][:5000]}\n"

    base_prompt = f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서.
아래 {len(docs)}개의 사업계획서를 비교 분석해줘.

{docs_text}

## 지시사항
"""

    if mode == "evaluate":
        base_prompt += """
각 문서의 강점과 약점을 비교하고, TIPS 선정 가능성이 더 높은 문서를 추천해줘.
평가 기준: 기술성(30), 시장성(25), 사업성(25), 팀 역량(20).

형식:
1. **요약 비교**: (각 항목별 점수 비교 표)
2. **문서별 상세 분석**:
   - 문서 1: (강점/약점)
   - 문서 2: (강점/약점)
3. **최종 추천 및 이유**: 왜 이 문서가 더 나은지 설명.
"""
    elif mode == "improve":
        base_prompt += """
두 문서의 장점을 결합하여 더 완벽한 사업계획서를 만들기 위한 조언을 해줘.
어떤 부분을 어느 문서에서 가져와야 할지 구체적으로 설명해.
"""
    elif mode == "match":
        base_prompt += """
각 문서의 사업 아이템에 따라 적합한 정부 지원사업이 다를 수 있어.
각각 최적의 지원사업(TIPS, 예비창업, 초기창업 등)을 매칭해줘.
"""
    else:
        base_prompt += "두 문서를 비교 분석해줘."
    
    return base_prompt


def parse_score(text):
    """Extract 종합 점수: XX/100 from LLM output. Validate against sub-scores sum."""
    if not text:
        return None
    import re

    # Extract sub-scores first
    sub_patterns = [
        (r'기술성[^(]*\((\d+)/30\)', 30),
        (r'시장성[^(]*\((\d+)/25\)', 25),
        (r'사업성[^(]*\((\d+)/25\)', 25),
        (r'(?:대표자|팀)[^(]*\((\d+)/20\)', 20),
    ]
    sub_total = 0
    sub_found = 0
    for pat, max_score in sub_patterns:
        m = re.search(pat, text)
        if m:
            s = int(m.group(1))
            if 0 <= s <= max_score:
                sub_total += s
                sub_found += 1

    # Extract stated total score
    stated_score = None
    for pattern in [
        r'종합\s*점수[:\s]*(\d+)\s*/\s*100',
        r'총점[:\s]*(\d+)\s*/\s*100',
        r'\*\*(\d+)\s*/\s*100\*\*',
    ]:
        m = re.search(pattern, text)
        if m:
            score = int(m.group(1))
            if 0 <= score <= 100:
                stated_score = score
                break

    # If all 4 sub-scores found, use their sum (more reliable)
    if sub_found == 4 and 0 <= sub_total <= 100:
        return sub_total

    return stated_score


# ─── Track B 소상공인 평가 ────────────────────────────────────────────────────

TRACK_B_EVAL_PROMPT = """당신은 소상공인 창업 전문 컨설턴트 '라온'입니다.
절대 "기술성", "R&D", "혁신성" 기준으로 평가하지 마세요.
아래 사업 내용을 5가지 기준으로 평가하세요.

## 평가 대상

{text}

## 평가 기준 (소상공인 Track B, 100점 만점)

| 항목 | 배점 | 내용 |
|------|------|------|
| 입지/상권 적합성 | 25점 | 목표 상권과 업종 매칭, 접근성, 경쟁 밀도 |
| 대표자 경험/전문성 | 20점 | 해당 업종 경력, 운영 경험, 전문 자격 |
| 서비스/상품 차별화 | 25점 | 다른 가게보다 뭐가 더 특별한지 |
| 자금 조달 계획 | 15점 | 초기 자본, 운영 자금, 정부 지원 활용 계획 |
| 지역 특화 가능성 | 15점 | 지역 특색 반영, 단골 고객 확보 전략 |

## 출력 형식

### 라온의 소상공인 창업 평가 (Track B)

**종합 점수: __/100**

#### 1. 입지/상권 적합성 (__/25)
- 현황:
- 제안: "어떤 동네/상권에서 시작할 계획인가요?"

#### 2. 대표자 경험/전문성 (__/20)
- 현황:
- 제안: "이 업종 관련 경험이 있으신가요? 몇 년이나 하셨나요?"

#### 3. 서비스/상품 차별화 (__/25)
- 현황:
- 제안: "다른 가게보다 뭐가 더 특별한가요? 손님이 여기 와야 하는 이유는?"

#### 4. 자금 조달 계획 (__/15)
- 현황:
- 제안: "초기 자금은 어떻게 마련할 계획인가요? 소진공 정책자금도 알아보셨나요?"

#### 5. 지역 특화 가능성 (__/15)
- 현황:
- 제안: "이 지역 단골손님을 어떻게 만들 계획인가요?"

---

### 핵심 요약
- **잘 준비된 부분:**
- **지금 당장 보완할 부분:**
- **소진공/정책자금 활용 가능성:**

### 다음 3가지 할 일
1.
2.
3.

쉬운 말로 설명하고, "기술성" 같은 어려운 용어는 절대 쓰지 마세요."""


def build_track_b_prompt(text):
    # type: (str) -> str
    """소상공인 Track B 전용 평가 프롬프트."""
    return TRACK_B_EVAL_PROMPT.format(text=text[:8000])


def evaluate_track_b(text, model=None):
    # type: (str, object) -> str
    """
    소상공인 Track B 전용 평가 (5항목 100점).
    쉬운 한국어 용어 사용, 기술성 기준 제외.
    """
    prompt = build_track_b_prompt(text)
    result = None

    if RAON_API_URL and RAON_API_KEY:
        result = call_raon_api(text, "evaluate")

    if not result:
        result = call_ollama(prompt, model)

    return result or ""


def evaluate_track_a(text, model=None):
    # type: (str, object) -> str
    """
    기술창업 Track A 평가 (기존 TIPS 기준 100점).
    기존 build_prompt evaluate 로직 위임.
    """
    prompt = build_prompt(text, mode="evaluate")
    result = None

    if RAON_API_URL and RAON_API_KEY:
        result = call_raon_api(text, "evaluate")

    if not result:
        result = call_ollama(prompt, model)

    return result or ""


def evaluate(text, model=None, track=None):
    # type: (str, object, object) -> str
    """
    트랙 자동 감지 후 적절한 평가 실행.

    Args:
        text: 사업계획서 텍스트
        model: LLM 모델명 (None이면 자동)
        track: "A" | "B" | "AB" (None이면 자동 감지)

    Returns:
        평가 결과 텍스트
    """
    # 트랙 자동 감지
    if track is None and _HAS_TRACK_CLASSIFIER:
        try:
            clf = _TrackClassifier()
            track = clf.classify(text)
        except Exception as e:
            print("[evaluate] TrackClassifier 실패: %s" % e, file=sys.stderr)
            track = "A"  # 기본값 Track A (기존 동작 유지)

    if track == "B":
        return evaluate_track_b(text, model)
    if track == "AB":
        # AB는 Track B 우선 (소상공인 접근성 우선)
        return evaluate_track_b(text, model)

    # Track A (기본)
    return evaluate_track_a(text, model)


def fix_score_text(text):
    """Fix scores in LLM output: clamp sub-items, recalculate section totals, fix overall total."""
    if not text:
        return text
    import re

    # Step 1: Clamp sub-item scores that exceed max (e.g., 7/5 → 5/5)
    def clamp_sub_item(m):
        score = int(m.group(1))
        max_s = int(m.group(2))
        if score > max_s:
            print("⚠️ 세부항목 교정: %d/%d → %d/%d" % (score, max_s, max_s, max_s), file=sys.stderr)
            return "%d/%d" % (max_s, max_s)
        return m.group(0)

    text = re.sub(r'(\d+)/(\d+)\)', lambda m: clamp_sub_item(m) + ')', text)

    # Step 2: Recalculate section scores from sub-items
    section_configs = [
        ('기술성', 30, [15, 10, 5]),
        ('시장성', 25, [10, 10, 5]),
        ('사업성', 25, [10, 10, 5]),
        ('(?:대표자|팀)[^(]*?역량', 20, [10, 5, 5]),
    ]
    section_totals = []
    for section_name, section_max, item_maxes in section_configs:
        section_pat = r'(%s[^(]*\()(\d+)(/%d\))' % (section_name, section_max)
        m = re.search(section_pat, text)
        if m:
            stated = int(m.group(2))
            if stated > section_max:
                text = text[:m.start(2)] + str(section_max) + text[m.end(2):]
                print("⚠️ 영역 교정: %s %d → %d" % (section_name, stated, section_max), file=sys.stderr)
                stated = section_max
            section_totals.append(min(stated, section_max))
        else:
            # Try to find it anyway for total calculation
            alt_pat = r'%s[^(]*\((\d+)/%d\)' % (section_name, section_max)
            alt_m = re.search(alt_pat, text)
            if alt_m:
                section_totals.append(min(int(alt_m.group(1)), section_max))

    # Step 3: Fix overall total to match section sum
    if len(section_totals) == 4:
        correct_total = sum(section_totals)
        if correct_total > 100:
            correct_total = 100
        for pattern in [
            r'(종합\s*점수[:\s]*)(\d+)(\s*/\s*100)',
            r'(총점[:\s]*)(\d+)(\s*/\s*100)',
            r'(\*\*)(\d+)(\s*/\s*100\*\*)',
        ]:
            m = re.search(pattern, text)
            if m:
                stated = int(m.group(2))
                if stated != correct_total:
                    text = text[:m.start(2)] + str(correct_total) + text[m.end(2):]
                    print("⚠️ 총점 교정: %d → %d (부분점수 합산 기준)" % (stated, correct_total), file=sys.stderr)
                break

    return text


def build_followup_prompt(history, followup_question, original_text):
    """대화 히스토리 기반 후속 질문 프롬프트 생성"""
    history_text = ""
    for turn in history:
        role = "사용자" if turn["role"] == "user" else "라온"
        history_text += f"\n[{role}]: {turn['content'][:2000]}\n"

    return f"""너는 '라온'이야. 한국 스타트업 파운더의 AI 비서이자 참모.
이전에 사업계획서를 평가했고, 파운더가 후속 질문을 하고 있어.
이전 대화 맥락을 유지하면서 답변해줘.

## 원본 사업계획서 (요약)

{original_text[:4000]}

## 이전 대화

{history_text}

## 후속 질문

{followup_question}

## 지시사항

이전 평가 내용과 사업계획서를 참고해서 후속 질문에 답변해줘.
구체적이고 실용적인 조언을 해줘. 톤은 스타트업 참모처럼 솔직하되 응원하는 느낌으로."""


def run_interactive(file_path=None, text=None, model=None):
    """대화형 평가 세션 실행"""
    model_name = model or OLLAMA_MODEL

    # 텍스트 추출
    if file_path:
        if file_path.lower().endswith(".pdf"):
            print(f"📄 PDF 파싱 중: {os.path.basename(file_path)}", file=sys.stderr)
            original_text = extract_pdf_text(file_path)
        else:
            with open(file_path, "r", encoding="utf-8") as f:
                original_text = f.read()
    elif text:
        original_text = text
    else:
        print("❌ --file 또는 --text를 입력하세요.", file=sys.stderr)
        sys.exit(1)

    if not original_text.strip():
        print("❌ 빈 문서입니다.", file=sys.stderr)
        sys.exit(1)

    # 초기 평가
    print(f"📊 초기 평가 시작 ({len(original_text)}자)...", file=sys.stderr)
    print(f"🤖 AI 엔진 구동 중 ({model_name})...", file=sys.stderr)

    prompt = build_prompt(original_text, mode="evaluate")
    initial_result = None

    if RAON_API_URL and RAON_API_KEY:
        initial_result = call_raon_api(original_text, "evaluate")
    if not initial_result:
        initial_result = call_ollama(prompt, model_name)

    if not initial_result:
        print("⚠️ LLM 연결 실패.", file=sys.stderr)
        sys.exit(1)

    print(initial_result)
    print("\n" + "=" * 60, file=sys.stderr)
    print("💬 대화형 모드: 후속 질문을 입력하세요 (종료: quit/exit/q)", file=sys.stderr)
    print("=" * 60, file=sys.stderr)

    history = [
        {"role": "assistant", "content": initial_result},
    ]

    while True:
        try:
            question = input("\n🗣️  > ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\n👋 대화 종료", file=sys.stderr)
            break

        if not question:
            continue
        if question.lower() in ("quit", "exit", "q", "종료"):
            print("👋 대화 종료", file=sys.stderr)
            break

        history.append({"role": "user", "content": question})

        followup_prompt = build_followup_prompt(history, question, original_text)
        print(f"🤖 답변 생성 중...", file=sys.stderr)

        answer = call_ollama(followup_prompt, model_name)
        if not answer:
            print("⚠️ LLM 응답 실패. 다시 시도해주세요.", file=sys.stderr)
            history.pop()  # remove failed user message
            continue

        history.append({"role": "assistant", "content": answer})
        print(answer)

    return history


def main():
    import argparse
    parser = argparse.ArgumentParser(description="🌅 Raon OS — 사업계획서 평가")
    parser.add_argument("--version", "-V", action="version", version="%(prog)s 0.3.7")
    parser.add_argument("mode", choices=["evaluate", "improve", "match", "draft", "checklist", "investor", "interactive"], help="실행 모드")
    parser.add_argument("--program", help="지원사업 프로그램명 (draft 모드용)")
    parser.add_argument("--file", "-f", action="append", help="PDF 파일 경로 (여러 번 사용 가능)")
    parser.add_argument("--text", help="직접 텍스트 입력")
    parser.add_argument("--model", default=OLLAMA_MODEL, help=f"Ollama 모델 (기본: {OLLAMA_MODEL})")
    parser.add_argument("--json", action="store_true", help="JSON 출력")
    parser.add_argument("--output", "-o", help="결과 저장 파일 경로")
    args = parser.parse_args()

    model_name = args.model
    start_time = time.time()

    # Interactive mode
    if args.mode == "interactive":
        file_path = args.file[0] if args.file else None
        run_interactive(file_path=file_path, text=args.text, model=model_name)
        return

    if args.mode in ("draft", "checklist") and not args.program:
        print(f"❌ {args.mode} 모드에는 --program 옵션이 필요합니다. (예: --program TIPS)", file=sys.stderr)
        sys.exit(1)

    # 텍스트 추출 (다중 파일 지원)
    docs = []
    if args.file:
        for fpath in args.file:
            if not os.path.exists(fpath):
                print(f"❌ 파일을 찾을 수 없습니다: {fpath}", file=sys.stderr)
                sys.exit(1)
            
            content = ""
            if fpath.lower().endswith(".pdf"):
                print(f"📄 PDF 파싱 중: {os.path.basename(fpath)}", file=sys.stderr)
                content = extract_pdf_text(fpath)
            else:
                with open(fpath, "r", encoding="utf-8") as f:
                    content = f.read()
            docs.append({"name": os.path.basename(fpath), "text": content})

    if args.text:
        docs.append({"name": "직접 입력", "text": args.text})

    if not docs:
        # stdin
        if not sys.stdin.isatty():
            docs.append({"name": "STDIN", "text": sys.stdin.read()})
        else:
            print("❌ --file 또는 --text를 입력하세요 (또는 파이프 입력).", file=sys.stderr)
            sys.exit(1)

    # 빈 문서 체크
    valid_docs = [d for d in docs if d["text"].strip()]
    if not valid_docs:
        print("❌ 빈 문서입니다.", file=sys.stderr)
        sys.exit(1)
    
    total_len = sum(len(d["text"]) for d in valid_docs)
    print(f"📊 분석 시작 ({len(valid_docs)}개 문서, 총 {total_len}자)...", file=sys.stderr)

    # 다중 문서 비교 모드 (문서가 2개 이상일 때)
    is_comparison = len(valid_docs) > 1

    result = None

    # 1순위: K-Startup AI API
    if RAON_API_URL and RAON_API_KEY:
        if is_comparison:
            print("⚠️ API 모드에서는 다중 문서 비교를 지원하지 않습니다. 첫 번째 문서만 분석합니다.", file=sys.stderr)
        
        print("🔗 K-Startup AI 엔진 연결 중...", file=sys.stderr)
        result = call_raon_api(valid_docs[0]["text"], args.mode)

    # 2순위: Ollama (로컬 LLM) — API 실패 시 fallback
    if not result:
        if is_comparison:
            prompt = build_comparison_prompt(valid_docs, args.mode)
            print(f"🤖 AI 엔진 구동 중 ({model_name}) — 비교 분석 모드...", file=sys.stderr)
        else:
            prompt = build_prompt(valid_docs[0]["text"], args.mode, program=getattr(args, 'program', None) or 'TIPS')
            print(f"🤖 AI 엔진 구동 중 ({model_name})...", file=sys.stderr)
        
        result = call_ollama(prompt, model_name)

    # Fix score mismatch in text and extract score
    if result:
        result = fix_score_text(result)
    score = parse_score(result) if result else None

    # Logging
    try:
        log_file = os.path.join(BASE_DIR, "history.jsonl")
        duration = time.time() - start_time
        log_entry = {
            "timestamp": int(start_time),
            "mode": args.mode,
            "model": model_name,
            "input_len": total_len,
            "duration": round(duration, 2),
            "status": "success" if result else "fail"
        }
        if score is not None:
            log_entry["score"] = score
        # Store filename(s) for reference
        if valid_docs:
            log_entry["files"] = [d.get("name", "unknown") for d in valid_docs]
        with open(log_file, "a", encoding="utf-8") as f:
            f.write(json.dumps(log_entry) + "\n")
    except Exception:
        pass

    # Gamification
    gami_result = None
    if result and HAS_GAMIFICATION and args.mode in ("evaluate", "improve", "match", "draft", "checklist", "valuation"):
        action = args.mode
        if action == "checklist":
            action = "checklist_complete"
        ctx = {}
        if score is not None:
            ctx["score"] = score
        try:
            gami_result = add_xp(action, ctx)
        except Exception:
            pass

    if result:
        json_payload = {
            "status": "ok",
            "mode": args.mode,
            "model": model_name,
            "text_length": total_len,
            "doc_count": len(valid_docs),
            "result": result
        }
        if score is not None:
            json_payload["score"] = score
        if gami_result:
            json_payload["xp_gained"] = gami_result["xp_gained"]
            json_payload["level"] = gami_result["level"]
            json_payload["title"] = gami_result["title"]
            json_payload["new_badges"] = gami_result["new_badges"]

        if args.output:
            try:
                content = json.dumps(json_payload, ensure_ascii=False, indent=2) if args.json else result
                with open(args.output, "w", encoding="utf-8") as f:
                    f.write(content)
                print(f"✅ 결과 저장 완료: {args.output}", file=sys.stderr)
            except Exception as e:
                print(f"❌ 파일 저장 실패: {e}", file=sys.stderr)
                sys.exit(1)
        else:
            if args.json:
                print(json.dumps(json_payload, ensure_ascii=False, indent=2))
            else:
                print(result)
                if gami_result and gami_result.get("xp_gained", 0) > 0:
                    print(format_xp_gain(
                        gami_result["xp_gained"],
                        gami_result.get("new_badges"),
                        gami_result.get("leveled_up", False),
                        gami_result.get("title"),
                    ))
    else:
        prov = _detect_provider()
        print("⚠️ LLM 연결 실패. 다음을 확인하세요:", file=sys.stderr)
        print(f"   현재 감지된 프로바이더: {prov}", file=sys.stderr)
        print(f"   1. API 키 설정: ~/.openclaw/.env 에 OPENROUTER_API_KEY, GEMINI_API_KEY 등", file=sys.stderr)
        print(f"   2. 로컬 Ollama: ollama pull {model_name}", file=sys.stderr)
        print(f"   3. K-Startup AI API: RAON_API_URL, RAON_API_KEY 설정", file=sys.stderr)
        print(f"   4. 진단: python3 scripts/raon_llm.py --detect", file=sys.stderr)
        
        if args.json:
            print(json.dumps({"error": "no_llm_available", "text_length": total_len}))
        else:
            fallback_prompt = build_comparison_prompt(valid_docs, args.mode) if is_comparison else build_prompt(valid_docs[0]["text"], args.mode)
            print("\n--- 아래 프롬프트를 LLM에 직접 붙여넣어도 됩니다 ---\n")
            print(fallback_prompt)
        sys.exit(1)


if __name__ == "__main__":
    main()
