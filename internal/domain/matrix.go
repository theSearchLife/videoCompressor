package domain

type MatrixConfig struct {
	Codecs      []string
	CRFs        map[string][]int // codec -> CRF values
	Presets     []string
	Resolutions []Resolution
}

func DefaultMatrixConfig() MatrixConfig {
	return MatrixConfig{
		Codecs: []string{"libx265", "libx264"},
		CRFs: map[string][]int{
			"libx265": {23, 26, 28},
			"libx264": {20, 23, 25},
		},
		Presets:     []string{"slow", "medium", "fast"},
		Resolutions: []Resolution{Res1080p, Res720p},
	}
}

func (m MatrixConfig) Profiles() []Profile {
	var profiles []Profile
	for _, codec := range m.Codecs {
		crfs := m.CRFs[codec]
		for _, crf := range crfs {
			for _, preset := range m.Presets {
				profiles = append(profiles, Profile{
					Name:            "matrix",
					Codec:           codec,
					CRF:             crf,
					Preset:          preset,
					AudioCodec:      "copy",
					ContainerFormat: "mp4",
				})
			}
		}
	}
	return profiles
}

func (m MatrixConfig) TotalCombinations(numSources int) int {
	profileCount := 0
	for _, codec := range m.Codecs {
		profileCount += len(m.CRFs[codec]) * len(m.Presets)
	}
	return numSources * profileCount * len(m.Resolutions)
}
