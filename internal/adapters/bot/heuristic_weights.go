package bot

// Hand-crafted heuristic weights for different bot personalities. These weights influence the bot's decision-making process in the game.

// Bot_Aggressive prioritizes penalizing the rival over its own path.
// High positive RivalDistanceWeight combined with negative MyDistanceWeight makes it act aggressively.
func Bot_Aggressive() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -10.0,
			RivalDistanceWeight: 9.98,
			WallReserveWeight:   0.74,
			CentralityWeight:    2.85,
			JumpConcededPenalty: -30.94,
			FunnelingBonus:      14.33,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    2.22,
			RivalDistanceWeight: 20.0,
			WallReserveWeight:   2.6,
			CentralityWeight:    1.47,
			JumpConcededPenalty: -49.26,
			FunnelingBonus:      25.49,
		},
	}
}

// Bot_Defensive prioritizes its own progression to the goal line while conserving walls.
// High positive MyDistanceWeight keeps it focused on its own path, minimizing risky offensive moves.
func Bot_Defensive() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    15.0,
			RivalDistanceWeight: -2.0,
			WallReserveWeight:   10.0,
			CentralityWeight:    1.5,
			JumpConcededPenalty: -10.0,
			FunnelingBonus:      5.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    25.0,
			RivalDistanceWeight: -5.0,
			WallReserveWeight:   15.0,
			CentralityWeight:    0.5,
			JumpConcededPenalty: -20.0,
			FunnelingBonus:      2.0,
		},
	}
}

// Bot_Balanced keeps an even weight distribution between advancing and obstructing the opponent.
// Balanced values for MyDistanceWeight and RivalDistanceWeight allow it to adapt dynamically to game state.
func Bot_Balanced() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    5.0,
			RivalDistanceWeight: 5.0,
			WallReserveWeight:   3.0,
			CentralityWeight:    3.0,
			JumpConcededPenalty: -25.0,
			FunnelingBonus:      10.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    10.0,
			RivalDistanceWeight: 10.0,
			WallReserveWeight:   5.0,
			CentralityWeight:    2.0,
			JumpConcededPenalty: -35.0,
			FunnelingBonus:      15.0,
		},
	}
}

// Bot_Chaotic uses unconventional weight distributions to create unpredictable positional dynamics.
// Zero or inverted distance weights force the decision engine to prioritize secondary factors like centrality and funneling.
func Bot_Chaotic() BotConfig {
	return BotConfig{
		PanicThreshold: 3,
		NormalWeights: BotWeights{
			MyDistanceWeight:    0.0,
			RivalDistanceWeight: 0.0,
			WallReserveWeight:   20.0,
			CentralityWeight:    15.0,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      30.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -15.0,
			RivalDistanceWeight: -15.0,
			WallReserveWeight:   0.0,
			CentralityWeight:    0.0,
			JumpConcededPenalty: 0.0,
			FunnelingBonus:      50.0,
		},
	}
}

// end hand-crafted heuristic weights for different bot personalities. These weights influence the bot's decision-making process in the game.

// Best 5 bots trained vs heuristic. First is the best one, last is the worst one. All of them are better than heuristic.

// E6G5M74
func Bot_E6G5M74() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.89,
			RivalDistanceWeight: 9.76,
			WallReserveWeight:   -4.22,
			CentralityWeight:    7.96,
			JumpConcededPenalty: -28.82,
			FunnelingBonus:      15.73,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    1.91,
			RivalDistanceWeight: 22.28,
			WallReserveWeight:   5.05,
			CentralityWeight:    6.26,
			JumpConcededPenalty: -49.93,
			FunnelingBonus:      19.8,
		},
	}
}

// E6G5M81
func Bot_E6G5M81() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -12.55,
			RivalDistanceWeight: 11.48,
			WallReserveWeight:   -4.13,
			CentralityWeight:    9.32,
			JumpConcededPenalty: -28.82,
			FunnelingBonus:      15.51,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    2.49,
			RivalDistanceWeight: 23.69,
			WallReserveWeight:   5.2,
			CentralityWeight:    4.21,
			JumpConcededPenalty: -49.48,
			FunnelingBonus:      20.31,
		},
	}
}

// E6G5M41
func Bot_E6G5M41() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -12.55,
			RivalDistanceWeight: 11.48,
			WallReserveWeight:   -4.22,
			CentralityWeight:    9.32,
			JumpConcededPenalty: -28.82,
			FunnelingBonus:      15.68,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    3.33,
			RivalDistanceWeight: 23.41,
			WallReserveWeight:   5.2,
			CentralityWeight:    5.77,
			JumpConcededPenalty: -49.48,
			FunnelingBonus:      20.31,
		},
	}
}

// E6G5M39
func Bot_E6G5M39() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.89,
			RivalDistanceWeight: 9.87,
			WallReserveWeight:   -2.99,
			CentralityWeight:    9.5,
			JumpConcededPenalty: -29.67,
			FunnelingBonus:      16.13,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    1.93,
			RivalDistanceWeight: 22.35,
			WallReserveWeight:   5.2,
			CentralityWeight:    7.29,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      20.71,
		},
	}
}

// E6G5M86
func Bot_E6G5M86() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.89,
			RivalDistanceWeight: 8.91,
			WallReserveWeight:   -4.73,
			CentralityWeight:    8.66,
			JumpConcededPenalty: -28.82,
			FunnelingBonus:      16.13,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    0.97,
			RivalDistanceWeight: 22.35,
			WallReserveWeight:   5.55,
			CentralityWeight:    5.66,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      20.85,
		},
	}
}

// Best 5 rivals trained vs heuristic. First is the best one, last is the worst one.

// E1G3M41
func Bot_E1G3M41() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -9.39,
			RivalDistanceWeight: 6.13,
			WallReserveWeight:   7.37,
			CentralityWeight:    6.17,
			JumpConcededPenalty: -29.05,
			FunnelingBonus:      7.6,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    2.22,
			RivalDistanceWeight: 17.48,
			WallReserveWeight:   4.0,
			CentralityWeight:    3.41,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      25.49,
		},
	}
}

// E2G3M99
func Bot_E2G3M99() BotConfig {
	return BotConfig{
		PanicThreshold: 0,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.92,
			RivalDistanceWeight: 9.8,
			WallReserveWeight:   -1.08,
			CentralityWeight:    7.2,
			JumpConcededPenalty: -29.54,
			FunnelingBonus:      21.29,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -6.04,
			RivalDistanceWeight: 26.31,
			WallReserveWeight:   2.53,
			CentralityWeight:    4.02,
			JumpConcededPenalty: -48.71,
			FunnelingBonus:      17.44,
		},
	}
}

// E1G1M78
func Bot_E1G1M78() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -12.58,
			RivalDistanceWeight: 13.91,
			WallReserveWeight:   -0.91,
			CentralityWeight:    5.27,
			JumpConcededPenalty: -26.98,
			FunnelingBonus:      18.33,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    2.22,
			RivalDistanceWeight: 20.0,
			WallReserveWeight:   3.02,
			CentralityWeight:    3.37,
			JumpConcededPenalty: -49.59,
			FunnelingBonus:      22.91,
		},
	}
}

// E2G4M40
func Bot_E2G4M40() BotConfig {
	return BotConfig{
		PanicThreshold: 0,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -14.83,
			RivalDistanceWeight: 7.99,
			WallReserveWeight:   -1.08,
			CentralityWeight:    7.2,
			JumpConcededPenalty: -29.54,
			FunnelingBonus:      21.45,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -6.04,
			RivalDistanceWeight: 26.31,
			WallReserveWeight:   3.44,
			CentralityWeight:    5.2,
			JumpConcededPenalty: -48.71,
			FunnelingBonus:      17.44,
		},
	}
}

// E5G8M26
func Bot_E5G8M26() BotConfig {
	return BotConfig{
		PanicThreshold: -2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -13.96,
			RivalDistanceWeight: 4.73,
			WallReserveWeight:   6.02,
			CentralityWeight:    12.88,
			JumpConcededPenalty: -28.42,
			FunnelingBonus:      22.62,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -3.45,
			RivalDistanceWeight: 24.54,
			WallReserveWeight:   8.64,
			CentralityWeight:    6.94,
			JumpConcededPenalty: -48.15,
			FunnelingBonus:      21.35,
		},
	}
}

// Best 5 bots trained vs minimax3. First is the best one, last is the worst one. All of them are better than minimax3.

// E4G3M81
func Bot_E4G3M81() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   1.59,
			CentralityWeight:    2.23,
			JumpConcededPenalty: -33.32,
			FunnelingBonus:      20.59,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 19.85,
			WallReserveWeight:   -4.71,
			CentralityWeight:    -6.79,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      24.77,
		},
	}
}

// E4G4M55
func Bot_E4G4M55() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   3.9,
			CentralityWeight:    2.23,
			JumpConcededPenalty: -33.32,
			FunnelingBonus:      21.16,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.55,
			RivalDistanceWeight: 22.49,
			WallReserveWeight:   -4.71,
			CentralityWeight:    -6.84,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      25.8,
		},
	}
}

// E4G4M45
func Bot_E4G4M45() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   1.59,
			CentralityWeight:    2.23,
			JumpConcededPenalty: -35.16,
			FunnelingBonus:      20.59,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 19.85,
			WallReserveWeight:   -2.87,
			CentralityWeight:    -6.79,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      24.77,
		},
	}
}

// E4G4M78
func Bot_E4G4M78() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   1.59,
			CentralityWeight:    2.23,
			JumpConcededPenalty: -33.32,
			FunnelingBonus:      20.59,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 19.85,
			WallReserveWeight:   -4.71,
			CentralityWeight:    -6.79,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      23.9,
		},
	}
}

// E4G4M79
func Bot_E4G4M79() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -12.43,
			RivalDistanceWeight: 7.32,
			WallReserveWeight:   1.35,
			CentralityWeight:    0.0,
			JumpConcededPenalty: -33.99,
			FunnelingBonus:      20.15,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -7.64,
			RivalDistanceWeight: 21.76,
			WallReserveWeight:   -4.69,
			CentralityWeight:    -4.82,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      22.95,
		},
	}
}

// Best rivals promoted from the minimax3 training. First is the best one, last is the worst one. All of them are better than minimax3.

// E4G10M52
func Bot_E4G10M52() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   -0.37,
			CentralityWeight:    2.23,
			JumpConcededPenalty: -33.32,
			FunnelingBonus:      20.59,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 19.24,
			WallReserveWeight:   -4.71,
			CentralityWeight:    -8.25,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      23.05,
		},
	}
}

// E5G4M77
func Bot_E5G4M77() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -13.25,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   1.59,
			CentralityWeight:    2.01,
			JumpConcededPenalty: -35.16,
			FunnelingBonus:      21.88,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 21.0,
			WallReserveWeight:   -2.87,
			CentralityWeight:    -6.79,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      23.8,
		},
	}
}

// E5G4M80
func Bot_E5G4M80() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -12.93,
			RivalDistanceWeight: 8.61,
			WallReserveWeight:   3.31,
			CentralityWeight:    1.13,
			JumpConcededPenalty: -30.65,
			FunnelingBonus:      18.58,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -4.57,
			RivalDistanceWeight: 21.28,
			WallReserveWeight:   -1.91,
			CentralityWeight:    -4.35,
			JumpConcededPenalty: -48.49,
			FunnelingBonus:      21.21,
		},
	}
}

// E6G4M92
func Bot_E6G4M92() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   1.16,
			CentralityWeight:    2.52,
			JumpConcededPenalty: -33.32,
			FunnelingBonus:      21.71,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.55,
			RivalDistanceWeight: 21.94,
			WallReserveWeight:   -4.71,
			CentralityWeight:    -6.84,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      25.8,
		},
	}
}

// E6G4M33
func Bot_E6G4M33() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -11.44,
			RivalDistanceWeight: 7.97,
			WallReserveWeight:   1.59,
			CentralityWeight:    1.83,
			JumpConcededPenalty: -35.16,
			FunnelingBonus:      20.59,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 19.17,
			WallReserveWeight:   -2.87,
			CentralityWeight:    -6.79,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      24.77,
		},
	}
}

// Best players promoted from the minimax4 training. First is the best one, last is the worst one. All of them are better than minimax4.

// E6G5M81
func Bot_E6G5M81_b() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -14.3,
			RivalDistanceWeight: 9.44,
			WallReserveWeight:   -4.48,
			CentralityWeight:    6.79,
			JumpConcededPenalty: -30.23,
			FunnelingBonus:      18.38,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    1.91,
			RivalDistanceWeight: 20.61,
			WallReserveWeight:   4.17,
			CentralityWeight:    9.53,
			JumpConcededPenalty: -48.35,
			FunnelingBonus:      22.28,
		},
	}
}

// E6G3M75
func Bot_E6G3M75() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -15.43,
			RivalDistanceWeight: 12.37,
			WallReserveWeight:   -5.69,
			CentralityWeight:    8.33,
			JumpConcededPenalty: -29.22,
			FunnelingBonus:      18.38,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    3.2,
			RivalDistanceWeight: 20.36,
			WallReserveWeight:   5.88,
			CentralityWeight:    8.41,
			JumpConcededPenalty: -48.98,
			FunnelingBonus:      22.28,
		},
	}
}

// E6G4M97
func Bot_E6G4M97() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -15.43,
			RivalDistanceWeight: 12.37,
			WallReserveWeight:   -5.69,
			CentralityWeight:    8.33,
			JumpConcededPenalty: -29.22,
			FunnelingBonus:      18.56,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    3.2,
			RivalDistanceWeight: 20.36,
			WallReserveWeight:   6.01,
			CentralityWeight:    8.41,
			JumpConcededPenalty: -49.63,
			FunnelingBonus:      20.69,
		},
	}
}

// E6G4M71
func Bot_E6G4M71() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -15.43,
			RivalDistanceWeight: 12.37,
			WallReserveWeight:   -5.69,
			CentralityWeight:    8.33,
			JumpConcededPenalty: -29.22,
			FunnelingBonus:      18.38,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    3.2,
			RivalDistanceWeight: 20.36,
			WallReserveWeight:   5.88,
			CentralityWeight:    8.41,
			JumpConcededPenalty: -48.98,
			FunnelingBonus:      22.28,
		},
	}
}

// E6G4M29
func Bot_E6G4M29() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -15.43,
			RivalDistanceWeight: 10.77,
			WallReserveWeight:   -6.29,
			CentralityWeight:    8.38,
			JumpConcededPenalty: -27.58,
			FunnelingBonus:      18.38,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    1.91,
			RivalDistanceWeight: 19.16,
			WallReserveWeight:   5.88,
			CentralityWeight:    9.69,
			JumpConcededPenalty: -48.98,
			FunnelingBonus:      22.28,
		},
	}
}

// Best 2 rivals promoted from the minimax4 training. First is the best one, last is the worst one. All of them are better than minimax4.

// E1G2M90
func Bot_E1G2M90() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -12.54,
			RivalDistanceWeight: 10.44,
			WallReserveWeight:   -0.99,
			CentralityWeight:    3.59,
			JumpConcededPenalty: -33.32,
			FunnelingBonus:      21.72,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -10.13,
			RivalDistanceWeight: 19.85,
			WallReserveWeight:   -4.71,
			CentralityWeight:    -5.01,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      25.4,
		},
	}
}

// E1G3M77
func Bot_E1G3M77() BotConfig {
	return BotConfig{
		PanicThreshold: 0,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -15.4,
			RivalDistanceWeight: 7.35,
			WallReserveWeight:   1.12,
			CentralityWeight:    5.29,
			JumpConcededPenalty: -35.0,
			FunnelingBonus:      23.72,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -9.44,
			RivalDistanceWeight: 20.99,
			WallReserveWeight:   -2.65,
			CentralityWeight:    -5.01,
			JumpConcededPenalty: -48.53,
			FunnelingBonus:      26.9,
		},
	}
}
