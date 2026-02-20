#!/usr/bin/env python3
"""
Raon OS — Agentic RAG 테스트 (18개)

테스트 커버리지:
- QueryRouter 분류 (factual/search/realtime/multistep)
- HyDE retrieve 동작 (정상 + LLM 실패)
- Multi-Query Fusion RRF + 중복 제거
- Speculative RAG 흐름
- Recursive RAG 반복 검색
- CRAG Critic 평가 (정상 + 실패)
- Tools (search_gov_programs, check_eligibility, fetch_realtime)
- AgenticRAG.run() 통합 (factual/search/multistep + 폴백)
- structured_extractor (schema 추출 + filter)

Python 3.9+ compatible
"""

from __future__ import annotations  # Python 3.9 compatibility

import json
import sys
from pathlib import Path
from typing import Any
from unittest.mock import patch, MagicMock, call

import pytest

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

from agentic_rag import (
    QueryRouter,
    hyde_retrieve,
    multi_query_retrieve,
    speculative_retrieve,
    recursive_retrieve,
    crag_critic,
    _reciprocal_rank_fusion,
    Tools,
    AgenticRAG,
    _rag_search,
)
from structured_extractor import (
    extract_program_schema,
    filter_programs,
    _rule_based_extract,
    _default_schema,
)


# ─── Mock RAG ────────────────────────────────────────────────────────────────

class MockRAG:
    """테스트용 Mock RAG 객체."""

    def __init__(self, docs: Any = None) -> None:
        self.docs = docs if docs is not None else [
            {
                "text": "TIPS 프로그램은 민간 투자를 받은 스타트업을 지원합니다. 창업 7년 이내 기업만 가능.",
                "meta": {"type": "gov_program", "program": "TIPS"},
                "score": 0.9,
            },
            {
                "text": "초기창업패키지는 창업 3년 이내 기업이 신청할 수 있습니다. 최대 1억원 지원.",
                "meta": {"type": "gov_program", "program": "초기창업패키지"},
                "score": 0.8,
            },
            {
                "text": "예비창업패키지 합격 후기: 사업성과 기술성 위주로 평가됩니다. 팀 구성이 중요.",
                "meta": {"type": "success_case", "program": "예비창업패키지"},
                "score": 0.75,
            },
            {
                "text": "창업도약패키지 심사기준: 성장가능성 40%, 실현가능성 30%, 문제해결력 30%.",
                "meta": {"type": "criteria", "criteria": "심사기준"},
                "score": 0.7,
            },
        ]
        self.search_call_count = 0

    def hybrid_search(self, query: str, top_k: int = 5) -> list:
        self.search_call_count += 1
        return self.docs[:top_k]

    def search(self, query: str, top_k: int = 5) -> list:
        self.search_call_count += 1
        return self.docs[:top_k]


# ─── Test 1: QueryRouter — factual 분류 (heuristic) ──────────────────────────

def test_query_router_factual_heuristic():
    """자격조건 키워드 → factual 분류."""
    router = QueryRouter()
    result = router._heuristic_classify("TIPS 지원 자격조건이 뭐야?")
    assert result == "factual", f"Expected 'factual', got '{result}'"


# ─── Test 2: QueryRouter — realtime 분류 ──────────────────────────────────────

def test_query_router_realtime_heuristic():
    """현재/접수중 키워드 → realtime 분류."""
    router = QueryRouter()
    result = router._heuristic_classify("TIPS 현재 접수중인가요?")
    assert result == "realtime", f"Expected 'realtime', got '{result}'"


# ─── Test 3: QueryRouter — multistep 분류 ────────────────────────────────────

def test_query_router_multistep_heuristic():
    """전략/단계 키워드 → multistep 분류."""
    router = QueryRouter()
    result = router._heuristic_classify("TIPS 합격 전략을 단계별로 알려줘")
    assert result == "multistep", f"Expected 'multistep', got '{result}'"


# ─── Test 4: QueryRouter — search 기본 폴백 ──────────────────────────────────

def test_query_router_search_default():
    """일반 질문 → search 분류 (default)."""
    router = QueryRouter()
    result = router._heuristic_classify("좋은 창업 지원사업 추천해줘")
    assert result == "search", f"Expected 'search', got '{result}'"


# ─── Test 5: QueryRouter — LLM 성공 시 결과 사용 ─────────────────────────────

def test_query_router_llm_success():
    """LLM이 유효한 유형을 반환하면 해당 값 사용."""
    router = QueryRouter()
    with patch("agentic_rag.chat", return_value="  factual  "):
        result = router.classify("TIPS 제출서류는?")
    assert result == "factual"


# ─── Test 6: QueryRouter — LLM 실패 시 heuristic 폴백 ───────────────────────

def test_query_router_llm_failure_fallback():
    """LLM 예외 발생 시 heuristic으로 폴백 (valid 타입 반환)."""
    router = QueryRouter()
    with patch("agentic_rag.chat", side_effect=Exception("LLM 연결 실패")):
        result = router.classify("TIPS 자격요건은?")
    assert result in ("factual", "search", "realtime", "multistep")


# ─── Test 7: HyDE Retrieve — 기본 동작 ───────────────────────────────────────

def test_hyde_retrieve_basic():
    """HyDE: 가상 문서로 실제 문서 검색 — 결과 반환."""
    mock_rag = MockRAG()
    with patch(
        "agentic_rag.chat",
        return_value="TIPS는 민간투자를 기반으로 R&D를 지원하는 프로그램입니다.",
    ):
        results = hyde_retrieve("TIPS 자격조건은?", mock_rag, top_k=3)

    assert isinstance(results, list), "결과가 list여야 함"
    assert len(results) > 0, "최소 1개 결과 반환"
    assert mock_rag.search_call_count >= 1, "검색이 호출돼야 함"


# ─── Test 8: HyDE Retrieve — LLM None 반환 시 원본 쿼리 폴백 ────────────────

def test_hyde_retrieve_llm_none_fallback():
    """LLM이 None 반환 시 원본 쿼리로 폴백해도 검색 결과 반환."""
    mock_rag = MockRAG()
    with patch("agentic_rag.chat", return_value=None):
        results = hyde_retrieve("TIPS 자격조건은?", mock_rag, top_k=3)
    assert isinstance(results, list)
    # 원본 쿼리로 검색하므로 결과 존재
    assert len(results) > 0


# ─── Test 9: Reciprocal Rank Fusion ──────────────────────────────────────────

def test_reciprocal_rank_fusion_ordering():
    """RRF: 두 목록 모두 1위인 문서가 최상위 점수 획득."""
    doc_a = {"text": "Document A — TIPS 자격", "meta": {}, "score": 0.9}
    doc_b = {"text": "Document B — 초기창업패키지", "meta": {}, "score": 0.8}
    doc_c = {"text": "Document C — 예비창업패키지", "meta": {}, "score": 0.7}

    list1 = [doc_a, doc_b, doc_c]
    list2 = [doc_a, doc_c, doc_b]

    fused = _reciprocal_rank_fusion([list1, list2])

    assert len(fused) > 0
    assert fused[0]["text"] == "Document A — TIPS 자격", (
        f"doc_a가 1위여야 하는데, 실제: {fused[0]['text']}"
    )


# ─── Test 10: Multi-Query Fusion — 중복 제거 ─────────────────────────────────

def test_multi_query_fusion_dedup():
    """Multi-query fusion: 동일 문서 중복 제거 검증."""
    mock_rag = MockRAG()

    with patch(
        "agentic_rag.chat",
        return_value="TIPS 선정 기준\nTIPS 제외 대상\nTIPS 심사 항목",
    ):
        results = multi_query_retrieve("TIPS 자격", mock_rag, n_variants=3, top_k=5)

    # 텍스트 앞 100자를 ID로 중복 체크
    ids = [r.get("text", "")[:100] for r in results]
    assert len(ids) == len(set(ids)), f"중복 문서 발견: {len(ids)} vs {len(set(ids))}"


# ─── Test 11: Speculative RAG — 초안 + 검색 결과 반환 ────────────────────────

def test_speculative_retrieve_returns_draft_and_docs():
    """Speculative RAG: (draft_answer, docs) 튜플 반환."""
    mock_rag = MockRAG()
    call_count = [0]

    def mock_chat(messages: list) -> str:
        call_count[0] += 1
        if call_count[0] == 1:
            return "TIPS는 민간투자 기반 R&D 지원 프로그램입니다."
        return "TIPS 자격조건 심사기준"

    with patch("agentic_rag.chat", side_effect=mock_chat):
        draft, docs = speculative_retrieve("TIPS 지원 방법", mock_rag, top_k=3)

    assert isinstance(draft, str), "초안이 string이어야 함"
    assert len(draft) > 0, "초안이 비어있으면 안됨"
    assert isinstance(docs, list), "검색 결과가 list여야 함"
    assert len(docs) > 0, "검색 결과가 비어있으면 안됨"


# ─── Test 12: CRAG Critic — 정상 응답 파싱 ───────────────────────────────────

def test_crag_critic_valid_json():
    """CRAG Critic: valid JSON 응답 파싱 및 action 검증."""
    mock_resp = '{"relevant": 0.8, "sufficient": 0.7, "confident": 0.9, "action": "use"}'

    with patch("agentic_rag.chat", return_value=mock_resp):
        result = crag_critic("TIPS 자격", "TIPS 관련 문서들...", "TIPS는...")

    assert "relevant" in result
    assert "action" in result
    assert result["action"] in ("use", "refine", "retry_different")
    assert 0.0 <= result["relevant"] <= 1.0
    assert result["relevant"] == pytest.approx(0.8)


# ─── Test 13: CRAG Critic — LLM 실패 시 기본값 반환 ──────────────────────────

def test_crag_critic_llm_failure_default():
    """CRAG Critic: LLM 예외 발생 시 safe default 반환."""
    with patch("agentic_rag.chat", side_effect=Exception("LLM 실패")):
        result = crag_critic("질문", "컨텍스트", "답변")

    assert "relevant" in result
    assert "action" in result
    assert result["action"] == "use"  # safe default
    assert 0.0 <= result["relevant"] <= 1.0


# ─── Test 14: Tools.search_gov_programs — 기본 동작 ──────────────────────────

def test_tools_search_gov_programs_no_crash():
    """search_gov_programs: eval_data 없어도 크래시 없이 빈 리스트 반환."""
    tools = Tools()
    # eval_data가 없거나 비어있어도 예외 없이 반환
    results = tools.search_gov_programs(keywords=["TIPS", "스타트업"])
    assert isinstance(results, list)


# ─── Test 15: Tools.fetch_realtime — 연결 불가 폴백 ──────────────────────────

def test_tools_fetch_realtime_unreachable():
    """fetch_realtime: 허용된 도메인 + 연결 불가 시 '실시간 조회 불가' 반환."""
    tools = Tools()
    # 허용된 도메인이지만 포트 19999는 연결 불가 → '실시간 조회 불가' 반환
    result = tools.fetch_realtime("http://k-startup.go.kr:19999/nonexistent_raon_test")
    assert result == "실시간 조회 불가"


def test_tools_fetch_realtime_disallowed_domain():
    """fetch_realtime: 허용되지 않은 도메인은 ValueError 발생."""
    tools = Tools()
    import pytest
    with pytest.raises(ValueError, match="허용되지 않은 도메인"):
        tools.fetch_realtime("http://localhost:19999/nonexistent_raon_test")


# ─── Test 16: AgenticRAG.run() — search 전략 통합 ────────────────────────────

def test_agentic_rag_run_search_strategy():
    """AgenticRAG.run(): search 전략으로 정상 실행."""
    mock_rag = MockRAG()
    agentic = AgenticRAG(mock_rag)

    responses = [
        "search",  # router
        "TIPS 관련 쿼리1\nTIPS 관련 쿼리2\nTIPS 관련 쿼리3",  # multi-query variants
        '{"relevant": 0.8, "sufficient": 0.8, "confident": 0.8, "action": "use"}',  # CRAG
        "TIPS 프로그램에 대한 답변입니다.",  # final answer
    ]

    with patch("agentic_rag.chat", side_effect=responses):
        result = agentic.run("TIPS 추천해줘")

    assert "answer" in result
    assert "strategy_used" in result
    assert "sources" in result
    assert "confidence" in result
    assert "iterations" in result
    assert isinstance(result["answer"], str)
    assert len(result["answer"]) > 0
    assert result["strategy_used"] == "search"


# ─── Test 17: AgenticRAG.run() — factual 전략 ────────────────────────────────

def test_agentic_rag_run_factual_strategy():
    """AgenticRAG.run(): factual → HyDE 전략 실행."""
    mock_rag = MockRAG()
    agentic = AgenticRAG(mock_rag)

    def chat_side_effect(messages: list) -> str:
        content = messages[0]["content"] if messages else ""
        if "유형을 분류" in content:
            return "factual"
        if "가상의 정부 지원사업" in content or "가상 공고문" in content:
            return "가상 문서: TIPS 자격조건은 창업 7년 이내..."
        if "relevant" in content or "평가" in content:
            return '{"relevant": 0.9, "sufficient": 0.9, "confident": 0.9, "action": "use"}'
        if "검색 결과를 바탕으로" in content:
            return "TIPS 자격조건은 창업 7년 이내입니다."
        return "factual"

    with patch("agentic_rag.chat", side_effect=chat_side_effect):
        result = agentic.run("TIPS 자격조건은?")

    assert result["strategy_used"] == "factual"
    assert "answer" in result
    assert isinstance(result["answer"], str)


# ─── Test 18: AgenticRAG.run() — 검색 결과 없음 폴백 ────────────────────────

def test_agentic_rag_run_empty_results():
    """AgenticRAG.run(): 검색 결과 없을 때 confidence=0.0, answer 반환."""
    empty_rag = MockRAG(docs=[])
    agentic = AgenticRAG(empty_rag)

    with patch("agentic_rag.chat", return_value="search"):
        result = agentic.run("존재하지 않는 프로그램 XYZ", max_retries=0)

    assert "answer" in result
    assert isinstance(result["confidence"], float)
    # 문서 없으면 confidence 0.0
    assert result["confidence"] == 0.0


# ─── Test 19: structured_extractor — rule-based 추출 ─────────────────────────

def test_structured_extractor_rule_based():
    """_rule_based_extract: 텍스트에서 기본 정보 추출."""
    sample_text = """
    사업명: TIPS (민간투자주도형 기술창업지원)
    주관기관: 중소벤처기업부
    마감일: 2026.03.31
    URL: https://www.tips.or.kr/apply
    예산: 300억 원
    자격조건: 창업 7년 이내 기업
    """
    result = _rule_based_extract(sample_text, _default_schema())

    assert result["program_name"] == "TIPS (민간투자주도형 기술창업지원)"
    assert result["deadline"] == "2026-03-31"
    assert "https://www.tips.or.kr/apply" in result["application_url"]
    assert result["budget_won"] == 30_000_000_000  # 300억 = 30,000,000,000원


# ─── Test 20: structured_extractor — filter_programs ─────────────────────────

def test_filter_programs_basic():
    """filter_programs: 하드 필터 + 소프트 랭킹 검증."""
    programs = [
        {
            "program_name": "AI 창업 지원",
            "deadline": "2026-06-30",
            "excluded": ["폐업기업"],
            "industry_focus": ["AI", "소프트웨어"],
            "keywords": ["딥테크", "AI"],
            "budget_won": 500_000_000,
        },
        {
            "program_name": "제조업 지원",
            "deadline": "2025-12-31",  # 마감 지남
            "excluded": [],
            "industry_focus": ["제조"],
            "keywords": ["하드웨어"],
            "budget_won": 200_000_000,
        },
        {
            "program_name": "바이오 지원",
            "deadline": "2026-07-15",
            "excluded": ["폐업기업", "휴면법인"],
            "industry_focus": ["바이오", "헬스케어"],
            "keywords": ["신약"],
            "budget_won": 1_000_000_000,
        },
    ]

    criteria = {
        "deadline_after": "2026-01-01",
        "industry": "AI",
        "keywords": ["딥테크"],
    }

    result = filter_programs(programs, criteria)

    # 마감일 지난 제조업 지원 제외
    names = [p.get("program_name", "") for p in result]
    assert "제조업 지원" not in names, "마감 지난 프로그램이 포함됨"
    # AI 관련 프로그램이 상위
    assert names[0] == "AI 창업 지원", f"AI 프로그램이 1위여야 하는데: {names}"


# ─── 메인 ─────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
