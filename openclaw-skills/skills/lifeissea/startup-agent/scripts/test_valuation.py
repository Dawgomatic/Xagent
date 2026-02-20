#!/usr/bin/env python3
"""Tests for valuation module"""
import json
import pytest
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))

from valuation import (
    scorecard_method,
    berkus_method,
    revenue_multiple_method,
    apply_kr_adjustments,
    estimate_valuation,
    format_report,
    cli_main,
)


class TestScorecardMethod:
    def test_default_scores_seed(self):
        r = scorecard_method("seed")
        assert r["valuation_억"] == 20  # mid * 1.0
        assert r["method"] == "Scorecard"

    def test_above_average(self):
        scores = {k: 1.5 for k in ["team", "market", "product", "competition", "marketing", "other"]}
        r = scorecard_method("seed", scores)
        assert r["valuation_억"] == 30.0  # 20 * 1.5
        assert r["weighted_factor"] == 1.5

    def test_below_average(self):
        scores = {k: 0.5 for k in ["team", "market", "product", "competition", "marketing", "other"]}
        r = scorecard_method("seed", scores)
        assert r["valuation_억"] == 10.0  # 20 * 0.5

    def test_pre_seed(self):
        r = scorecard_method("pre-seed")
        assert r["base_valuation_억"] == 7.5

    def test_series_a(self):
        r = scorecard_method("series-a")
        assert r["base_valuation_억"] == 65


class TestBerkusMethod:
    def test_default(self):
        r = berkus_method()
        # 5 items * $250K = $1.25M
        assert r["valuation_usd"] == 1_250_000
        assert r["method"] == "Berkus"

    def test_all_max(self):
        scores = {k: 1.0 for k in ["idea", "prototype", "team", "strategic_relations", "revenue"]}
        r = berkus_method(scores)
        assert r["valuation_usd"] == 2_500_000

    def test_all_zero(self):
        scores = {k: 0.0 for k in ["idea", "prototype", "team", "strategic_relations", "revenue"]}
        r = berkus_method(scores)
        assert r["valuation_usd"] == 0

    def test_clamp(self):
        scores = {"idea": 1.5, "prototype": -0.5, "team": 0.5, "strategic_relations": 0.5, "revenue": 0.5}
        r = berkus_method(scores)
        assert r["details"]["idea"]["score"] == 1.0
        assert r["details"]["prototype"]["score"] == 0.0


class TestRevenueMultiple:
    def test_no_revenue(self):
        r = revenue_multiple_method(0, 0, "ai")
        assert r["valuation_억"] is None

    def test_with_arr(self):
        arr = 10 * 100_000_000  # 10억
        r = revenue_multiple_method(arr, 0, "ai")
        assert r["valuation_억"] is not None
        assert r["arr_억"] == 10.0
        # AI: 10-20x -> mid = 15x -> 150억
        assert r["valuation_억"] == 150.0

    def test_mrr_to_arr(self):
        mrr = 1 * 100_000_000  # 1억/월 -> ARR 12억
        r = revenue_multiple_method(0, mrr, "saas")
        assert r["arr_억"] == 12.0

    def test_general_industry(self):
        arr = 5 * 100_000_000  # 5억
        r = revenue_multiple_method(arr, 0, "general")
        # 3-5x -> 15~25억, mid 20억
        assert r["valuation_억"] == 20.0


class TestKRAdjustments:
    def test_tips_premium(self):
        r = apply_kr_adjustments(20, tips=True)
        assert r["adjusted_valuation_억"] == 24.0  # 20 * 1.2

    def test_gov_rnd(self):
        r = apply_kr_adjustments(20, gov_rnd_억=10)
        # 10 * 0.125 = 1.25 -> 21.25
        assert r["adjusted_valuation_억"] == 21.2  # rounded

    def test_no_adjustments(self):
        r = apply_kr_adjustments(20)
        assert r["adjusted_valuation_억"] == 20
        assert r["adjustments"] == []


class TestEstimateValuation:
    def test_basic(self):
        r = estimate_valuation(stage="seed", industry="ai")
        assert r["status"] == "ok"
        assert "recommendation" in r
        assert "methods" in r
        assert r["recommendation"]["valuation_억"] > 0

    def test_with_revenue(self):
        r = estimate_valuation(stage="seed", industry="saas", revenue=5 * 100_000_000)
        rm = r["methods"]["revenue_multiple"]
        assert rm["valuation_억"] is not None

    def test_with_tips(self):
        r1 = estimate_valuation(stage="seed")
        r2 = estimate_valuation(stage="seed", tips=True)
        assert r2["recommendation"]["valuation_억"] > r1["recommendation"]["valuation_억"]

    def test_rationale(self):
        r = estimate_valuation(stage="seed", tips=True, gov_rnd=5)
        assert any("TIPS" in s for s in r["rationale"])
        assert any("R&D" in s for s in r["rationale"])

    def test_range(self):
        r = estimate_valuation()
        rec = r["recommendation"]
        assert rec["range_low_억"] < rec["valuation_억"] < rec["range_high_억"]


class TestFormatReport:
    def test_output(self):
        r = estimate_valuation(stage="seed", industry="ai", tips=True)
        text = format_report(r)
        assert "라온" in text
        assert "Scorecard" in text
        assert "Berkus" in text
        assert "억" in text


class TestCLI:
    def test_estimate_json(self, capsys):
        cli_main(["estimate", "--stage", "seed", "--industry", "ai", "--json"])
        out = capsys.readouterr().out
        data = json.loads(out)
        assert data["status"] == "ok"

    def test_estimate_text(self, capsys):
        cli_main(["estimate", "--stage", "pre-seed"])
        out = capsys.readouterr().out
        assert "라온" in out

    def test_tips_flag(self, capsys):
        cli_main(["estimate", "--tips", "--json"])
        out = capsys.readouterr().out
        data = json.loads(out)
        assert data["tips"] is True

    def test_revenue(self, capsys):
        cli_main(["estimate", "--revenue", "500000000", "--industry", "saas", "--json"])
        out = capsys.readouterr().out
        data = json.loads(out)
        assert data["methods"]["revenue_multiple"]["valuation_억"] is not None
