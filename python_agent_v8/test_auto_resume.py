import tempfile
import unittest
from pathlib import Path

from .auto_resume import candidate_checkpoints, load_latest_valid


class TestAutoResume(unittest.TestCase):
    def test_uses_step_number_and_skips_corrupt_latest(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "ppo_quoridor_v8_10_steps.zip").touch()
            (root / "ppo_quoridor_v8_20_steps.zip").touch()
            (root / "unrelated.zip").touch()

            loaded, path = load_latest_valid(root, lambda value: (_ for _ in ()).throw(ValueError("broken")) if value.endswith("20_steps.zip") else "ok")

            self.assertEqual(loaded, "ok")
            self.assertEqual(path.name, "ppo_quoridor_v8_10_steps.zip")
            self.assertEqual([p.name for p in candidate_checkpoints(root)], [
                "ppo_quoridor_v8_20_steps.zip", "ppo_quoridor_v8_10_steps.zip"
            ])


if __name__ == "__main__":
    unittest.main()