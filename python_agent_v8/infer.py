from __future__ import annotations

import argparse
import json
import sys

import numpy as np
from sb3_contrib import MaskablePPO


def main():
    parser = argparse.ArgumentParser(description="Persistent RL V8 inference sidecar")
    parser.add_argument("--checkpoint", required=True)
    args = parser.parse_args()
    model = MaskablePPO.load(args.checkpoint, device="cpu")
    for line in sys.stdin:
        try:
            request = json.loads(line)
            observation = np.asarray(request["observation"], dtype=np.float32)
            action_mask = np.asarray(request["action_mask"], dtype=bool)
            if observation.size != 491 or action_mask.size != 136:
                raise ValueError("invalid V8 observation or mask size")
            action, _ = model.predict(observation, action_masks=action_mask, deterministic=True)
            print(json.dumps({"action": int(action)}), flush=True)
        except Exception as error:
            print(json.dumps({"action": -1, "error": str(error)}), flush=True)


if __name__ == "__main__":
    main()