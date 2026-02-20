"""idea.py 단위 테스트 — YC RFS & a16z 기반 창업 아이디어 추천"""
import json
import io
import os
import sys
import pytest

sys.path.insert(0, os.path.dirname(__file__))
from idea import (
    load_rfs_categories,
    list_categories,
    get_category_detail,
    suggest_ideas,
    format_list,
    format_detail,
    format_suggest,
)


class TestLoadCategories:
    def test_loads_9_categories(self):
        """기존 호환성: 최소 9개 카테고리 로드 (이제 50개+)"""
        cats = load_rfs_categories()
        assert len(cats) >= 9

    def test_category_has_required_fields(self):
        cats = load_rfs_categories()
        for cat in cats:
            assert "id" in cat
            assert "name" in cat
            assert "subtitle" in cat
            assert isinstance(cat["id"], int)
            assert cat["id"] >= 1

    def test_loads_50_plus_categories(self):
        cats = load_rfs_categories()
        assert len(cats) >= 50

    def test_categories_have_source(self):
        cats = load_rfs_categories()
        for cat in cats:
            assert "source" in cat
            assert "source_org" in cat
            assert cat["source"] != ""

    def test_categories_have_keywords(self):
        cats = load_rfs_categories()
        for cat in cats:
            assert "keywords" in cat
            assert isinstance(cat["keywords"], list)
            assert len(cat["keywords"]) > 0


class TestGetDetail:
    def test_valid_id_returns_category(self):
        cat = get_category_detail(1)
        assert cat is not None
        assert cat["id"] == 1
        assert "Cursor" in cat["name"]

    def test_invalid_id_returns_none(self):
        assert get_category_detail(0) is None
        assert get_category_detail(999) is None

    def test_detail_has_source(self):
        cat = get_category_detail(16)
        assert cat is not None
        assert "a16z" in cat["source"]


class TestSuggestIdeas:
    def test_returns_top3_matches(self):
        result = suggest_ideas("ML 엔지니어", "LLM 파인튜닝 인프라")
        assert "matches" in result
        assert len(result["matches"]) == 3

    def test_ml_background_matches_llm_training(self):
        result = suggest_ideas("ML 엔지니어, MLOps", "LLM train 파인튜닝")
        ids = [m["id"] for m in result["matches"]]
        # Category 9 (Make LLMs Easy to Train) should be in top matches
        assert 9 in ids[:3]

    def test_finance_background_matches_hedge_or_stablecoin(self):
        result = suggest_ideas("퀀트 트레이더", "금융 트레이딩 알고리즘")
        ids = [m["id"] for m in result["matches"]]
        assert 2 in ids[:2]  # AI-Native Hedge Funds

    def test_empty_input_returns_results(self):
        result = suggest_ideas("", "")
        assert "matches" in result
        assert len(result["matches"]) == 3


class TestFiltering:
    def test_filter_by_source_yc(self):
        cats = list_categories(source="yc")
        assert len(cats) > 0
        for cat in cats:
            assert cat["source_org"] == "yc"

    def test_filter_by_source_a16z(self):
        cats = list_categories(source="a16z")
        assert len(cats) > 0
        for cat in cats:
            assert cat["source_org"] == "a16z"

    def test_filter_by_season(self):
        cats = list_categories(season="Spring 2026")
        assert len(cats) > 0
        for cat in cats:
            assert "spring 2026" in cat["season"].lower()

    def test_filter_by_season_fall_2025(self):
        cats = list_categories(season="Fall 2025")
        assert len(cats) > 0

    def test_filter_combined(self):
        cats = list_categories(source="yc", season="Spring 2026")
        assert len(cats) > 0
        for cat in cats:
            assert cat["source_org"] == "yc"

    def test_no_results_filter(self):
        cats = list_categories(source="nonexistent")
        assert len(cats) == 0


class TestFormatting:
    def test_format_list_contains_all(self):
        cats = list_categories()
        text = format_list(cats)
        assert "아이디어" in text
        assert "1." in text

    def test_format_detail(self):
        cat = get_category_detail(5)
        text = format_detail(cat)
        assert "AI for Government" in text
        assert "한국 시장" in text

    def test_format_suggest(self):
        result = suggest_ideas("개발자", "AI")
        text = format_suggest(result)
        assert "라온" in text
        assert "🥇" in text

    def test_format_list_shows_source(self):
        cats = list_categories()
        text = format_list(cats)
        assert "YC" in text or "a16z" in text


class TestServerEndpoints:
    """server.py의 /v1/idea 엔드포인트 테스트 (핸들러 레벨)"""

    def _make_handler(self):
        from server import RaonHandler

        class FakeHandler(RaonHandler):
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

        return FakeHandler()

    def test_idea_list_endpoint(self):
        h = self._make_handler()
        h.path = "/v1/idea/list"
        h.do_GET()
        resp = h._get_response()
        assert resp["status"] == "ok"
        assert len(resp["categories"]) >= 9

    def test_idea_detail_endpoint(self):
        h = self._make_handler()
        h.path = "/v1/idea/detail/3"
        h.do_GET()
        resp = h._get_response()
        assert resp["status"] == "ok"
        assert resp["category"]["id"] == 3
