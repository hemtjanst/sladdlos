package tradfri

type Light struct {
	LightSetting
	TransitionTime        *int     `json:"5712,omitempty"`
	CumulativeActivePower *float64 `json:"5805,omitempty"`
	OnTime                *int64   `json:"5852,omitempty"`
	PowerFactor           *float64 `json:"5820,omitempty"`
	Unit                  *string  `json:"5701,omitempty"`
}
