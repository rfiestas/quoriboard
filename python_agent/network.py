import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.distributions import Categorical

class ActorCritic(nn.Module):
    def __init__(self, state_dim=294, action_dim=140):
        super(ActorCritic, self).__init__()
        
        # Tronco común de procesamiento de características
        self.feature_extractor = nn.Sequential(
            nn.Linear(state_dim, 256),
            nn.ReLU(),
            nn.Linear(256, 256),
            nn.ReLU()
        )
        
        # Cabeza del Actor (Política de decisiones)
        self.actor = nn.Sequential(
            nn.Linear(256, 128),
            nn.ReLU(),
            nn.Linear(128, action_dim)
        )
        
        # Cabeza del Crítico (Estimador del valor del tablero)
        self.critic = nn.Sequential(
            nn.Linear(256, 128),
            nn.ReLU(),
            nn.Linear(128, 1)
        )

    def forward(self, state, action_mask):
        features = self.feature_extractor(state)
        logits = self.actor(features)
        
        # APLICAR MÁSCARA: Asigna -1e8 a las acciones ilegales para que su probabilidad sea 0
        masked_logits = torch.where(action_mask, logits, torch.tensor(-1e8, device=logits.device))
        
        probs = F.softmax(masked_logits, dim=-1)
        value = self.critic(features)
        
        return probs, value

    def get_action(self, state, action_mask):
        state_t = torch.FloatTensor(state).unsqueeze(0)
        mask_t = torch.BoolTensor(action_mask).unsqueeze(0)
        
        probs, value = self.forward(state_t, mask_t)
        dist = Categorical(probs)
        action = dist.sample()
        
        return action.item(), dist.log_prob(action), value.squeeze(0)