package spp

import "strings"

type ModelInfo struct {
	Codename          string
	Product           string
	FastPairID        string
	Protocol          string
	Tier              string
	BatteryCaseSource string
	Features          []string
	Aliases           []string
}

var knownModels = []ModelInfo{
	{Codename: "EarOne", Product: "Nothing ear (1)", FastPairID: "31D53D", Protocol: "EarOneProtocol", Tier: "A", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"ear one", "ear1", "ear (1)", "nothing ear (1)"}},
	{Codename: "EarTwo", Product: "Ear (2)", FastPairID: "DEE8C0", Protocol: "EarTwoProtocol", Tier: "B", BatteryCaseSource: "case", Features: []string{"anc", "eq", "dual", "lhdc", "mimi", "3d"}, Aliases: []string{"ear two", "ear2", "ear (2)"}},
	{Codename: "EarTwos", Product: "Nothing Ear (2024)", FastPairID: "FEB1C7", Protocol: "EarTwosProtocol", Tier: "B+", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "dual", "advance_eq", "mimi", "3d"}, Aliases: []string{"ear twos", "nothing ear", "ear 2024"}},
	{Codename: "EarThree", Product: "Ear (3)", FastPairID: "C1EBFD", Protocol: "EarTwosProtocol", Tier: "B+", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "dual", "advance_eq", "bass"}, Aliases: []string{"ear three", "ear3", "ear (3)", "nothing ear (3)", "feraligatr"}},
	{Codename: "EarStick", Product: "Ear (stick)", FastPairID: "1016DD", Protocol: "EarStickProtocol", Tier: "B-", BatteryCaseSource: "case", Features: []string{"eq", "advance_eq"}, Aliases: []string{"ear stick", "ear (stick)"}},
	{Codename: "EarColor", Product: "Nothing Ear (a)", FastPairID: "5E3FBC", Protocol: "EarColorProtocol", Tier: "B", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "dual", "advance_eq", "3d"}, Aliases: []string{"ear color", "ear a", "ear (a)", "nothing ear (a)"}},
	{Codename: "Flaffy", Product: "Nothing ear (open)", FastPairID: "FC3AAF", Protocol: "FlaffyProtocol", Tier: "B", BatteryCaseSource: "case", Features: []string{"eq", "dual", "advance_eq", "3d"}, Aliases: []string{"ear open", "ear (open)", "nothing ear (open)", "cc3444"}},
	{Codename: "Elekid", Product: "Nothing Headphone (1)", FastPairID: "2D6FDA", Protocol: "ElekidProtocol", Tier: "C", BatteryCaseSource: "stereo", Features: []string{"anc", "eq", "spatial", "dual", "bass"}, Aliases: []string{"headphone 1", "headphone (1)", "nothing headphone (1)"}},
	{Codename: "Forretress", Product: "Headphone Pro", FastPairID: "73C9EB", Protocol: "ElekidProtocol", Tier: "C+", BatteryCaseSource: "stereo", Features: []string{"anc", "eq", "spatial", "headtrack", "le_audio", "system_audio", "magic_button", "bass"}, Aliases: []string{"forretress", "headphone pro", "24211"}},
	{Codename: "Crobat", Product: "CMF Neckband Pro", FastPairID: "AE35FD", Protocol: "CrobatProtocol", Tier: "C", BatteryCaseSource: "stereo", Features: []string{"anc", "eq", "spatial"}, Aliases: []string{"neckband pro", "cmf neckband pro"}},
	{Codename: "Corsola", Product: "CMF Buds Pro", FastPairID: "ADD2C4", Protocol: "CorsolaProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial"}, Aliases: []string{"buds pro", "cmf buds pro"}},
	{Codename: "Donphan", Product: "CMF Buds", FastPairID: "150A27", Protocol: "DonphanProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"eq"}, Aliases: []string{"cmf buds", "buds"}},
	{Codename: "Espeon", Product: "CMF Buds Pro 2", FastPairID: "F29566", Protocol: "EspeonProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "bass"}, Aliases: []string{"buds pro 2", "cmf buds pro 2"}},
	{Codename: "Girafarig", Product: "24232", FastPairID: "19EF24", Protocol: "GirafarigProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "bass"}, Aliases: []string{"24232"}},
	{Codename: "Gligar", Product: "24241", FastPairID: "4AEB6E", Protocol: "GligarProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "bass"}, Aliases: []string{"24241"}},
	{Codename: "Hitmontop", Product: "24272", FastPairID: "404D6D", Protocol: "unpublished", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"24272"}},
	{Codename: "Hoothoot", Product: "24283", FastPairID: "70F8E3", Protocol: "unpublished", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"24283"}},
	{Codename: "Heracross", Product: "24253", FastPairID: "2F45F5", Protocol: "unpublished", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"24253"}},
}

func normalizeModelKey(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", " "))
}

func ResolveModelInfo(value string) (ModelInfo, bool) {
	key := normalizeModelKey(value)
	if key == "" {
		return ModelInfo{}, false
	}

	for _, model := range knownModels {
		if normalizeModelKey(model.Codename) == key ||
			normalizeModelKey(model.Product) == key ||
			normalizeModelKey(model.FastPairID) == key {
			return model, true
		}

		for _, alias := range model.Aliases {
			if normalizeModelKey(alias) == key {
				return model, true
			}
		}
	}

	return ModelInfo{}, false
}

func ResolveModelFromBluetooth(values ...string) (ModelInfo, string, bool) {
	for _, value := range values {
		if model, ok := ResolveModelInfo(value); ok {
			return model, strings.TrimSpace(value), true
		}
	}

	haystack := strings.ToUpper(strings.Join(values, "\n"))
	for _, model := range knownModels {
		fastPairID := strings.ToUpper(strings.TrimSpace(model.FastPairID))
		if fastPairID != "" && strings.Contains(haystack, fastPairID) {
			return model, "fast_pair_id:" + fastPairID, true
		}
	}

	normalizedHaystack := normalizeModelKey(strings.Join(values, "\n"))
	for _, model := range knownModels {
		for _, candidate := range append([]string{model.Product, model.Codename}, model.Aliases...) {
			key := normalizeModelKey(candidate)
			if len(key) >= 6 && strings.Contains(normalizedHaystack, key) {
				return model, strings.TrimSpace(candidate), true
			}
		}
	}

	return ModelInfo{}, "", false
}

func ModelSupportsFeature(model ModelInfo, feature string) bool {
	if model.Codename == "" {
		return true
	}
	if feature == "lag" {
		return true
	}

	for _, item := range model.Features {
		if item == feature {
			return true
		}
	}

	return false
}

func KnownModels() []ModelInfo {
	out := make([]ModelInfo, len(knownModels))
	copy(out, knownModels)
	return out
}

func DefaultModel() ModelInfo { return ModelInfo{BatteryCaseSource: "case"} }

