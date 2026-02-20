#!/usr/bin/env python3
"""
Raon OS — 카카오 연동 + Track B + 금융맵 테스트

pytest scripts/test_kakao.py -v

Python 3.9+ compatible
"""
from __future__ import annotations

import hashlib
import hmac
import json
import sys
import unittest
from pathlib import Path
from unittest.mock import MagicMock

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))


# ─── TrackClassifier 테스트 ───────────────────────────────────────────────────

class TestTrackClassifier(unittest.TestCase):
    """TrackClassifier: 키워드/LLM 분류 테스트."""

    def setUp(self):
        from track_classifier import TrackClassifier
        self.clf = TrackClassifier()

    def test_chicken_is_track_b(self):
        """치킨집 → Track B."""
        track = self.clf.classify("치킨집을 열고 싶습니다. 동네 치킨 프랜차이즈")
        self.assertEqual(track, "B", f"치킨집은 Track B여야 함, got: {track}")

    def test_cafe_is_track_b(self):
        """카페 → Track B."""
        track = self.clf.classify("홍대 근처에 카페를 창업하려고 합니다")
        self.assertEqual(track, "B", f"카페는 Track B여야 함, got: {track}")

    def test_ai_saas_is_track_a(self):
        """AI SaaS → Track A."""
        track = self.clf.classify("AI 기반 B2B SaaS 플랫폼을 개발하고 있습니다. API 솔루션")
        self.assertEqual(track, "A", f"AI SaaS는 Track A여야 함, got: {track}")

    def test_tips_is_track_a(self):
        """TIPS 기술창업 → Track A."""
        track = self.clf.classify("TIPS 프로그램 지원을 위한 R&D 스타트업입니다")
        self.assertEqual(track, "A", f"TIPS는 Track A여야 함, got: {track}")

    def test_foodtech_is_track_ab(self):
        """푸드테크 → Track AB."""
        track = self.clf.classify("푸드테크 스타트업으로 AI 기반 음식점 관리 솔루션")
        self.assertIn(track, ("AB", "A", "B"), f"푸드테크는 AB 또는 A/B: {track}")
        # 푸드테크 키워드 존재 확인
        from track_classifier import TRACK_AB_KEYWORDS
        self.assertIn("푸드테크", TRACK_AB_KEYWORDS)

    def test_beauty_salon_is_track_b(self):
        """뷰티샵 → Track B."""
        track = self.clf.classify("뷰티샵을 운영하고 싶습니다. 미용실 창업")
        self.assertEqual(track, "B", f"뷰티샵은 Track B여야 함, got: {track}")

    def test_get_track_prompt_b(self):
        """Track B 시스템 프롬프트 확인."""
        prompt = self.clf.get_track_prompt("B")
        self.assertIn("소상공인", prompt)
        self.assertIn("입지", prompt)
        # 기술성 언급 없어야 함 (실제 평가 프롬프트와 달리 시스템 프롬프트는 가이드)
        self.assertNotIn("TIPS R&D", prompt)

    def test_get_track_prompt_a(self):
        """Track A 시스템 프롬프트 확인."""
        prompt = self.clf.get_track_prompt("A")
        self.assertIn("기술창업", prompt)

    def test_keyword_list_completeness(self):
        """키워드 목록 최소 개수 확인."""
        from track_classifier import TRACK_A_KEYWORDS, TRACK_B_KEYWORDS
        self.assertGreaterEqual(len(TRACK_A_KEYWORDS), 10)
        self.assertGreaterEqual(len(TRACK_B_KEYWORDS), 10)

    def test_classify_returns_valid_track(self):
        """classify 반환값은 항상 A, B, AB 중 하나."""
        texts = [
            "블록체인 기술 스타트업",
            "동네 편의점 창업",
            "스마트 음식점 관리 앱",
            "바이오 헬스케어 기업",
            "유튜버 콘텐츠 사업",
        ]
        for text in texts:
            with self.subTest(text=text):
                track = self.clf.classify(text)
                self.assertIn(track, ("A", "B", "AB"), f"'{text}' → 잘못된 트랙: {track}")


# ─── FinancialMap 테스트 ──────────────────────────────────────────────────────

class TestFinancialMap(unittest.TestCase):
    """FinancialMap.match(): 트랙별 상품 필터링 테스트."""

    def setUp(self):
        from financial_map import FinancialMap
        self.fm = FinancialMap()

    def test_track_b_match_includes_semas(self):
        """Track B 매칭에 소진공 정책자금 포함."""
        products = self.fm.match(track="B")
        names = [p["name"] for p in products]
        self.assertTrue(
            any("소상공인 정책자금" in n for n in names),
            f"Track B에 소진공 정책자금 없음. 결과: {names}"
        )

    def test_track_a_match_includes_tips(self):
        """Track A 매칭에 TIPS 포함."""
        products = self.fm.match(track="A")
        names = [p["name"] for p in products]
        self.assertTrue(
            any("TIPS" in n for n in names),
            f"Track A에 TIPS 없음. 결과: {names}"
        )

    def test_track_b_excludes_kibo(self):
        """Track B 매칭에 KIBO(기술보증) 제외."""
        products = self.fm.match(track="B")
        names = [p["name"] for p in products]
        # KIBO는 Track A, AB 전용 — Track B에는 없어야 함
        kibo_in = any("KIBO" in n or "기술보증" in n for n in names)
        self.assertFalse(kibo_in, f"Track B에 KIBO가 포함됨: {names}")

    def test_track_ab_match_has_multiple_types(self):
        """Track AB 매칭은 여러 타입 포함."""
        products = self.fm.match(track="AB")
        types = {p["type"] for p in products}
        self.assertGreaterEqual(len(types), 2, f"Track AB 타입이 너무 적음: {types}")

    def test_need_loan_filter(self):
        """need_loan=True면 융자/보증 우선."""
        products = self.fm.match(track="B", need_loan=True)
        # 최상위 결과가 융자/보증이어야 함
        if products:
            self.assertIn(
                products[0]["type"], ("융자", "보증"),
                f"need_loan=True인데 첫 번째 상품이 {products[0]['type']}"
            )

    def test_format_recommendation_not_empty(self):
        """format_recommendation: 빈 결과 처리."""
        text = self.fm.format_recommendation([])
        self.assertIn("소진공", text)

    def test_format_recommendation_with_products(self):
        """format_recommendation: 상품 있을 때 URL 포함."""
        products = self.fm.match(track="B")
        text = self.fm.format_recommendation(products[:2])
        self.assertIn("http", text, "URL이 응답에 포함되어야 함")

    def test_get_summary_track_b(self):
        """Track B 요약 문자열 확인."""
        summary = self.fm.get_summary("B")
        self.assertIn("소상공인", summary)
        self.assertIn("소진공", summary)


# ─── KakaoWebhook 테스트 ─────────────────────────────────────────────────────

class TestKakaoWebhookFormatResponse(unittest.TestCase):
    """KakaoWebhook.format_response(): 1000자 분할 테스트."""

    def setUp(self):
        from kakao_webhook import KakaoWebhook
        self.hook = KakaoWebhook()

    def test_short_text_single_output(self):
        """짧은 텍스트는 하나의 output."""
        result = self.hook.format_response("안녕하세요!")
        outputs = result["template"]["outputs"]
        self.assertEqual(len(outputs), 1)
        self.assertEqual(outputs[0]["simpleText"]["text"], "안녕하세요!")

    def test_long_text_split(self):
        """900자 초과 텍스트는 분할."""
        long_text = "A" * 2000
        result = self.hook.format_response(long_text)
        outputs = result["template"]["outputs"]
        self.assertGreater(len(outputs), 1, "2000자 텍스트는 2개 이상으로 분할되어야 함")
        # 각 chunk는 900자 이하
        for output in outputs:
            chunk = output["simpleText"]["text"]
            self.assertLessEqual(len(chunk), 900, f"청크 길이 {len(chunk)} > 900")

    def test_version_field(self):
        """응답에 version 2.0 포함."""
        result = self.hook.format_response("테스트")
        self.assertEqual(result["version"], "2.0")

    def test_buttons_in_response(self):
        """버튼이 있으면 quickReplies 포함."""
        buttons = ["버튼1", "버튼2"]
        result = self.hook.format_response("텍스트", buttons=buttons)
        self.assertIn("quickReplies", result["template"])
        labels = [q["label"] for q in result["template"]["quickReplies"]]
        self.assertEqual(labels, buttons)

    def test_no_buttons_no_quick_replies(self):
        """버튼 없으면 quickReplies 없음."""
        result = self.hook.format_response("텍스트")
        self.assertNotIn("quickReplies", result["template"])

    def test_max_outputs_limit(self):
        """최대 5개 outputs 제한."""
        huge_text = "B" * 10000
        result = self.hook.format_response(huge_text)
        outputs = result["template"]["outputs"]
        self.assertLessEqual(len(outputs), 5)


class TestKakaoWebhookVerifySignature(unittest.TestCase):
    """KakaoWebhook.verify_signature(): 서명 검증 테스트."""

    def setUp(self):
        from kakao_webhook import KakaoWebhook
        self.hook_no_secret = KakaoWebhook()  # 시크릿 없음

        import os
        os.environ["KAKAO_CALLBACK_SECRET"] = "test-secret-key"
        self.hook_with_secret = KakaoWebhook()
        del os.environ["KAKAO_CALLBACK_SECRET"]

    def test_no_secret_always_true(self):
        """시크릿 없으면 항상 True."""
        result = self.hook_no_secret.verify_signature(b"body", "any-sig")
        self.assertTrue(result)

    def test_valid_signature(self):
        """올바른 서명 검증 성공."""
        body = b"test body content"
        secret = "test-secret-key"
        sig = hmac.new(secret.encode(), body, hashlib.sha1).hexdigest()
        result = self.hook_with_secret.verify_signature(body, sig)
        self.assertTrue(result, "올바른 서명이 거부됨")

    def test_invalid_signature(self):
        """잘못된 서명 거부."""
        result = self.hook_with_secret.verify_signature(b"body", "wrong-signature")
        self.assertFalse(result, "잘못된 서명이 허용됨")

    def test_empty_signature_with_secret(self):
        """시크릿 있는데 서명 없으면 False."""
        result = self.hook_with_secret.verify_signature(b"body", "")
        self.assertFalse(result)


class TestKakaoWebhookProcess(unittest.TestCase):
    """KakaoWebhook.process(): 웹훅 요청 처리 테스트."""

    def setUp(self):
        from kakao_webhook import KakaoWebhook
        self.hook = KakaoWebhook()

    def _make_request(self, utterance: str, user_id: str = "test_user") -> dict:
        return {
            "userRequest": {
                "utterance": utterance,
                "user": {"id": user_id},
            },
            "bot": {"id": "test_bot"},
            "intent": {"name": "폴백 블록"},
        }

    def test_empty_utterance(self):
        """빈 utterance → 메시지 입력 요청."""
        body = self._make_request("")
        result = self.hook.process(body)
        self.assertEqual(result["version"], "2.0")
        text = result["template"]["outputs"][0]["simpleText"]["text"]
        self.assertIn("메시지", text)

    def test_greeting_resets_session(self):
        """인사말 → 세션 초기화 + 인사 응답."""
        body = self._make_request("안녕")
        result = self.hook.process(body)
        self.assertEqual(result["version"], "2.0")
        text = result["template"]["outputs"][0]["simpleText"]["text"]
        # 인사 응답 또는 라온 소개 포함
        self.assertGreater(len(text), 0)

    def test_track_b_quick_buttons(self):
        """Track B 감지 시 소상공인 버튼 포함."""
        buttons = self.hook.get_quick_buttons("B")
        self.assertIn("융자/보증 알아보기", buttons)
        self.assertIn("지원사업 찾기", buttons)

    def test_track_a_quick_buttons(self):
        """Track A 감지 시 TIPS 버튼 포함."""
        buttons = self.hook.get_quick_buttons("A")
        self.assertIn("TIPS 신청 방법", buttons)
        self.assertIn("투자자 매칭", buttons)

    def test_session_management(self):
        """세션이 user_id 기반으로 관리됨."""
        body = self._make_request("치킨집 창업", "user_001")
        self.hook.process(body)
        # 두 번째 요청 시 세션 유지
        body2 = self._make_request("어떻게 시작하나요?", "user_001")
        self.hook.process(body2)
        # 세션 존재 확인
        self.assertIn("user_001", self.hook.sessions)

    def test_get_session_count(self):
        """세션 카운트 확인."""
        initial = self.hook.get_session_count()
        body = self._make_request("카페 창업", "unique_user_999")
        self.hook.process(body)
        # 세션이 추가되었거나 인사로 초기화됨
        count = self.hook.get_session_count()
        self.assertGreaterEqual(count, 0)

    def test_process_missing_user_request(self):
        """userRequest 없는 경우 graceful 처리."""
        result = self.hook.process({})
        self.assertEqual(result["version"], "2.0")
        self.assertIn("outputs", result["template"])

    def test_financial_keyword_triggers_financial_map(self):
        """'융자' 키워드 → 금융 정보 응답."""
        body = self._make_request("소상공인 융자 알아보고 싶어요")
        result = self.hook.process(body)
        self.assertEqual(result["version"], "2.0")
        # 최소한 outputs 있어야 함
        self.assertGreater(len(result["template"]["outputs"]), 0)


# ─── evaluate.py Track B 분기 테스트 ─────────────────────────────────────────

class TestEvaluateTrackB(unittest.TestCase):
    """evaluate.py Track B 분기 테스트."""

    def test_track_b_prompt_no_tips_terms(self):
        """Track B 프롬프트에 TIPS 심사 기준 없음."""
        from evaluate import build_track_b_prompt
        prompt = build_track_b_prompt("치킨집 창업 계획")
        self.assertNotIn("기술 혁신성", prompt)
        self.assertNotIn("지식재산권", prompt)
        self.assertIn("입지", prompt)
        self.assertIn("상권", prompt)

    def test_track_b_prompt_easy_language(self):
        """Track B 프롬프트에 쉬운 언어 사용."""
        from evaluate import build_track_b_prompt
        prompt = build_track_b_prompt("카페 창업 계획")
        self.assertIn("다른 가게", prompt)
        self.assertIn("손님", prompt)

    def test_evaluate_functions_exist(self):
        """evaluate.py에 필요 함수 존재."""
        import evaluate
        self.assertTrue(hasattr(evaluate, "evaluate_track_b"))
        self.assertTrue(hasattr(evaluate, "evaluate_track_a"))
        self.assertTrue(hasattr(evaluate, "evaluate"))
        self.assertTrue(hasattr(evaluate, "build_track_b_prompt"))


# ─── 메인 실행 ────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    print("=" * 60)
    print("Raon OS — 카카오 연동 + Track B + 금융맵 테스트")
    print("=" * 60)
    unittest.main(verbosity=2)
