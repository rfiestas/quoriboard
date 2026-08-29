package bot

import "testing"

func TestNewRLJSONBotLoadsV8Contract(t *testing.T) {
	bot, err := NewRLJSONBot()
	if err != nil {
		t.Fatal(err)
	}
	if len(bot.layers) != 3 {
		t.Fatalf("layer count = %d, want 3", len(bot.layers))
	}
	if bot.layers[0].InFeatures != ObservationSize {
		t.Fatalf("input size = %d, want %d", bot.layers[0].InFeatures, ObservationSize)
	}
	if bot.layers[len(bot.layers)-1].OutFeatures != ActionSpaceSize {
		t.Fatalf("output size = %d, want %d", bot.layers[len(bot.layers)-1].OutFeatures, ActionSpaceSize)
	}
}
