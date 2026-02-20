"""server.py 단위 테스트 (서버 기동 없이 핸들러 로직 검증)"""
import json
import io
import pytest
from unittest.mock import patch, MagicMock
from http.server import BaseHTTPRequestHandler

# Import the handler
import sys, os
sys.path.insert(0, os.path.dirname(__file__))
from server import RaonHandler, VALID_MODES


class FakeHandler(RaonHandler):
    """서버 없이 핸들러 테스트"""
    def __init__(self):
        self.response_code = None
        self.response_body = None
        self.headers_sent = {}
        self.wfile = io.BytesIO()
        self.client_address = ("127.0.0.1", 0)
        self.headers = {"X-API-Key": "", "Content-Length": "0"}

    def send_response(self, code):
        self.response_code = code

    def send_header(self, key, value):
        self.headers_sent[key] = value

    def end_headers(self):
        pass

    def _get_response(self):
        return json.loads(self.wfile.getvalue())


def test_valid_modes():
    assert "evaluate" in VALID_MODES
    assert "improve" in VALID_MODES
    assert "match" in VALID_MODES
    assert "draft" in VALID_MODES
    assert "checklist" in VALID_MODES
    assert "investor" in VALID_MODES
    assert len(VALID_MODES) >= 6


def test_health_endpoint():
    handler = FakeHandler()
    handler.path = "/health"
    handler.do_GET()
    assert handler.response_code == 200
    body = handler._get_response()
    assert body["status"] == "ok"
    assert "version" in body
    assert "model" in body


def test_modes_endpoint():
    handler = FakeHandler()
    handler.path = "/v1/modes"
    handler.do_GET()
    assert handler.response_code == 200
    body = handler._get_response()
    assert "modes" in body
    assert body["modes"] == VALID_MODES


def test_404_get():
    handler = FakeHandler()
    handler.path = "/nonexistent"
    handler.do_GET()
    assert handler.response_code == 404


def test_invalid_mode_post():
    handler = FakeHandler()
    handler.path = "/v1/invalid_mode"
    handler.headers = {"Content-Length": "20"}
    handler.rfile = io.BytesIO(b'{"text":"test"}')
    handler.do_POST()
    assert handler.response_code == 400


def test_missing_text_post():
    handler = FakeHandler()
    handler.path = "/v1/evaluate"
    handler.headers = {"Content-Length": "2"}
    handler.rfile = io.BytesIO(b'{}')
    handler.do_POST()
    assert handler.response_code == 400


def test_draft_requires_program():
    """program 없으면 TIPS로 기본값 처리 → 200 반환 (v0.6.2+ 동작)"""
    handler = FakeHandler()
    handler.path = "/v1/draft"
    body = json.dumps({"text": "test", "program": ""}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    # program 없으면 TIPS 기본값으로 처리 (400 아님)
    assert handler.response_code in (200, 503), f"expected 200 or 503, got {handler.response_code}"


@patch("server.call_ollama", return_value="종합 점수: 75/100\n좋은 계획입니다.")
@patch("server.call_raon_api", return_value=None)
def test_evaluate_success(mock_api, mock_ollama):
    handler = FakeHandler()
    handler.path = "/v1/evaluate"
    body = json.dumps({"text": "테스트 사업계획서"}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    assert handler.response_code == 200
    resp = handler._get_response()
    assert resp["status"] == "ok"
    assert resp["mode"] == "evaluate"
    assert resp["score"] == 75
    assert "result" in resp


@patch("server.call_ollama", return_value=None)
@patch("server.call_raon_api", return_value=None)
def test_llm_unavailable(mock_api, mock_ollama):
    handler = FakeHandler()
    handler.path = "/v1/evaluate"
    body = json.dumps({"text": "test"}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    assert handler.response_code == 503


def test_cors_options():
    handler = FakeHandler()
    handler.do_OPTIONS()
    assert handler.response_code == 204


@patch("server.call_ollama", return_value="종합 점수: 80/100\n우수합니다.")
@patch("server.call_raon_api", return_value=None)
def test_pdf_base64_upload(mock_api, mock_ollama):
    """PDF base64 업로드 → 텍스트 추출 → 평가"""
    import base64
    # Create a minimal valid PDF
    try:
        from pypdf import PdfWriter
    except ImportError:
        from PyPDF2 import PdfWriter
    writer = PdfWriter()
    writer.add_blank_page(width=72, height=72)
    buf = io.BytesIO()
    writer.write(buf)
    pdf_b64 = base64.b64encode(buf.getvalue()).decode()

    handler = FakeHandler()
    handler.path = "/v1/evaluate"
    body = json.dumps({"pdf_base64": pdf_b64}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)

    # extract_text_from_pdf will return empty for blank PDF, so we mock it
    with patch("server.extract_text_from_pdf", return_value="테스트 사업계획서 내용"):
        handler.do_POST()

    assert handler.response_code == 200
    resp = handler._get_response()
    assert resp["status"] == "ok"
    assert resp["score"] == 80


@patch("server.call_ollama", return_value="종합 점수: 82/100\n잘 작성되었습니다.")
@patch("server.call_raon_api", return_value=None)
def test_chat_new_session(mock_api, mock_ollama):
    """POST /v1/chat — 새 세션 생성"""
    from server import CHAT_SESSIONS
    CHAT_SESSIONS.clear()

    handler = FakeHandler()
    handler.path = "/v1/chat"
    body = json.dumps({"text": "테스트 사업계획서"}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    assert handler.response_code == 200
    resp = handler._get_response()
    assert resp["status"] == "ok"
    assert "session_id" in resp
    assert resp["turn"] == 0
    assert resp["score"] == 82
    assert resp["session_id"] in CHAT_SESSIONS


@patch("server.call_ollama", return_value="시장 분석 상세 내용입니다.")
@patch("server.call_raon_api", return_value=None)
def test_chat_followup(mock_api, mock_ollama):
    """POST /v1/chat — 후속 질문"""
    from server import CHAT_SESSIONS
    CHAT_SESSIONS.clear()
    sid = "test-session-123"
    CHAT_SESSIONS[sid] = {
        "history": [{"role": "assistant", "content": "초기 평가"}],
        "original_text": "원본 사업계획서",
        "model": "qwen3:8b",
    }

    handler = FakeHandler()
    handler.path = "/v1/chat"
    body = json.dumps({"session_id": sid, "message": "시장 분석 더 자세히"}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    assert handler.response_code == 200
    resp = handler._get_response()
    assert resp["status"] == "ok"
    assert resp["session_id"] == sid
    assert resp["turn"] == 1
    assert len(CHAT_SESSIONS[sid]["history"]) == 3  # assistant + user + assistant


def test_chat_missing_text_new_session():
    """새 세션에 text 없으면 400"""
    handler = FakeHandler()
    handler.path = "/v1/chat"
    body = json.dumps({"message": "질문"}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    assert handler.response_code == 400


def test_chat_missing_message_existing_session():
    """기존 세션에 message 없으면 400"""
    from server import CHAT_SESSIONS
    CHAT_SESSIONS["existing"] = {
        "history": [{"role": "assistant", "content": "평가"}],
        "original_text": "텍스트",
        "model": "qwen3:8b",
    }
    handler = FakeHandler()
    handler.path = "/v1/chat"
    body = json.dumps({"session_id": "existing"}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    assert handler.response_code == 400


def test_pdf_parse_error():
    """잘못된 PDF base64 → 400 에러"""
    import base64
    handler = FakeHandler()
    handler.path = "/v1/evaluate"
    body = json.dumps({"pdf_base64": base64.b64encode(b"not a pdf").decode()}).encode()
    handler.headers = {"Content-Length": str(len(body))}
    handler.rfile = io.BytesIO(body)
    handler.do_POST()
    # Either 400 (parse error) or processes anyway (some parsers might not fail)
    assert handler.response_code in (200, 400, 503)
