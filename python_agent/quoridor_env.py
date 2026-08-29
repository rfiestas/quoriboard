import grpc
import numpy as np
import sys
import os

# Permitir importaciones locales desde la carpeta pb
sys.path.append(os.path.join(os.path.dirname(__file__), 'pb'))
import quoridor_pb2
import quoridor_pb2_grpc


# Vertical board flip (Y axis mirror from agent perspective on a 9x9 board).
# Action encoding must match internal/grpc/server.go.
MOVE_VERTICAL_FLIP_MAP = np.array([
    1, 0, 2, 3,
    5, 4, 6, 7,
    10, 11, 8, 9,
], dtype=np.int32)

ACTION_SPACE_SIZE = 140
STATE_TENSOR_BOARD_SIZE = 486
STATE_TENSOR_SIZE = 491
PLANE_SIZE = 81


def map_action_vertical_flip(action_id):
    action_id = int(action_id)

    # Pawn moves / jumps / diagonals (0..11)
    if 0 <= action_id < 12:
        return int(MOVE_VERTICAL_FLIP_MAP[action_id])

    # Horizontal walls (12..75): index = 12 + (x * 8) + y
    if 12 <= action_id < 76:
        idx = action_id - 12
        wx = idx // 8
        wy = idx % 8
        return int(12 + (wx * 8) + (7 - wy))

    # Vertical walls (76..139): index = 76 + (x * 8) + y
    if 76 <= action_id < 140:
        idx = action_id - 76
        wx = idx // 8
        wy = idx % 8
        return int(76 + (wx * 8) + (7 - wy))

    raise ValueError(f"action_id fuera de rango: {action_id}")


def map_action_mask_vertical_flip(mask):
    if len(mask) != ACTION_SPACE_SIZE:
        raise ValueError(f"Máscara inválida: se esperaban {ACTION_SPACE_SIZE} acciones, llegaron {len(mask)}")

    src = np.asarray(mask, dtype=bool)
    dst = np.zeros_like(src)

    # canonical_mask[a] = real_mask[f(a)]
    for a in range(ACTION_SPACE_SIZE):
        dst[a] = src[map_action_vertical_flip(a)]

    return dst


def flip_vertical_9x9_plane(flat_plane):
    plane = np.asarray(flat_plane, dtype=np.float32).reshape(9, 9)
    return np.flip(plane, axis=1).reshape(-1)


def flip_vertical_wall_plane(flat_plane):
    plane = np.asarray(flat_plane, dtype=np.float32).reshape(9, 9)
    # Wall coordinates in Go use y in [0..7], so we only mirror that band.
    out = np.zeros_like(plane)
    out[:, :8] = np.flip(plane[:, :8], axis=1)
    # Keep y=8 untouched to preserve information and make the transform involutive.
    out[:, 8] = plane[:, 8]
    return out.reshape(-1)


def map_state_vertical_flip(state):
    if len(state) < STATE_TENSOR_BOARD_SIZE:
        raise ValueError(f"Estado inválido: se esperaban al menos {STATE_TENSOR_BOARD_SIZE} floats, llegaron {len(state)}")

    src = np.asarray(state, dtype=np.float32)
    out = np.zeros_like(src)

    # Planes: [agent][opponent][h_walls][v_walls][agent_dist][opp_dist]
    out[0:81] = flip_vertical_9x9_plane(src[0:81])
    out[81:162] = flip_vertical_9x9_plane(src[81:162])
    out[162:243] = flip_vertical_wall_plane(src[162:243])
    out[243:324] = flip_vertical_wall_plane(src[243:324])
    out[324:405] = flip_vertical_9x9_plane(src[324:405])
    out[405:486] = flip_vertical_9x9_plane(src[405:486])

    # Scalar/extra features are orientation-invariant in this mapping.
    if len(src) > STATE_TENSOR_BOARD_SIZE:
        out[STATE_TENSOR_BOARD_SIZE:] = src[STATE_TENSOR_BOARD_SIZE:]

    return out

class QuoridorEnv:
    def __init__(self, host="localhost:50051", use_vertical_flip=False):
        self.channel = grpc.insecure_channel(host)
        self.stub = quoridor_pb2_grpc.QuoridorEnvStub(self.channel)
        self.use_vertical_flip = bool(use_vertical_flip)

    def canonical_to_real_action(self, action_id):
        if not self.use_vertical_flip:
            return int(action_id)
        return map_action_vertical_flip(action_id)

    def real_to_canonical_action(self, action_id):
        if not self.use_vertical_flip:
            return int(action_id)
        # The vertical flip mapping is its own inverse.
        return map_action_vertical_flip(action_id)

    def real_to_canonical_mask(self, mask):
        if not self.use_vertical_flip:
            return np.asarray(mask, dtype=bool)
        return map_action_mask_vertical_flip(mask)

    def real_to_canonical_state(self, state):
        if not self.use_vertical_flip:
            return np.asarray(state, dtype=np.float32)
        return map_state_vertical_flip(state)

    def reset(self, opponent_type=0):
        req = quoridor_pb2.ResetRequest(opponent_type=opponent_type)
        res = self.stub.Reset(req)

        state = self.real_to_canonical_state(np.array(res.state_tensor, dtype=np.float32))
        mask = self.real_to_canonical_mask(np.array(res.action_mask, dtype=bool))
        return state, mask
    # # Ejemplo si en tu .proto el campo se llama 'board_state' u 'observation':
    # def reset(self, opponent_type=0):
    #     request = quoridor_pb2.ResetRequest(opponent_type=opponent_type)
    #     response = self.stub.Reset(request)
    #     print("Respuesta gRPC de Reset:", response)
    #     # Reemplaza 'board_state' por el nombre real de tu .proto
    #     return response.board_state, response.mask
    def step(self, action_id):
        real_action_id = self.canonical_to_real_action(action_id)
        req = quoridor_pb2.StepRequest(action_id=real_action_id)
        res = self.stub.Step(req,timeout=5.0)

        state = self.real_to_canonical_state(np.array(res.state_tensor, dtype=np.float32))
        mask = self.real_to_canonical_mask(np.array(res.action_mask, dtype=bool))
        reward = float(res.reward)
        done = bool(res.done)
        
        return state, mask, reward, done, res.winner