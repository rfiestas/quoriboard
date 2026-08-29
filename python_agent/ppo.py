import torch
import torch.nn as nn
import torch.optim as optim
from torch.distributions import Categorical
import numpy as np

class QuoridorMLP(nn.Module):
    def __init__(self, state_dim=491, action_dim=140):
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(state_dim, 256),
            nn.ReLU(),
            nn.Linear(256, 256),
            nn.ReLU(),
            nn.Linear(256, 128),
            nn.ReLU()
        )
        self.actor = nn.Linear(128, action_dim)
        self.critic = nn.Linear(128, 1)

    def forward(self, x):
        if x.dim() == 1:
            x = x.unsqueeze(0)
        feats = self.net(x)
        logits = self.actor(feats)
        value = self.critic(feats)
        return logits, value

    def get_action(self, state, mask):
        device = next(self.parameters()).device
        
        state_t = torch.FloatTensor(state).to(device) if not isinstance(state, torch.Tensor) else state.to(device)
        mask_t = torch.BoolTensor(mask).to(device) if not isinstance(mask, torch.Tensor) else mask.to(device)

        with torch.no_grad():
            logits, value = self.forward(state_t)
            logits = logits.squeeze(0)
            
            # Enmascaramiento seguro evitando NaNs
            logits = logits.masked_fill(~mask_t, -1e8)
            
            dist = Categorical(logits=logits)
            action = dist.sample()
            
            return action.item(), dist.log_prob(action), value.squeeze().item()


class PPOAgent:
    def __init__(self, state_dim=491, action_dim=140, lr=3e-4, gamma=0.99, clip_eps=0.2, ppo_epochs=8, entropy_coef=0.04):
        self.device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
        self.gamma = gamma
        self.clip_eps = clip_eps
        self.ppo_epochs = ppo_epochs
        self.entropy_coef = entropy_coef
        
        self.policy = QuoridorMLP(state_dim=state_dim, action_dim=action_dim).to(self.device)
        self.optimizer = optim.Adam(self.policy.parameters(), lr=lr)

        self.policy_old = QuoridorMLP(state_dim=state_dim, action_dim=action_dim).to(self.device)
        self.policy_old.load_state_dict(self.policy.state_dict())

        self.MseLoss = nn.MSELoss()

    def update(self, memory):
        if isinstance(memory, dict):
            m_rewards = memory.get('rewards', [])
            m_terminals = memory.get('dones', [])
            m_states = memory.get('states', [])
            m_actions = memory.get('actions', [])
            m_logprobs = memory.get('logprobs', [])
            m_masks = memory.get('masks', [])
        else:
            m_rewards = memory.rewards
            m_terminals = getattr(memory, 'dones', [])
            m_states = memory.states
            m_actions = memory.actions
            m_logprobs = memory.logprobs
            m_masks = memory.masks

        # Guardabarros: Abortar si no hay experiencias guardadas
        if len(m_states) == 0:
            return

        # 1. Retornos Monte Carlo
        rewards = []
        discounted_reward = 0
        for reward, is_terminal in zip(reversed(m_rewards), reversed(m_terminals)):
            if is_terminal:
                discounted_reward = 0
            discounted_reward = reward + (self.gamma * discounted_reward)
            rewards.insert(0, discounted_reward)
            
        rewards = torch.tensor(rewards, dtype=torch.float32).to(self.device)

        # 2. Conversión de datos a tensores
        old_states = torch.FloatTensor(np.array(m_states)).to(self.device)
        old_actions = torch.LongTensor(m_actions).to(self.device)
        old_logprobs = torch.FloatTensor(m_logprobs).to(self.device)
        old_masks = torch.BoolTensor(np.array(m_masks)).to(self.device)

        # 3. Optimización PPO
        for _ in range(self.ppo_epochs):
            logits, state_values = self.policy(old_states)
            state_values = state_values.squeeze(-1)
            
            logits = logits.masked_fill(~old_masks, -1e8)
            dist = Categorical(logits=logits)
            
            logprobs = dist.log_prob(old_actions)
            dist_entropy = dist.entropy()

            ratios = torch.exp(logprobs - old_logprobs.detach())
            advantages = rewards - state_values.detach()
            
            # Normalizar únicamente las ventajas
            if len(advantages) > 1:
                advantages = (advantages - advantages.mean()) / (advantages.std() + 1e-8)

            surr1 = ratios * advantages
            surr2 = torch.clamp(ratios, 1 - self.clip_eps, 1 + self.clip_eps) * advantages

            loss = -torch.min(surr1, surr2) + 0.5 * self.MseLoss(state_values, rewards) - self.entropy_coef * dist_entropy

            self.optimizer.zero_grad()
            loss.mean().backward()
            self.optimizer.step()
            
        self.policy_old.load_state_dict(self.policy.state_dict())

        if isinstance(memory, dict):
            for key in memory:
                if isinstance(memory[key], list):
                    memory[key].clear()
        else:
            memory.clear()