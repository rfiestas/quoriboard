import unittest
import os
import sys


sys.path.insert(0, os.path.dirname(__file__))


try:
    import grpc
    import numpy as np
    from quoridor_env import QuoridorEnv
except ModuleNotFoundError:
    grpc = None
    np = None
    QuoridorEnv = None


@unittest.skipIf(grpc is None or np is None or QuoridorEnv is None, "faltan dependencias python (grpcio/numpy)")
class TestQuoridorEnvGrpcIntegration(unittest.TestCase):
    def setUp(self):
        self.env_raw = QuoridorEnv(use_vertical_flip=False)
        self.env_flip = QuoridorEnv(use_vertical_flip=True)

        try:
            grpc.channel_ready_future(self.env_raw.channel).result(timeout=2.0)
        except Exception as exc:  # pragma: no cover - depends on external server process
            self.skipTest(f"servidor gRPC no disponible en localhost:50051: {exc}")

    def test_reset_and_step_with_canonical_mapping(self):
        state, mask = self.env_flip.reset(opponent_type=0)

        self.assertEqual(state.shape, (491,))
        self.assertEqual(mask.shape, (140,))
        self.assertGreater(np.sum(mask), 0)

        action = int(np.where(mask)[0][0])
        next_state, next_mask, reward, done, winner = self.env_flip.step(action)

        self.assertEqual(next_state.shape, (491,))
        self.assertEqual(next_mask.shape, (140,))
        self.assertIsInstance(reward, float)
        self.assertIsInstance(done, bool)
        self.assertIsInstance(int(winner), int)

    def test_mask_mapping_matches_real_action_legality(self):
        _, mask_raw = self.env_raw.reset(opponent_type=0)
        mask_can = self.env_flip.real_to_canonical_mask(mask_raw)

        self.assertEqual(mask_raw.shape, (140,))
        self.assertEqual(mask_can.shape, (140,))

        for action in range(140):
            real_action = self.env_flip.canonical_to_real_action(action)
            self.assertEqual(bool(mask_can[action]), bool(mask_raw[real_action]))


if __name__ == "__main__":
    unittest.main()
