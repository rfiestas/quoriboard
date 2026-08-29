from __future__ import annotations

import argparse
import re
from pathlib import Path

import yaml
from sb3_contrib import MaskablePPO
from stable_baselines3.common.callbacks import BaseCallback, CallbackList, CheckpointCallback
from stable_baselines3.common.logger import configure

from .env import QuoridorLeagueEnv


class OutcomeMetricsCallback(BaseCallback):
    """Logs episode outcomes to TensorBoard with negligible overhead."""

    def __init__(self):
        super().__init__()
        self.wins = 0
        self.losses = 0
        self.draws = 0
        self.episodes = 0
        self.rollout_wins = 0
        self.rollout_losses = 0
        self.rollout_draws = 0
        self.rollout_episodes = 0

    def _on_step(self) -> bool:
        infos = self.locals.get("infos") or []
        dones = self.locals.get("dones") or []
        for info, done in zip(infos, dones):
            if not done:
                continue
            winner = int(info.get("winner", 0))
            self.episodes += 1
            self.rollout_episodes += 1
            if winner == 1:
                self.wins += 1
                self.rollout_wins += 1
            elif winner == 0:
                self.draws += 1
                self.rollout_draws += 1
            else:
                self.losses += 1
                self.rollout_losses += 1
        return True

    def _on_rollout_end(self) -> None:
        if self.rollout_episodes > 0:
            self.logger.record("train/episode_win_rate", self.rollout_wins / self.rollout_episodes)
            self.logger.record("train/episode_loss_rate", self.rollout_losses / self.rollout_episodes)
            self.logger.record("train/episode_draw_rate", self.rollout_draws / self.rollout_episodes)
        self.logger.record("train/episodes_total", self.episodes)
        self.logger.record("train/wins_total", self.wins)
        self.logger.record("train/losses_total", self.losses)
        self.logger.record("train/draws_total", self.draws)
        self.rollout_wins = 0
        self.rollout_losses = 0
        self.rollout_draws = 0
        self.rollout_episodes = 0


def _candidate_resume_checkpoints(checkpoint_dir):
    checkpoint_dir = Path(checkpoint_dir)
    candidates = set()
    interrupted = checkpoint_dir / "ppo_a136_league_interrupted.zip"
    if interrupted.exists():
        candidates.add(interrupted)
    final = checkpoint_dir / "ppo_a136_league_final.zip"
    if final.exists():
        candidates.add(final)
    for path in checkpoint_dir.glob("ppo_a136_league_*_steps.zip"):
        candidates.add(path)
    return sorted(candidates)


def _checkpoint_timesteps(path):
    match = re.search(r"_(\d+)_steps\.zip$", path.name)
    if match:
        return int(match.group(1))
    try:
        model = MaskablePPO.load(path, device="cpu")
    except Exception:
        return -1
    return int(getattr(model, "num_timesteps", -1))


def find_resume_checkpoint(checkpoint_dir):
    candidates = _candidate_resume_checkpoints(checkpoint_dir)
    if not candidates:
        return None
    scored = []
    for path in candidates:
        steps = _checkpoint_timesteps(path)
        if steps >= 0:
            scored.append((steps, path))
    if not scored:
        return None
    return max(scored, key=lambda item: item[0])[1]


def train(config_path):
    config = yaml.safe_load(Path(config_path).read_text(encoding="utf-8"))
    checkpoint_dir = Path(config["checkpoint_dir"])
    checkpoint_dir.mkdir(parents=True, exist_ok=True)

    env = QuoridorLeagueEnv(
        grpc_target=config["grpc_target"],
        checkpoint_dir=str(checkpoint_dir),
        opponent_mode=config.get("opponent_mode", "minimax"),
        opponent_sampling=config.get("opponent_sampling", "uniform"),
        opponent_pool_cache_size=config.get("opponent_pool_cache_size", 8),
        timeout=config.get("timeout", 5.0),
        self_model_path=config.get("self_model_path"),
    )

    initial_model_path = config.get("initial_model_path")
    if config.get("resume", False):
        resume_checkpoint = find_resume_checkpoint(checkpoint_dir)
        if resume_checkpoint is not None:
            initial_model_path = resume_checkpoint
    if initial_model_path:
        initial_model_path = Path(initial_model_path)
        if not initial_model_path.exists():
            raise FileNotFoundError(f"initial_model_path does not exist: {initial_model_path}")
        print(f"Resuming PPO model from {initial_model_path}")
        model = MaskablePPO.load(initial_model_path, env=env, device="cpu")
    else:
        model = MaskablePPO(
            "MlpPolicy",
            env,
            learning_rate=config["learning_rate"],
            n_steps=config["n_steps"],
            batch_size=config["batch_size"],
            tensorboard_log=config["tensorboard_log_dir"],
        )

    model.set_logger(configure(config["tensorboard_log_dir"], ["stdout", "tensorboard"]))

    checkpoint_callback = CheckpointCallback(
        save_freq=config["save_freq"],
        save_path=str(checkpoint_dir),
        name_prefix="ppo_a136_league",
    )
    outcome_callback = OutcomeMetricsCallback()
    callback = CallbackList([checkpoint_callback, outcome_callback])

    try:
        model.learn(total_timesteps=config["total_timesteps"], callback=callback, reset_num_timesteps=False)
        model.save(checkpoint_dir / "ppo_a136_league_final")
    except KeyboardInterrupt:
        interrupted_path = checkpoint_dir / "ppo_a136_league_interrupted"
        print(f"Interrupt received; saving current model to {interrupted_path}.zip")
        model.save(interrupted_path)
    finally:
        env.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Parallel Quoridor league trainer")
    parser.add_argument("--config", default="configs/ppo_a136_league.yaml")
    args = parser.parse_args()
    train(args.config)
