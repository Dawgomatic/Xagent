package llmcheck

import (
	"fmt"
	"sort"
	"strings"
)

type Recommendation struct {
	Model      ModelVariant   `json:"model"`
	Score      ScoreBreakdown `json:"score"`
	Fits       bool           `json:"fits"`
	Installed  bool           `json:"installed"`
}

type AnalysisResult struct {
	Hardware       *HardwareProfile  `json:"hardware"`
	Ollama         OllamaVersion     `json:"ollama"`
	Compatible     []Recommendation  `json:"compatible"`
	Marginal       []Recommendation  `json:"marginal"`
	Incompatible   []Recommendation  `json:"incompatible"`
	TopPick        *Recommendation   `json:"top_pick,omitempty"`
}

type AnalysisOptions struct {
	UseCase   string
	TargetCtx int
}

func Analyze(opts AnalysisOptions) (*AnalysisResult, error) {
	hw, err := DetectHardware()
	if err != nil {
		return nil, fmt.Errorf("hardware detection failed: %w", err)
	}

	return AnalyzeWithHardware(hw, opts)
}

func AnalyzeWithHardware(hw *HardwareProfile, opts AnalysisOptions) (*AnalysisResult, error) {
	if opts.UseCase == "" {
		opts.UseCase = "general"
	}
	if opts.TargetCtx <= 0 {
		opts.TargetCtx = 8192
	}

	ollama := NewOllamaClient()
	ollamaStatus := ollama.CheckAvailability()

	var installedNames map[string]bool
	if ollamaStatus.Available {
		installedNames = make(map[string]bool)
		if models, err := ollama.ListModels(); err == nil {
			for _, m := range models {
				installedNames[m.Name] = true
				short := strings.Split(m.Name, ":")[0]
				installedNames[short] = true
			}
		}
	}

	result := &AnalysisResult{
		Hardware: hw,
		Ollama:   ollamaStatus,
	}

	effectiveMem := hw.EffectiveMemoryGB()

	for i := range CuratedCatalog {
		m := &CuratedCatalog[i]
		score := ScoreModel(m, hw, opts.UseCase, opts.TargetCtx)
		fits := FitsInMemory(m, hw)

		installed := false
		if installedNames != nil {
			installed = installedNames[m.OllamaTag] || installedNames[m.Name]
		}

		rec := Recommendation{
			Model:     *m,
			Score:     score,
			Fits:      fits,
			Installed: installed,
		}

		modelSize := m.EffectiveSize()
		usage := 0.0
		if effectiveMem > 0 {
			usage = modelSize / effectiveMem
		}

		switch {
		case usage <= 0.85:
			result.Compatible = append(result.Compatible, rec)
		case usage <= 1.05:
			result.Marginal = append(result.Marginal, rec)
		default:
			result.Incompatible = append(result.Incompatible, rec)
		}
	}

	sort.Slice(result.Compatible, func(i, j int) bool {
		return result.Compatible[i].Score.FinalScore > result.Compatible[j].Score.FinalScore
	})
	sort.Slice(result.Marginal, func(i, j int) bool {
		return result.Marginal[i].Score.FinalScore > result.Marginal[j].Score.FinalScore
	})

	if len(result.Compatible) > 0 {
		top := result.Compatible[0]
		result.TopPick = &top
	}

	return result, nil
}

func Recommend(category string, hw *HardwareProfile) []Recommendation {
	if category == "" {
		category = "general"
	}

	var recs []Recommendation
	for i := range CuratedCatalog {
		m := &CuratedCatalog[i]
		if !FitsInMemory(m, hw) {
			continue
		}
		if !MatchesCategory(m, category) && category != "general" {
			continue
		}

		score := ScoreModel(m, hw, category, 8192)
		recs = append(recs, Recommendation{
			Model: *m,
			Score: score,
			Fits:  true,
		})
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Score.FinalScore > recs[j].Score.FinalScore
	})

	if len(recs) > 10 {
		recs = recs[:10]
	}

	return recs
}

func RankInstalled(hw *HardwareProfile, useCase string) ([]Recommendation, error) {
	ollama := NewOllamaClient()
	status := ollama.CheckAvailability()
	if !status.Available {
		return nil, fmt.Errorf("ollama not available: %s", status.Error)
	}

	models, err := ollama.ListModels()
	if err != nil {
		return nil, err
	}

	catalogByTag := make(map[string]*ModelVariant)
	catalogByName := make(map[string]*ModelVariant)
	for i := range CuratedCatalog {
		catalogByTag[CuratedCatalog[i].OllamaTag] = &CuratedCatalog[i]
		catalogByName[CuratedCatalog[i].Name] = &CuratedCatalog[i]
	}

	var recs []Recommendation
	for _, installed := range models {
		var mv *ModelVariant

		if m, ok := catalogByTag[installed.Name]; ok {
			mv = m
		} else if m, ok := catalogByName[installed.Name]; ok {
			mv = m
		} else {
			mv = &ModelVariant{
				Name:       installed.Name,
				Family:     installed.Family,
				Quant:      installed.QuantLevel,
				SizeGB:     float64(installed.Size) / (1024 * 1024 * 1024),
				ContextLen: 4096,
				OllamaTag:  installed.Name,
			}
			mv.ParamsB = parseParamSize(installed.ParamSize)
		}

		score := ScoreModel(mv, hw, useCase, 8192)
		recs = append(recs, Recommendation{
			Model:     *mv,
			Score:     score,
			Fits:      FitsInMemory(mv, hw),
			Installed: true,
		})
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Score.FinalScore > recs[j].Score.FinalScore
	})

	return recs, nil
}

func parseParamSize(s string) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(s, "b")

	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

func FormatRecommendation(rec Recommendation) string {
	installed := ""
	if rec.Installed {
		installed = " [INSTALLED]"
	}
	return fmt.Sprintf("%-28s  Score: %5.1f  (Q:%.0f S:%.0f F:%.0f C:%.0f)  ~%.0f TPS  %.1fGB%s",
		rec.Model.Name,
		rec.Score.FinalScore,
		rec.Score.Quality,
		rec.Score.Speed,
		rec.Score.Fit,
		rec.Score.Context,
		rec.Score.EstTPS,
		rec.Model.EffectiveSize(),
		installed,
	)
}
