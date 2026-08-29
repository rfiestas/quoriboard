from quoridor_env import QuoridorEnv
import numpy as np

env = QuoridorEnv()
state, mask = env.reset(opponent_type=0)

print("¡Conexión exitosa con Go!")
print(f"Tamaño del Vector de Estado: {state.shape} (Esperado: 491)")
print(f"Acciones legales disponibles en el turno 1: {np.sum(mask)} de 140")

# Ejecutamos una jugada aleatoria válida
valid_actions = np.where(mask)[0]
random_action = np.random.choice(valid_actions)

next_state, next_mask, reward, done, _ = env.step(random_action)
print(f"Turno ejecutado correctamente. Recompensa: {reward}")