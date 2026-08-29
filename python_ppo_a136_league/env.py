from __future__ import annotations

import random
from collections import OrderedDict
from pathlib import Path

import grpc
import gymnasium as gym
import numpy as np
from sb3_contrib import MaskablePPO

from python_agent_v8.pb import quoridor_rl_v8_pb2, quoridor_rl_v8_pb2_grpc

ACTION_SPACE_SIZE = 136
OBSERVATION_SIZE = 491


class QuoridorLeagueEnv(gym.Env):
    """Parallel RL league environment.

    This intentionally mirrors the real RL V8 environment while allowing a fixed
    opponent mode: self, heuristic, minimax, or mixed.
    """

    metadata = {"render_modes": []}

    def __init__(self, grpc_target="localhost:50052", checkpoint_dir="models_ppo_a136_league",
                 opponent_mode="minimax", opponent_sampling="uniform", opponent_pool_cache_size=8, timeout=5.0,
                 model_loader=None, self_model_path=None):
        super().__init__()
        self.channel = grpc.insecure_channel(grpc_target)
        self.stub = quoridor_rl_v8_pb2_grpc.QuoridorEnvStub(self.channel)
        self.timeout = float(timeout)
        self.checkpoint_dir = Path(checkpoint_dir)
        self.opponent_mode = str(opponent_mode).strip().lower()
        self.opponent_sampling = str(opponent_sampling).strip().lower()
        self._cache_size = max(0, int(opponent_pool_cache_size))
        self._model_cache = OrderedDict()
        self._model_loader = model_loader or MaskablePPO.load
        self.self_model_path = self_model_path
        self.self_model = None
        self.self_checkpoint = None
        self.action_space = gym.spaces.Discrete(ACTION_SPACE_SIZE)
        self.observation_space = gym.spaces.Box(low=0.0, high=1.0, shape=(OBSERVATION_SIZE,), dtype=np.float32)
        self.agent_player_id = 1
        self.current_mask = np.zeros(ACTION_SPACE_SIZE, dtype=bool)
        self._last_info = {}
        self._prev_phi = 0.0

        valid_modes = {
            "self",
            "heuristic",
            "heuristic_chaotic",
            "heuristic_balanced",
            "heuristic_aggressive",
            "heuristic_defensive",
            "heuristic_tier_beats_heuristic",
            "heuristic_tier_beats_minimax3",
            "heuristic_tier_beats_minimax4",
            "greedy_path",
            "minimax1",
            "minimax2",
            "minimax3",
            "minimax",
            "mixed",
        }
        if self.opponent_mode not in valid_modes:
            raise ValueError(
                "opponent_mode must be self, heuristic, heuristic_chaotic, heuristic_balanced, "
                "heuristic_aggressive, heuristic_defensive, heuristic_tier_beats_heuristic, "
                "heuristic_tier_beats_minimax3, heuristic_tier_beats_minimax4, minimax1, "
                "minimax2, minimax3, minimax, greedy_path, or mixed"
            )
        if self.opponent_sampling not in {"uniform", "recency_weighted"}:
            raise ValueError("opponent_sampling must be uniform or recency_weighted")

    def _candidate_self_checkpoints(self):
        checkpoints = list(self.checkpoint_dir.glob("*.zip"))
        if self.self_model_path:
            path = Path(self.self_model_path)
            if path.exists() and path.suffix.lower() == ".zip":
                checkpoints.append(path)
        unique = {}
        for checkpoint in checkpoints:
            unique[str(checkpoint)] = checkpoint
        return list(unique.values())

    def _load_self_opponent(self):
        checkpoints = self._candidate_self_checkpoints()
        if not checkpoints:
            self.self_model = None
            self.self_checkpoint = None
            return
        if self.opponent_sampling == "recency_weighted":
            ordered = sorted(checkpoints)
            weights = np.arange(1, len(ordered) + 1, dtype=np.float64)
            chosen = random.choices(ordered, weights=weights, k=1)[0]
        else:
            chosen = random.choice(checkpoints)
        self.self_checkpoint = chosen
        key = str(chosen)
        if key in self._model_cache:
            self.self_model = self._model_cache.pop(key)
            self._model_cache[key] = self.self_model
            return
        try:
            model = self._model_loader(key, device="cpu")
        except Exception:
            self.self_model = None
            return
        if self._cache_size:
            self._model_cache[key] = model
            while len(self._model_cache) > self._cache_size:
                self._model_cache.popitem(last=False)
        self.self_model = model

    def _to_array(self, observation):
        values = np.asarray(observation, dtype=np.float32)
        if values.size != OBSERVATION_SIZE:
            raise ValueError(f"expected {OBSERVATION_SIZE} observation values, got {values.size}")
        return values.reshape(self.observation_space.shape)

    def _mask(self, mask):
        values = np.asarray(mask, dtype=bool)
        if values.size != ACTION_SPACE_SIZE:
            raise ValueError(f"expected {ACTION_SPACE_SIZE} mask values, got {values.size}")
        return values

    def reset(self, *, seed=None, options=None):
        super().reset(seed=seed)
        if self.opponent_mode == "self":
            self._load_self_opponent()
        self.agent_player_id = 1
        self._prev_phi = 0.0
        response = self.stub.Reset(quoridor_rl_v8_pb2.ResetRequest(), timeout=self.timeout)
        response = self._play_self_turn(response)
        self.current_mask = self._mask(response.action_mask)
        self._last_info = dict(response.info)
        return self._to_array(response.observation), self._last_info.copy()

    def step(self, action):
        action = int(action)
        if not self.action_space.contains(action) or not self.current_mask[action]:
            raise ValueError(f"invalid masked action: {action}")

        response = self.stub.Step(quoridor_rl_v8_pb2.StepRequest(action=action), timeout=self.timeout)
        response = self._play_self_turn(response)
        self.current_mask = self._mask(response.action_mask)
        self._last_info = dict(response.info)
        reward = self._agent_reward(response.done, self._last_info)
        return self._to_array(response.observation), reward, bool(response.done), False, self._last_info.copy()

    SHAPING_COEF = 0.01

    # Debe coincidir con drawReasonNone/Timeout/Repetition en server.go
    DRAW_REASON_REPETITION = 2

    def _agent_reward(self, done, info):
        if done:
            winner = int(info.get("winner", 0))
            if winner == self.agent_player_id:
                return 1.0
            if winner == 0:
                draw_reason = int(info.get("draw_reason", 0))
                if draw_reason == self.DRAW_REASON_REPETITION:
                    return -0.3  # tablas por bucle repetitivo: estancamiento evitable
                return -0.1  # tablas por límite de turnos: puede pasar jugando honesto
            return -1.0

        agent_len = info.get("agent_path_len", 0.0)
        opp_len = info.get("opp_path_len", 0.0)
        phi = opp_len - agent_len
        shaping = self.SHAPING_COEF * (phi - self._prev_phi)
        self._prev_phi = phi
        return shaping

    def action_masks(self):
        return self.current_mask.copy()

    def _play_self_turn(self, response):
        while self.opponent_mode == "self" and not response.done and response.current_player != self.agent_player_id:
            mask = self._mask(response.action_mask)
            valid = np.flatnonzero(mask)
            if valid.size == 0:
                return response
            if self.self_model is None:
                action = int(random.choice(valid.tolist()))
            else:
                observation = self._to_array(response.observation)
                action, _ = self.self_model.predict(observation, action_masks=mask, deterministic=False)
                action = int(action)
                if action < 0 or action >= ACTION_SPACE_SIZE or not mask[action]:
                    action = int(random.choice(valid.tolist()))
            response = self.stub.Step(quoridor_rl_v8_pb2.StepRequest(action=action), timeout=self.timeout)
        return response

    def close(self):
        self.channel.close()