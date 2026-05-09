package stego

type FeatureInfo struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	MinNormal float64 `json:"min_normal"`
	MaxNormal float64 `json:"max_normal"`
	IsAnomaly bool    `json:"is_anomaly"`
}

type AnalyzeResponse struct {
	StegoProb float64       `json:"stego_prob"`
	Result    string        `json:"result"`
	Features  []FeatureInfo `json:"features"`
}

type PredictionResponse struct {
	StegoProb float64 `json:"stego_prob"`
}

// Normal ranges thresholds for steganalysis features
// Based on typical value distributions for clean images
var featureNormalRanges = []struct {
	Name string
	Min  float64
	Max  float64
}{
	{"LSB Transition", 0.45, 0.55},
	{"Bit Run 00", 0.20, 0.35},
	{"Bit Run 01", 0.20, 0.35},
	{"Bit Run 10", 0.20, 0.35},
	{"Bit Run 11", 0.20, 0.35},
	{"Neighbor Diff", 0.45, 0.55},
	{"Chi-Square", 0.0, 3.0},
	{"Entropy LSB", 0.95, 1.0},
	{"R (Regular)", 0.40, 0.60},
	{"S (Singular)", 0.40, 0.60},
	{"Rm (Regular flipped)", 0.40, 0.60},
	{"Sm (Singular flipped)", 0.40, 0.60},
	{"Rm-R", -0.10, 0.10},
	{"Sm-S", -0.10, 0.10},
}

func GetFeaturesWithAnalysisSlice(featureValues []float64) []FeatureInfo {
	features := make([]FeatureInfo, 14)
	for i := 0; i < 14 && i < len(featureValues); i++ {
		features[i] = FeatureInfo{
			Name:      featureNormalRanges[i].Name,
			Value:     featureValues[i],
			MinNormal: featureNormalRanges[i].Min,
			MaxNormal: featureNormalRanges[i].Max,
			IsAnomaly: featureValues[i] < featureNormalRanges[i].Min || featureValues[i] > featureNormalRanges[i].Max,
		}
	}
	return features
}
