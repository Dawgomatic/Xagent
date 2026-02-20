"""Tests for rerank function in rag_pipeline."""

import json
import unittest
from unittest.mock import patch, MagicMock
from rag_pipeline import rerank


def _make_candidates(n: int) -> list[dict]:
    """Generate n fake candidate documents."""
    return [
        {"text": f"문서 {i} 내용입니다.", "meta": {"type": "test"}, "score": 0.9 - i * 0.05}
        for i in range(n)
    ]


class TestRerank(unittest.TestCase):

    def test_empty_candidates(self):
        assert rerank("query", [], top_k=3) == []

    def test_fewer_than_top_k(self):
        cands = _make_candidates(2)
        result = rerank("query", cands, top_k=3)
        assert len(result) == 2

    @patch("rag_pipeline.urllib.request.urlopen")
    def test_rerank_parses_llm_response(self, mock_urlopen):
        """LLM이 '2,0,1' 반환 시 해당 순서로 재정렬."""
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"response": "2,0,1"}).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        cands = _make_candidates(5)
        result = rerank("test query", cands, top_k=3)

        assert len(result) == 3
        assert result[0]["text"] == "문서 2 내용입니다."
        assert result[1]["text"] == "문서 0 내용입니다."
        assert result[2]["text"] == "문서 1 내용입니다."

    @patch("rag_pipeline.urllib.request.urlopen")
    def test_rerank_handles_llm_failure(self, mock_urlopen):
        """LLM 호출 실패 시 원본 순서 Top-k 반환."""
        mock_urlopen.side_effect = Exception("connection refused")

        cands = _make_candidates(5)
        result = rerank("test query", cands, top_k=3)

        assert len(result) == 3
        # 원본 순서 유지
        assert result[0]["text"] == "문서 0 내용입니다."

    @patch("rag_pipeline.urllib.request.urlopen")
    def test_rerank_partial_response(self, mock_urlopen):
        """LLM이 부분적 응답(1개만) 줘도 나머지를 원본에서 채움."""
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"response": "4"}).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        cands = _make_candidates(5)
        result = rerank("test query", cands, top_k=3)

        assert len(result) == 3
        assert result[0]["text"] == "문서 4 내용입니다."


if __name__ == "__main__":
    unittest.main()
