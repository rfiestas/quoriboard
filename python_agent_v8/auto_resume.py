"""Checkpoint discovery and metadata for the isolated V8 trainer."""

from __future__ import annotations

import json
import re
from datetime import datetime, timezone
from pathlib import Path

_STEP_PATTERN = re.compile(r"(?:^|_)(\d+)_steps\.zip$")


def checkpoint_step(path: Path) -> int:
    match = _STEP_PATTERN.search(path.name)
    return int(match.group(1)) if match else -1


def candidate_checkpoints(directory: str | Path) -> list[Path]:
    paths = [path for path in Path(directory).glob("*.zip") if checkpoint_step(path) >= 0]
    return sorted(paths, key=checkpoint_step, reverse=True)


def load_latest_valid(directory, loader):
    """Return ``(model, path)`` using step order and skip broken archives."""
    failures = []
    for path in candidate_checkpoints(directory):
        try:
            return loader(str(path)), path
        except Exception as exc:
            failures.append((path, str(exc)))
    return None, failures


def write_training_state(path: str | Path, *, timesteps: int, hyperparameters: dict) -> None:
    payload = {
        "timesteps": int(timesteps),
        "saved_at_utc": datetime.now(timezone.utc).isoformat(),
        "hyperparameters": hyperparameters,
    }
    Path(path).write_text(json.dumps(payload, indent=2, sort_keys=True), encoding="utf-8")