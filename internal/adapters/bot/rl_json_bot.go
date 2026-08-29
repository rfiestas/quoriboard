package bot

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"

	"quoridor/assets"
	"quoridor/internal/domain"
)

type rlJSONLayer struct {
	Name        string    `json:"name"`
	Activation  string    `json:"activation"`
	InFeatures  int       `json:"in_features"`
	OutFeatures int       `json:"out_features"`
	Weight      []float32 `json:"weight"`
	Bias        []float32 `json:"bias"`
}

type rlJSONPolicy struct {
	Layers []rlJSONLayer `json:"layers"`
}

type RLJSONBot struct {
	layers []rlJSONLayer
}

func NewRLJSONBot() (*RLJSONBot, error) {
	var policy rlJSONPolicy
	if err := json.Unmarshal(assets.RLJSONWeights, &policy); err != nil {
		return nil, fmt.Errorf("decode embedded V8 policy: %w", err)
	}
	if err := validateRLJSONPolicy(policy.Layers); err != nil {
		return nil, err
	}
	return &RLJSONBot{layers: policy.Layers}, nil
}

func validateRLJSONPolicy(layers []rlJSONLayer) error {
	if len(layers) == 0 {
		return fmt.Errorf("embedded V8 policy has no layers")
	}
	inputSize := ObservationSize
	for index, layer := range layers {
		if layer.InFeatures != inputSize || layer.OutFeatures <= 0 {
			return fmt.Errorf("invalid V8 policy layer %d (%s): %d -> %d, expected input %d", index, layer.Name, layer.InFeatures, layer.OutFeatures, inputSize)
		}
		if len(layer.Weight) != layer.InFeatures*layer.OutFeatures || len(layer.Bias) != layer.OutFeatures {
			return fmt.Errorf("invalid V8 policy parameters in layer %d (%s)", index, layer.Name)
		}
		switch layer.Activation {
		case "none", "tanh", "relu":
		default:
			return fmt.Errorf("unsupported V8 policy activation %q in layer %d (%s)", layer.Activation, index, layer.Name)
		}
		inputSize = layer.OutFeatures
	}
	if inputSize != ActionSpaceSize {
		return fmt.Errorf("V8 policy output size = %d, want %d", inputSize, ActionSpaceSize)
	}
	return nil
}

func (b *RLJSONBot) GetAction(game *domain.Game, player *domain.Player) domain.PlayerAction {
	turns := game.Turns
	if turns > maxTurns {
		turns = maxTurns
	}
	logits := b.forward(BuildObservation(game, player, turns))
	mask := BuildActionMask(game, player)
	bestIndex := -1
	bestScore := float32(-math.MaxFloat32)
	for index, score := range logits {
		if mask[index] && score > bestScore {
			bestIndex, bestScore = index, score
		}
	}
	if bestIndex < 0 {
		return domain.PlayerAction{}
	}
	return DecodeAction(game, player, bestIndex)
}

// func (b *RLJSONBot) forward(input []float32) []float32 {
// 	values := input
// 	for _, layer := range b.layers {
// 		output := make([]float32, layer.OutFeatures)
// 		for outputIndex := range output {
// 			sum := layer.Bias[outputIndex]
// 			weightOffset := outputIndex * layer.InFeatures
// 			for inputIndex, value := range values {
// 				sum += layer.Weight[weightOffset+inputIndex] * value
// 			}
// 			switch layer.Activation {
// 			case "tanh":
// 				sum = float32(math.Tanh(float64(sum)))
// 			case "relu":
// 				if sum < 0 {
// 					sum = 0
// 				}
// 			}
// 			output[outputIndex] = sum
// 		}
// 		values = output
// 	}
// 	return values
// }

func (b *RLJSONBot) forward(input []float32) []float32 {
	values := input
	for _, layer := range b.layers {
		output := make([]float32, layer.OutFeatures)
		for outputIndex := range output {
			sum := layer.Bias[outputIndex]
			weightOffset := outputIndex * layer.InFeatures
			for inputIndex, value := range values {
				// PyTorch exporta con pesos en orden (out_features, in_features) aplanados por filas.
				// Por tanto, el peso para un outputIndex dado y un inputIndex está en (outputIndex * in_features) + inputIndex.
				sum += layer.Weight[weightOffset+inputIndex] * value
			}
			switch layer.Activation {
			case "tanh":
				sum = float32(math.Tanh(float64(sum)))
			case "relu":
				if sum < 0 {
					sum = 0
				}
			}
			output[outputIndex] = sum
		}
		values = output
	}
	return values
}
