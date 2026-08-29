import json
import os
import http.server
import socketserver
import threading
from collections import deque

class RLTelemetryServer:
    def __init__(self, web_dir="web", port=8080, window_size=100):
        self.web_dir = web_dir
        self.port = port
        self.filepath = os.path.join(web_dir, "data.json")
        os.makedirs(web_dir, exist_ok=True)
        
        # Ventanas deslizantes para métricas recientes
        self.recent_wins = deque(maxlen=window_size)
        self.recent_rewards = deque(maxlen=window_size)
        self.recent_lengths = deque(maxlen=window_size)
        self.recent_moves = deque(maxlen=window_size)
        self.recent_walls = deque(maxlen=window_size)
        self.history = []
        self.total_turns = 0
        self.turns_by_epoch = {}

        self._start_http_server()

    def _start_http_server(self):
        web_dir = self.web_dir
        class CustomHandler(http.server.SimpleHTTPRequestHandler):
            def __init__(self, *args, **kwargs):
                super().__init__(*args, directory=web_dir, **kwargs)

        def serve():
            with socketserver.TCPServer(("", self.port), CustomHandler) as httpd:
                httpd.serve_forever()

        thread = threading.Thread(target=serve, daemon=True)
        thread.start()

    def record_episode(self, episode, epoch, generation, reward, length, winner, moves=0, walls=0, stage_id=0):
        is_win = 1 if winner == 1 else 0
        self.recent_wins.append(is_win)
        self.recent_rewards.append(reward)
        self.recent_lengths.append(length)
        self.recent_moves.append(moves)
        self.recent_walls.append(walls)
        self.total_turns += int(length)
        self.turns_by_epoch[epoch] = self.turns_by_epoch.get(epoch, 0) + int(length)

        # Actualiza el JSON cada 10 episodios
        if episode % 10 == 0:
            win_rate = (sum(self.recent_wins) / len(self.recent_wins)) * 100
            avg_reward = sum(self.recent_rewards) / len(self.recent_rewards)
            avg_length = sum(self.recent_lengths) / len(self.recent_lengths)
            avg_moves = sum(self.recent_moves) / len(self.recent_moves)
            avg_walls = sum(self.recent_walls) / len(self.recent_walls)

            snapshot = {
                "episode": episode,
                "epoch": epoch,
                "generation": generation,
                "stage_id": stage_id,
                "win_rate": round(win_rate, 1),
                "avg_reward": round(avg_reward, 4),
                "avg_length": round(avg_length, 1),
                "avg_moves": round(avg_moves, 1),
                "avg_walls": round(avg_walls, 1),
                "total_turns": self.total_turns,
                "epoch_total_turns": self.turns_by_epoch.get(epoch, 0),
                "last_winner": "PPO Agent" if winner == 1 else "Opponent"
            }
            self.history.append(snapshot)

            data = {
                "current_episode": episode,
                "current_epoch": epoch,
                "current_generation": generation,
                "current_stage_id": stage_id,
                "latest_win_rate": round(win_rate, 1),
                "latest_avg_reward": round(avg_reward, 4),
                "latest_avg_length": round(avg_length, 1),
                "latest_avg_moves": round(avg_moves, 1),
                "latest_avg_walls": round(avg_walls, 1),
                "latest_total_turns": self.total_turns,
                "latest_epoch_total_turns": self.turns_by_epoch.get(epoch, 0),
                "history": self.history
            }

            with open(self.filepath, "w") as f:
                json.dump(data, f, indent=2)