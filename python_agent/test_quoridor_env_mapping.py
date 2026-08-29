import unittest
import os
import sys


sys.path.insert(0, os.path.dirname(__file__))


try:
    import numpy as np
    from quoridor_env import (
        map_action_vertical_flip,
        map_action_mask_vertical_flip,
        map_state_vertical_flip,
    )
except ModuleNotFoundError:
    np = None


@unittest.skipIf(np is None, "numpy no esta instalado en este entorno")
class TestQuoridorEnvMapping(unittest.TestCase):
    def test_action_mapping_is_involution(self):
        for action_id in range(140):
            mapped = map_action_vertical_flip(action_id)
            self.assertEqual(map_action_vertical_flip(mapped), action_id)

    def test_mask_mapping_is_involution(self):
        mask = np.zeros(140, dtype=bool)
        mask[[0, 1, 5, 13, 79, 139]] = True

        mapped = map_action_mask_vertical_flip(mask)
        unmapped = map_action_mask_vertical_flip(mapped)

        self.assertEqual(mapped.shape, (140,))
        self.assertTrue(np.array_equal(mask, unmapped))

    def test_state_mapping_is_involution(self):
        state = np.arange(491, dtype=np.float32)

        mapped = map_state_vertical_flip(state)
        unmapped = map_state_vertical_flip(mapped)

        self.assertEqual(mapped.shape, (491,))
        self.assertTrue(np.array_equal(state, unmapped))


if __name__ == "__main__":
    unittest.main()
