from __future__ import annotations

import argparse
from pathlib import Path

import yaml
from sb3_contrib import MaskablePPO
from stable_baselines3.common.callbacks import CheckpointCallback

from .auto_resume import load_latest_valid, write_training_state
from .env import QuoridorLeagueEnv


def train(config_path):
    config = yaml.safe_load(Path(config_path).read_text(encoding="utf-8"))
    checkpoint_dir = Path(config["checkpoint_dir"])
    checkpoint_dir.mkdir(parents=True, exist_ok=True)
    env = QuoridorLeagueEnv(
        grpc_target=config["grpc_target"], checkpoint_dir=checkpoint_dir,
        opponent_sampling=config["opponent_sampling"],
        opponent_pool_cache_size=config["opponent_pool_cache_size"],
    )

    model, checkpoint = load_latest_valid(checkpoint_dir, MaskablePPO.load)
    if model is None:
        model = MaskablePPO("MlpPolicy", env, learning_rate=config["learning_rate"],
                            n_steps=config["n_steps"], batch_size=config["batch_size"],
                            tensorboard_log=config["tensorboard_log_dir"])
    else:
        model.set_env(env)
        print(f"Resuming from {checkpoint} at {model.num_timesteps} timesteps")

    callback = CheckpointCallback(save_freq=config["save_freq"], save_path=str(checkpoint_dir),
                                  name_prefix="ppo_quoridor_v8")
    save_name = "ppo_quoridor_v8_interrupted"
    try:
        model.learn(total_timesteps=config["total_timesteps"], callback=callback, reset_num_timesteps=False)
        save_name = "ppo_quoridor_v8_final"
    except KeyboardInterrupt:
        print("\nInterrupcion solicitada. Guardando el progreso actual...")
        save_name = "ppo_quoridor_v8_interrupted"
    finally:
        model.save(checkpoint_dir / save_name)
        write_training_state(checkpoint_dir / "training_state.json", timesteps=model.num_timesteps,
                             hyperparameters={key: value for key, value in config.items() if key != "total_timesteps"})
        env.close()
        print(f"Checkpoint guardado: {checkpoint_dir / save_name}.zip")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Isolated Quoridor RL V8 trainer")
    parser.add_argument("--config", default="configs/rl_v8.yaml")
    train(parser.parse_args().config)