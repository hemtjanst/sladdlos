package tradfri

type OnOff struct {
	On *YesNo `json:"5850,omitempty"`
}

func (d *OnOff) IsOn() bool {
	return d.On != nil && *d.On == Yes
}
