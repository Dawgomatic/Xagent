#!/usr/bin/env python3
"""Tests for gamification engine."""

import json
import os
import sys
import tempfile
import unittest
from datetime import date, timedelta

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, SCRIPT_DIR)

from gamification import (
    XP_TABLE, LEVELS, BADGES,
    _default_profile, load_profile, save_profile,
    get_level, get_next_level_xp,
    check_badges, check_streak,
    add_xp, format_profile, format_xp_gain,
)


class TestGetLevel(unittest.TestCase):
    def test_zero_xp(self):
        level, title = get_level(0)
        self.assertEqual(level, 1)
        self.assertIn("예비창업자", title)

    def test_100_xp(self):
        level, title = get_level(100)
        self.assertEqual(level, 2)
        self.assertIn("초기창업자", title)

    def test_300_xp(self):
        level, title = get_level(300)
        self.assertEqual(level, 3)

    def test_1000_xp(self):
        level, title = get_level(1000)
        self.assertEqual(level, 5)
        self.assertIn("유니콘", title)

    def test_between_levels(self):
        level, title = get_level(150)
        self.assertEqual(level, 2)

    def test_next_level_xp(self):
        self.assertEqual(get_next_level_xp(0), 100)
        self.assertEqual(get_next_level_xp(50), 100)
        self.assertEqual(get_next_level_xp(100), 300)
        self.assertIsNone(get_next_level_xp(1000))


class TestProfileIO(unittest.TestCase):
    def test_load_missing_file(self):
        profile = load_profile("/tmp/nonexistent_raon_profile.json")
        self.assertEqual(profile["xp"], 0)
        self.assertIn("badges", profile)

    def test_save_and_load(self):
        with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
            path = f.name
        try:
            p = _default_profile()
            p["xp"] = 42
            save_profile(p, path)
            loaded = load_profile(path)
            self.assertEqual(loaded["xp"], 42)
        finally:
            os.unlink(path)

    def test_load_corrupt_json(self):
        with tempfile.NamedTemporaryFile(suffix=".json", delete=False, mode="w") as f:
            f.write("{bad json")
            path = f.name
        try:
            profile = load_profile(path)
            self.assertEqual(profile["xp"], 0)  # defaults
        finally:
            os.unlink(path)


class TestCheckBadges(unittest.TestCase):
    def test_first_eval_badge(self):
        profile = _default_profile()
        badges = check_badges(profile, "evaluate")
        self.assertIn("first_eval", badges)

    def test_first_eval_not_duplicate(self):
        profile = _default_profile()
        profile["badges"] = ["first_eval"]
        badges = check_badges(profile, "evaluate")
        self.assertNotIn("first_eval", badges)

    def test_club_90(self):
        profile = _default_profile()
        badges = check_badges(profile, "evaluate", {"score": 92})
        self.assertIn("club_90", badges)

    def test_club_90_below(self):
        profile = _default_profile()
        badges = check_badges(profile, "evaluate", {"score": 85})
        self.assertNotIn("club_90", badges)

    def test_growth_king(self):
        profile = _default_profile()
        badges = check_badges(profile, "evaluate", {"score_improvement": 25})
        self.assertIn("growth_king", badges)

    def test_match_master(self):
        profile = _default_profile()
        profile["stats"]["match"] = 3
        badges = check_badges(profile, "match")
        self.assertIn("match_master", badges)

    def test_draft_artisan(self):
        profile = _default_profile()
        profile["stats"]["draft"] = 3
        badges = check_badges(profile, "draft")
        self.assertIn("draft_artisan", badges)


class TestCheckStreak(unittest.TestCase):
    def test_empty_streak(self):
        profile = _default_profile()
        streak = check_streak(profile)
        self.assertEqual(streak, 1)  # today added

    def test_consecutive_days(self):
        profile = _default_profile()
        today = date.today()
        profile["streak_days"] = [
            (today - timedelta(days=i)).isoformat()
            for i in range(6, -1, -1)
        ]
        streak = check_streak(profile)
        self.assertEqual(streak, 7)

    def test_broken_streak(self):
        profile = _default_profile()
        today = date.today()
        profile["streak_days"] = [
            (today - timedelta(days=5)).isoformat(),
            (today - timedelta(days=3)).isoformat(),
            (today - timedelta(days=1)).isoformat(),
            today.isoformat(),
        ]
        streak = check_streak(profile)
        self.assertEqual(streak, 2)


class TestAddXP(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.NamedTemporaryFile(suffix=".json", delete=False)
        self.tmp.close()
        save_profile(_default_profile(), self.tmp.name)

    def tearDown(self):
        os.unlink(self.tmp.name)

    def test_basic_evaluate_xp(self):
        result = add_xp("evaluate", {"score": 70}, self.tmp.name)
        self.assertEqual(result["xp_gained"], 20)
        self.assertIn("first_eval", result["new_badge_ids"])

    def test_score_improvement_bonus(self):
        add_xp("evaluate", {"score": 60}, self.tmp.name)
        result = add_xp("evaluate", {"score": 75}, self.tmp.name)
        # re_evaluate_bonus (30) + score_improve_10 (50) + base (20) = 100
        self.assertEqual(result["xp_gained"], 100)

    def test_first_80_score_bonus(self):
        result = add_xp("evaluate", {"score": 85}, self.tmp.name)
        # base(20) + first_80(200) = 220
        self.assertEqual(result["xp_gained"], 220)

    def test_level_up(self):
        # Push to near level 2
        profile = load_profile(self.tmp.name)
        profile["xp"] = 95
        save_profile(profile, self.tmp.name)
        result = add_xp("match", profile_path=self.tmp.name)
        self.assertTrue(result["leveled_up"])
        self.assertEqual(result["level"], 2)

    def test_90_club_badge(self):
        result = add_xp("evaluate", {"score": 95}, self.tmp.name)
        self.assertIn("club_90", result["new_badge_ids"])

    def test_stats_increment(self):
        add_xp("match", profile_path=self.tmp.name)
        add_xp("match", profile_path=self.tmp.name)
        profile = load_profile(self.tmp.name)
        self.assertEqual(profile["stats"]["match"], 2)


class TestFormatters(unittest.TestCase):
    def test_format_profile(self):
        profile = _default_profile()
        profile["xp"] = 150
        profile["badges"] = ["first_eval"]
        text = format_profile(profile)
        self.assertIn("라온 프로필", text)
        self.assertIn("첫 평가", text)

    def test_format_xp_gain(self):
        text = format_xp_gain(20, ["🏆 첫 평가"], True, "🌿 초기창업자")
        self.assertIn("+20 XP", text)
        self.assertIn("레벨 업", text)
        self.assertIn("첫 평가", text)

    def test_format_xp_gain_empty(self):
        text = format_xp_gain(0)
        self.assertEqual(text, "")


if __name__ == "__main__":
    unittest.main()
