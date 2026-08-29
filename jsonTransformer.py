"""
One-time offline script. Not part of the production bot.

Replaces onnxTransformer.py + export_weights.py: goes directly from the training
.zip file (sb3_contrib.MaskablePPO) to the JSON loaded by the Go bot,
without generating an intermediate .onnx file.

Requirements: pip install sb3_contrib torch
"""
import argparse
import datetime
import glob
import json
import os
import re
import torch
from sb3_contrib import MaskablePPO

# Recognized activation modules. If trained with another activation in the future
# (e.g., ReLU instead of Tanh), it detects it automatically: no need to touch this script.
# If an unlisted activation appears, the script fails explicitly.
ACT_MAP = {
    "Tanh": "tanh",
    "ReLU": "relu",
    "Identity": "none",
}


def get_latest_checkpoint(checkpoints_dir):
    """Finds the zip file with the highest step count in the checkpoints directory."""
    pattern = os.path.join(checkpoints_dir, "*_steps.zip")
    checkpoint_files = glob.glob(pattern)
    
    if not checkpoint_files:
        raise FileNotFoundError(f"No checkpoint files found matching pattern {pattern}")

    latest_file = None
    max_steps = -1

    for file_path in checkpoint_files:
        filename = os.path.basename(file_path)
        # Extract the numeric steps from the filename using regex
        match = re.search(r"_(\d+)_steps\.zip$", filename)
        if match:
            steps = int(match.group(1))
            if steps > max_steps:
                max_steps = steps
                latest_file = file_path

    if not latest_file:
        raise ValueError(f"Could not parse step numbers from checkpoints in {checkpoints_dir}")

    return latest_file


def make_layer(name, linear, activation):
    w = linear.weight.detach().cpu().numpy()  # (out_features, in_features)
    b = linear.bias.detach().cpu().numpy()    # (out_features,)
    return {
        "name": name,
        "activation": activation,
        "in_features": int(w.shape[1]),
        "out_features": int(w.shape[0]),
        "weight": w.astype("float32").flatten().tolist(),  # row-major: out x in
        "bias": b.astype("float32").tolist(),
    }


def export_weights(zip_path, json_path):
    print(f"Loading MaskablePPO model from {zip_path}...")
    model = MaskablePPO.load(zip_path, device="cpu")
    policy = model.policy
    policy.eval()

    # policy.mlp_extractor.policy_net is an nn.Sequential like
    # [Linear, Tanh, Linear, Tanh, ...]. policy.action_net is the final head
    # (raw logits, no activation).
    seq = list(policy.mlp_extractor.policy_net)

    layers = []
    i, n = 0, len(seq)
    while i < n:
        mod = seq[i]
        if not isinstance(mod, torch.nn.Linear):
            raise ValueError(
                f"Expected a Linear layer at position {i} of policy_net, "
                f"but found {type(mod).__name__}. Unexpected architecture: "
                f"please inspect the model manually."
            )
        activation = "none"
        if i + 1 < n:
            act_type = type(seq[i + 1]).__name__
            if act_type in ACT_MAP:
                activation = ACT_MAP[act_type]
                i += 1  # consume the activation module as well
            else:
                raise ValueError(
                    f"Unknown activation '{act_type}' after layer {len(layers)}. "
                    f"Add it to ACT_MAP if intentional, and also handle it in policy.go."
                )
        layers.append(make_layer(f"policy_net.{len(layers)}", mod, activation))
        i += 1

    # Action head: raw logits, masking + softmax are handled externally
    layers.append(make_layer("action_net", policy.action_net, "none"))

    # Ensure destination directory exists
    dest_dir = os.path.dirname(os.path.abspath(json_path))
    os.makedirs(dest_dir, exist_ok=True)

    # 1. Save main model weights JSON
    with open(json_path, "w") as f:
        json.dump({"layers": layers}, f)

    print(f"Exported weights to {json_path}")

    # 2. Generate and save extra metadata JSON file
    meta_path = os.path.join(dest_dir, "model_meta.json")
    metadata = {
        "source_filename": os.path.basename(zip_path),
        "extracted_at_utc": datetime.datetime.utcnow().isoformat() + "Z",
        "total_layers": len(layers),
        "architecture_summary": [
            {
                "name": l["name"],
                "in_features": l["in_features"],
                "out_features": l["out_features"],
                "activation": l["activation"]
            }
            for l in layers
        ]
    }

    with open(meta_path, "w") as f:
        json.dump(metadata, f, indent=4)

    print(f"Exported metadata to {meta_path}")
    for l in layers:
        print(f"  {l['name']}: {l['in_features']} -> {l['out_features']} ({l['activation']})")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Export MaskablePPO weights directly to Go bot JSON format.")
    parser.add_argument(
        "--source", 
        type=str, 
        default=None, 
        help="Path to the model .zip file. If not provided, the script automatically picks the latest checkpoint."
    )
    parser.add_argument(
        "--destination", 
        type=str, 
        default=r"assets\model\quoridor_rl_model.json", 
        help="Path to the output JSON file."
    )

    args = parser.parse_args()

    # Determine source path
    if args.source:
        source_path = args.source
    else:
        checkpoints_directory = r"models\ppo_a136_league\checkpoints"
        print(f"No source specified. Searching for latest checkpoint in '{checkpoints_directory}'...")
        source_path = get_latest_checkpoint(checkpoints_directory)

    export_weights(source_path, args.destination)