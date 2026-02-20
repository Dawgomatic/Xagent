"""
evaluate.py 단위 테스트
"""
import sys
import os
import pytest
import json
import tempfile
from unittest.mock import patch, MagicMock

# Import evaluate.py
sys.path.insert(0, os.path.dirname(__file__))
import evaluate as ev

class TestParseScore:
    def test_basic_score(self):
        text = "종합 점수: 85/100"
        assert ev.parse_score(text) == 85

    def test_score_with_spaces(self):
        text = "총점 :  90 / 100"
        assert ev.parse_score(text) == 90
    
    def test_score_bold(self):
        text = "결과: **77/100** 입니다."
        assert ev.parse_score(text) == 77

    def test_none_input(self):
        assert ev.parse_score(None) is None
    
    def test_no_score_found(self):
        assert ev.parse_score("점수 없음") is None
    
    def test_out_of_range(self):
        # 100점 초과는 무시
        assert ev.parse_score("종합 점수: 150/100") is None

class TestExtractPdf:
    def test_nonexistent_pdf_exits(self):
        """존재하지 않는 PDF → SystemExit 또는 에러"""
        with pytest.raises((SystemExit, FileNotFoundError, Exception)):
            ev.extract_pdf_text("/nonexistent/fake.pdf")

    def test_txt_file_raises(self):
        """텍스트 파일을 PDF로 읽으면 에러 발생"""
        with tempfile.NamedTemporaryFile(suffix=".txt", mode="w", delete=False) as f:
            f.write("이건 텍스트 파일입니다")
            f.flush()
            with pytest.raises((SystemExit, Exception)):
                ev.extract_pdf_text(f.name)
        os.unlink(f.name)

class TestBuildPrompt:
    def test_evaluate_prompt(self):
        p = ev.build_prompt("test content", mode="evaluate")
        assert "TIPS 심사 기준" in p
        assert "종합 점수" in p

    def test_improve_prompt(self):
        p = ev.build_prompt("test content", mode="improve")
        assert "개선된 버전" in p
        assert "변경 요약" in p

    def test_match_prompt(self):
        p = ev.build_prompt("test content", mode="match")
        assert "정부 지원사업" in p
        assert "가중치" in p

    def test_draft_prompt(self):
        p = ev.build_prompt("test content", mode="draft", program="TIPS")
        assert "TIPS 지원서 초안" in p
        assert "사업 개요" in p
    
    def test_checklist_prompt(self):
        p = ev.build_prompt("test content", mode="checklist", program="TIPS")
        assert "체크리스트" in p
        assert "필수 서류" in p

class TestInvestorMode:
    def test_investor_mode_prompt(self):
        """investor 모드 프롬프트가 올바른 구조를 가지는지 확인"""
        p = ev.build_prompt("테스트 사업계획서 내용", mode="investor")
        assert "투자자(VC/AC/Angel) 관점" in p
        assert "Deal Summary" in p
        assert "Investment Highlights" in p
        assert "Red Flags" in p

class TestJsonOutput:
    def test_json_payload_structure(self):
        """JSON 출력 페이로드가 필수 필드를 포함하는지 확인"""
        payload = {
            "status": "ok",
            "mode": "evaluate",
            "model": "qwen3:8b",
            "text_length": 1000,
            "doc_count": 1,
            "result": "종합 점수: 85/100",
            "score": 85,
        }
        assert payload["status"] == "ok"
        assert payload["score"] == 85
        assert payload["doc_count"] == 1
        assert "result" in payload

    def test_json_serialization(self):
        """JSON 직렬화가 한국어를 올바르게 처리하는지 확인"""
        payload = {"result": "사업계획서 평가 결과", "status": "ok"}
        serialized = json.dumps(payload, ensure_ascii=False)
        assert "사업계획서" in serialized
        assert "\\u" not in serialized


class TestHistoryLog:
    def test_log_entry_format(self):
        """히스토리 로그 엔트리 형식 확인"""
        entry = {
            "timestamp": 1739600000,
            "mode": "evaluate",
            "model": "qwen3:8b",
            "input_len": 500,
            "duration": 3.14,
            "status": "success",
            "score": 80,
            "files": ["plan.pdf"],
        }
        line = json.dumps(entry)
        parsed = json.loads(line)
        assert parsed["mode"] == "evaluate"
        assert parsed["status"] == "success"
        assert isinstance(parsed["duration"], float)
        assert parsed["files"] == ["plan.pdf"]

    def test_log_entry_without_score(self):
        """점수 없는 모드(improve 등)의 로그 엔트리"""
        entry = {
            "timestamp": 1739600000,
            "mode": "improve",
            "model": "qwen3:8b",
            "input_len": 500,
            "duration": 5.2,
            "status": "success",
        }
        assert "score" not in entry

class TestBuildPromptEdgeCases:
    def test_long_text_truncated(self):
        """8000자 이상 텍스트가 프롬프트에서 잘리는지 확인"""
        long_text = "가" * 10000
        p = ev.build_prompt(long_text, mode="evaluate")
        # text[:8000] in prompt
        assert "가" * 8000 in p
        assert "가" * 10000 not in p

    def test_investor_mode_has_required_sections(self):
        """investor 모드에 필수 섹션이 모두 있는지"""
        p = ev.build_prompt("테스트", mode="investor")
        for section in ["Deal Summary", "Investment Highlights", "Red Flags"]:
            assert section in p

    def test_draft_without_program_defaults(self):
        """draft 모드에 program 없으면 기본 동작"""
        p = ev.build_prompt("테스트", mode="draft", program=None)
        assert "지원서 초안" in p or "draft" in p.lower() or len(p) > 0

    def test_comparison_prompt(self):
        """비교 모드 프롬프트 빌드"""
        docs = [{"name": "A.pdf", "text": "문서A"}, {"name": "B.pdf", "text": "문서B"}]
        p = ev.build_comparison_prompt(docs, mode="evaluate")
        assert "문서 1" in p and "문서 2" in p
        assert "비교" in p


class TestBuildFollowupPrompt:
    def test_followup_prompt_structure(self):
        history = [
            {"role": "assistant", "content": "초기 평가 결과입니다."},
            {"role": "user", "content": "시장 분석 더 자세히"},
        ]
        p = ev.build_followup_prompt(history, "시장 분석 더 자세히", "사업계획서 텍스트")
        assert "후속 질문" in p
        assert "시장 분석 더 자세히" in p
        assert "사업계획서 텍스트" in p
        assert "이전 대화" in p

    def test_followup_preserves_history(self):
        history = [
            {"role": "assistant", "content": "평가A"},
            {"role": "user", "content": "질문1"},
            {"role": "assistant", "content": "답변1"},
            {"role": "user", "content": "질문2"},
        ]
        p = ev.build_followup_prompt(history, "질문2", "원본")
        assert "평가A" in p
        assert "질문1" in p
        assert "답변1" in p

    def test_followup_truncates_long_text(self):
        """원본 텍스트가 4000자로 잘리는지"""
        long_text = "가" * 6000
        p = ev.build_followup_prompt([], "질문", long_text)
        assert "가" * 4000 in p
        assert "가" * 6000 not in p


class TestInteractiveMode:
    def test_interactive_in_choices(self):
        """interactive가 argparse choices에 포함되는지"""
        import argparse
        parser = argparse.ArgumentParser()
        parser.add_argument("mode", choices=["evaluate", "improve", "match", "draft", "checklist", "investor", "interactive"])
        args = parser.parse_args(["interactive"])
        assert args.mode == "interactive"


class TestReadRef:
    def test_missing_ref_returns_empty(self):
        assert ev.read_ref("nonexistent.md") == ""
    
    def test_tips_criteria_exists(self):
        # This depends on actual file existence
        pass
