"""Trust cache tests for TTL behavior and grant revoke flows."""

from __future__ import annotations

import time
from pathlib import Path

import cache


def test_cache_grant_and_revoke(tmp_path: Path) -> None:
    cache_path = tmp_path / "cache.json"

    grant = cache.grant(
        action="send_email",
        channel="discord",
        scope="specific",
        ttl_seconds=60,
        path=str(cache_path),
    )

    assert cache.check("send_email", "discord", path=str(cache_path)) is not None
    cache.revoke(grant["id"], path=str(cache_path))
    assert cache.check("send_email", "discord", path=str(cache_path)) is None


def test_cache_ttl_expiry_and_revoke_all(tmp_path: Path) -> None:
    cache_path = tmp_path / "cache.json"

    cache.grant("send_email", "discord", scope="specific", ttl_seconds=1, path=str(cache_path))
    cache.grant("delete_file", "discord", scope="action_type", ttl_seconds=120, path=str(cache_path))

    time.sleep(1.1)
    assert cache.check("send_email", "discord", path=str(cache_path)) is None
    assert cache.check("delete_file", "discord", path=str(cache_path)) is not None

    removed = cache.revoke_all(path=str(cache_path))
    assert removed == 2
    assert cache.list_active(path=str(cache_path)) == []
