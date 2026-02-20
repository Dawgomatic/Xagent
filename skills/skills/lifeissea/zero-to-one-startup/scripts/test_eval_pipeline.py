"""eval_pipeline.py 단위 테스트"""
import json
import tempfile
import os
import sys
import pytest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, os.path.dirname(__file__))
import eval_pipeline as ep


@pytest.fixture
def tmp_dir(tmp_path):
    """Override file paths to temp dir."""
    gt = tmp_path / "ground_truth.jsonl"
    er = tmp_path / "eval_results.jsonl"
    hist = tmp_path / "history.jsonl"
    gt.write_text("")
    er.write_text("")
    hist.write_text("")
    with patch.object(ep, "GROUND_TRUTH", gt), \
         patch.object(ep, "EVAL_RESULTS", er), \
         patch.object(ep, "HISTORY", hist):
        yield tmp_path, gt, er, hist


def test_load_empty_jsonl(tmp_dir):
    _, gt, _, _ = tmp_dir
    assert ep.load_jsonl(gt) == []


def test_append_and_load(tmp_dir):
    _, gt, _, _ = tmp_dir
    ep.append_jsonl(gt, {"file": "a.pdf", "actual_result": "pass"})
    ep.append_jsonl(gt, {"file": "b.pdf", "actual_result": "fail"})
    entries = ep.load_jsonl(gt)
    assert len(entries) == 2
    assert entries[0]["file"] == "a.pdf"
    assert entries[1]["actual_result"] == "fail"


def test_find_ground_truth(tmp_dir):
    _, gt, _, _ = tmp_dir
    ep.append_jsonl(gt, {"file": "plan.pdf", "actual_result": "pass", "actual_score": 80})
    result = ep.find_ground_truth("plan.pdf")
    assert result["actual_score"] == 80
    assert ep.find_ground_truth("nope.pdf") is None


def test_find_eval_scores_from_history(tmp_dir):
    _, _, er, hist = tmp_dir
    hist.write_text(json.dumps({"files": ["plan.pdf"], "score": 75, "model": "qwen3:8b"}) + "\n")
    scores = ep.find_eval_scores("plan.pdf")
    assert len(scores) == 1
    assert scores[0]["score"] == 75


def test_find_eval_scores_from_eval_results(tmp_dir):
    _, _, er, _ = tmp_dir
    ep.append_jsonl(er, {"file": "plan.pdf", "llm_score": 72, "model": "qwen3:8b"})
    scores = ep.find_eval_scores("plan.pdf")
    assert len(scores) == 1
    assert scores[0]["llm_score"] == 72


def test_cmd_add(tmp_dir, capsys):
    _, gt, _, _ = tmp_dir
    args = type("A", (), {"file": "new.pdf", "program": "TIPS", "result": "pass", "score": 85, "notes": ""})()
    ep.cmd_add(args)
    entries = ep.load_jsonl(gt)
    assert len(entries) == 1
    assert entries[0]["actual_result"] == "pass"
    assert entries[0]["actual_score"] == 85
    out = capsys.readouterr().out
    assert "Ground truth 등록" in out


def test_report_empty(tmp_dir, capsys):
    args = type("A", (), {})()
    ep.cmd_report(args)
    out = capsys.readouterr().out
    assert "Ground truth 데이터 없음" in out


def test_report_with_data(tmp_dir, capsys):
    _, gt, er, _ = tmp_dir
    ep.append_jsonl(gt, {"file": "p.pdf", "actual_result": "pass", "actual_score": 80, "program": "TIPS"})
    ep.append_jsonl(er, {"file": "p.pdf", "llm_score": 78, "model": "qwen3:8b"})
    args = type("A", (), {})()
    ep.cmd_report(args)
    out = capsys.readouterr().out
    assert "정확도 리포트" in out
    assert "±10점 이내" in out


def test_pass_fail_prediction(tmp_dir, capsys):
    _, gt, er, _ = tmp_dir
    # LLM score 75 → predict pass, actual pass → correct
    ep.append_jsonl(gt, {"file": "a.pdf", "actual_result": "pass", "program": "TIPS"})
    ep.append_jsonl(er, {"file": "a.pdf", "llm_score": 75, "model": "test"})
    # LLM score 60 → predict fail, actual pass → wrong
    ep.append_jsonl(gt, {"file": "b.pdf", "actual_result": "pass", "program": "TIPS"})
    ep.append_jsonl(er, {"file": "b.pdf", "llm_score": 60, "model": "test"})
    args = type("A", (), {})()
    ep.cmd_report(args)
    out = capsys.readouterr().out
    assert "Ground Truth:" in out
