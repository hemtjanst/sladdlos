package tradfri

type Scene struct {
	observable
	BaseType
	group                   *Group
	Index                   int             `json:"9057,omitempty"`
	IsPredefined            YesNo           `json:"9068,omitempty"`
	IsActive                YesNo           `json:"9058,omitempty"`
	LightSettings           []*LightSetting `json:"15013,omitempty"`
	UseCurrentLightSettings YesNo           `json:"9070,omitempty"`
}
