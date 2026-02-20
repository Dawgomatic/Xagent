"""kiwipiepy 형태소 분석 BM25 토크나이저 테스트."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent))


def test_kiwi_available():
    """kiwipiepy is optional — skip if not installed (bigram fallback used)"""
    import pytest
    from rag_pipeline import _get_kiwi
    kiwi = _get_kiwi()
    if kiwi is None:
        pytest.skip("kiwipiepy not installed — bigram fallback active (OK)")
    assert kiwi is not None


def test_tokenize_kiwi_korean():
    from rag_pipeline import _tokenize_kiwi
    tokens = _tokenize_kiwi("TIPS 합격하려면 어떻게 준비해야 해?")
    assert len(tokens) > 0
    # Should contain meaningful morphemes, not bigrams
    assert "합격" in tokens or "준비" in tokens


def test_tokenize_kiwi_english():
    from rag_pipeline import _tokenize_kiwi
    tokens = _tokenize_kiwi("startup funding strategy")
    assert "startup" in tokens or "funding" in tokens or "strategy" in tokens


def test_tokenize_fallback():
    from rag_pipeline import _tokenize_bigram
    tokens = _tokenize_bigram("정부지원사업 합격 전략")
    assert len(tokens) > 0
    # bigram should produce character pairs
    assert any(len(t) == 2 for t in tokens)


def test_tokenize_dispatches_to_kiwi():
    from rag_pipeline import _tokenize, _get_kiwi
    if _get_kiwi() is None:
        return  # skip if kiwi unavailable
    tokens = _tokenize("예비창업패키지 사업계획서 작성법")
    # kiwi should extract meaningful nouns
    assert any(t in tokens for t in ["예비", "창업", "패키지", "사업", "계획서", "작성"])


def test_bm25_score_with_kiwi():
    from rag_pipeline import _build_bm25_index, bm25_score
    store = [
        {"text": "TIPS 합격하려면 기술력과 팀 역량이 중요합니다"},
        {"text": "예비창업패키지는 예비창업자를 위한 정부 지원사업입니다"},
        {"text": "벤처투자 현황 2025년 상반기 실적"},
    ]
    index = _build_bm25_index(store)
    s0 = bm25_score("TIPS 합격 방법", 0, index)
    s1 = bm25_score("TIPS 합격 방법", 1, index)
    # TIPS 관련 문서가 더 높은 점수
    assert s0 > s1, f"TIPS doc should score higher: {s0} vs {s1}"
