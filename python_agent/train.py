import os
import argparse
import torch
from collections import deque
from quoridor_env import QuoridorEnv
from ppo import PPOAgent
from telemetry import RLTelemetryServer

try:
    import yaml
except ModuleNotFoundError:
    yaml = None

DEFAULT_CONFIG_PATH = "configs/rl_trainer.yaml"


def _deep_update(dst, src):
    for k, v in src.items():
        if isinstance(v, dict) and isinstance(dst.get(k), dict):
            _deep_update(dst[k], v)
        else:
            dst[k] = v


def default_config():
    return {
        "python_trainer": {
            "grpc_host": "localhost:50051",
            "use_vertical_flip": True,
            "epochs": 4,
            "episodes_per_generation": 200,
            "max_generations_per_epoch": 0,
            "promotion_window": 100,
            "promotion_win_rate": 0.85,
            "update_timestep": 2000,
            "save_interval": 50,
            "log_interval": 10,
            "model_path": "ppo_quoridor.pth",
            "stage_id_offset": 0,
            "telemetry": {
                "web_dir": "web",
                "port": 8080,
                "window_size": 100,
            },
        }
    }


def load_config(path):
    cfg = default_config()
    if not path or not os.path.exists(path):
        return cfg

    if yaml is None:
        print("--> PyYAML no está instalado. Se usan valores por defecto para entrenamiento.")
        return cfg

    with open(path, "r", encoding="utf-8") as f:
        loaded = yaml.safe_load(f) or {}

    if isinstance(loaded, dict):
        _deep_update(cfg, loaded)
    return cfg


def train(config_path=DEFAULT_CONFIG_PATH):
    cfg = load_config(config_path)
    py_cfg = cfg.get("python_trainer", {})

    grpc_host = py_cfg.get("grpc_host", "localhost:50051")
    use_vertical_flip = bool(py_cfg.get("use_vertical_flip", True))
    epochs = int(py_cfg.get("epochs", 4))
    episodes_per_generation = int(py_cfg.get("episodes_per_generation", 200))
    max_generations_per_epoch = int(py_cfg.get("max_generations_per_epoch", 0))
    promotion_window = int(py_cfg.get("promotion_window", 100))
    promotion_win_rate = float(py_cfg.get("promotion_win_rate", 0.85))
    update_timestep = int(py_cfg.get("update_timestep", 2000))
    save_interval = int(py_cfg.get("save_interval", 50))
    log_interval = int(py_cfg.get("log_interval", 10))
    stage_id_offset = int(py_cfg.get("stage_id_offset", 0))
    model_path = py_cfg.get("model_path", "ppo_quoridor.pth")

    telemetry_cfg = py_cfg.get("telemetry", {})
    telemetry = RLTelemetryServer(
        web_dir=telemetry_cfg.get("web_dir", "web"),
        port=int(telemetry_cfg.get("port", 8080)),
        window_size=int(telemetry_cfg.get("window_size", 100)),
    )

    win_history = deque(maxlen=100)

    env = QuoridorEnv(host=grpc_host, use_vertical_flip=use_vertical_flip)
    agent = PPOAgent()

    if os.path.exists(model_path):
        print(f"--> Cargando checkpoint existente desde '{model_path}'...")
        checkpoint = torch.load(model_path)
        try:
            agent.policy.load_state_dict(checkpoint)
            agent.policy_old.load_state_dict(checkpoint)
            print("--> Pesos cargados correctamente.")
        except RuntimeError as exc:
            print(f"--> Checkpoint incompatible con la arquitectura actual: {exc}")
            print("--> Se inicia entrenamiento desde cero con el nuevo tamaño de estado.")
    else:
        print("--> Iniciando red neuronal desde cero.")

    timestep = 0
    global_episode = 0
    
    memory = {'states': [], 'masks': [], 'actions': [], 'logprobs': [], 'rewards': [], 'dones': []}

    print("\n--- Iniciando Entrenamiento PPO por Épocas/Generaciones en Quoridor ---\n")

    try:
        for epoch in range(1, epochs + 1):
            stage_id = stage_id_offset + epoch
            generation = 1
            epoch_promoted = False
            epoch_win_history = deque(maxlen=max(1, promotion_window))

            print(f"\n=== Epoch {epoch}/{epochs} | StageID {stage_id} ===")

            while not epoch_promoted:
                if max_generations_per_epoch > 0 and generation > max_generations_per_epoch:
                    print(
                        f"--> Epoch {epoch} alcanzó el límite de generaciones ({max_generations_per_epoch}) "
                        f"sin cumplir win-rate objetivo. Se detiene el entrenamiento."
                    )
                    torch.save(agent.policy.state_dict(), model_path)
                    return

                print(f"--- Generation {generation} (Epoch {epoch}, StageID {stage_id}) ---")

                generation_wins = 0
                generation_games = 0

                for generation_episode in range(1, episodes_per_generation + 1):
                    state, mask = env.reset(opponent_type=stage_id)
                    done = False
                    ep_reward = 0
                    ep_length = 0
                    ep_moves = 0
                    ep_walls = 0

                    while not done:
                        timestep += 1

                        action, log_prob, _ = agent.policy_old.get_action(state, mask)

                        if action < 12:
                            ep_moves += 1
                        else:
                            ep_walls += 1

                        next_state, next_mask, reward, done, winner = env.step(action)

                        memory['states'].append(state)
                        memory['masks'].append(mask)
                        memory['actions'].append(action)
                        memory['logprobs'].append(log_prob.item())
                        memory['rewards'].append(reward)
                        memory['dones'].append(done)

                        state = next_state
                        mask = next_mask
                        ep_reward += reward
                        ep_length += 1

                        if timestep % update_timestep == 0 and len(memory['states']) > 0:
                            agent.update(memory)

                    global_episode += 1
                    is_win = 1 if winner == 1 else 0
                    win_history.append(is_win)
                    epoch_win_history.append(is_win)
                    generation_wins += is_win
                    generation_games += 1

                    telemetry.record_episode(
                        episode=global_episode,
                        epoch=epoch,
                        generation=generation,
                        reward=ep_reward,
                        length=ep_length,
                        winner=winner,
                        moves=ep_moves,
                        walls=ep_walls,
                        stage_id=stage_id,
                    )

                    if save_interval > 0 and global_episode % save_interval == 0:
                        torch.save(agent.policy.state_dict(), model_path)
                        print(f" [Auto-Save] Checkpoint guardado en '{model_path}'")

                    if log_interval > 0 and global_episode % log_interval == 0:
                        wr_str = f"{(sum(win_history)/len(win_history))*100:.1f}%" if len(win_history) > 0 else "0.0%"
                        epoch_wr = (sum(epoch_win_history) / len(epoch_win_history)) if len(epoch_win_history) > 0 else 0.0
                        print(
                            f"Ep: {global_episode} | Epoch: {epoch}/{epochs} | Gen: {generation} ({generation_episode}/{episodes_per_generation})"
                            f" | StageID: {stage_id} | WinRateGlobal({len(win_history)}/100): {wr_str}"
                            f" | WinRateEpoch({len(epoch_win_history)}/{promotion_window}): {epoch_wr*100:.1f}%"
                            f" | Reward: {ep_reward:.3f} | Winner: {winner}"
                        )

                generation_wr = (generation_wins / generation_games) if generation_games > 0 else 0.0
                epoch_wr = (sum(epoch_win_history) / len(epoch_win_history)) if len(epoch_win_history) > 0 else 0.0
                print(
                    f"[RESUMEN] Epoch {epoch} Gen {generation} | WinRateGen: {generation_wr*100:.1f}% "
                    f"| WinRateEpoch({len(epoch_win_history)}/{promotion_window}): {epoch_wr*100:.1f}%"
                )

                if len(epoch_win_history) >= min(promotion_window, episodes_per_generation) and epoch_wr >= promotion_win_rate:
                    epoch_promoted = True
                    print(
                        f"✅ PROMOCIÓN: Epoch {epoch} superada (StageID {stage_id}) "
                        f"con WinRateEpoch {epoch_wr*100:.1f}%"
                    )
                else:
                    generation += 1

        torch.save(agent.policy.state_dict(), model_path)
        print(f"\n--> Entrenamiento finalizado. Modelo guardado en '{model_path}'.")

    except KeyboardInterrupt:
        print("\n\n--> Entrenamiento interrumpido.")
        torch.save(agent.policy.state_dict(), model_path)
        print(f"--> Progreso guardado en '{model_path}'!")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Entrenamiento PPO para Quoridor")
    parser.add_argument("--config", default=DEFAULT_CONFIG_PATH, help="Ruta al YAML de configuración RL")
    args = parser.parse_args()
    train(config_path=args.config)