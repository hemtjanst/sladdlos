package tradfri

import (
	"github.com/lucasb-eyer/go-colorful"
	"image/color"
	"strings"
)

const (
	Cold    = "f5faf6"
	Normal  = "f1e0b5"
	Warm    = "efd275"
	coldX   = 24930
	coldY   = 24694
	normalX = 30140
	normalY = 26909
	warmX   = 33135
	warmY   = 27211
)

type LightSetting struct {
	BaseType
	Dimmable
	Color  string `json:"5706,omitempty"`
	ColorX int    `json:"5709,omitempty"`
	ColorY int    `json:"5710,omitempty"`

	Field5707 int `json:"5707,omitempty"`
	Field5708 int `json:"5708,omitempty"`
	Field5711 int `json:"5711,omitempty"`
}

func (l *LightSetting) SetColorTemp(c string) {
	switch strings.ToLower(c) {
	case "cold", Cold:
		l.Color = Cold
		l.ColorX = coldX
		l.ColorY = coldY
	case "normal", Normal:
		l.Color = Normal
		l.ColorX = normalX
		l.ColorY = normalY
	case "warm", Warm:
		l.Color = Warm
		l.ColorX = warmX
		l.ColorY = warmY
	}
}

func (l *LightSetting) SetColor(color color.Color) {
	var c colorful.Color
	var ok bool
	if c, ok = color.(colorful.Color); !ok {
		c = colorful.MakeColor(color)
	}
	x, y, _ := c.Xyy()
	l.ColorX = int(x*65535 + 0.5)
	l.ColorY = int(y*65535 + 0.5)
}

func (l *LightSetting) GetColorName() string {
	switch l.Color {
	case Cold:
		return "cold"
	case Normal:
		return "normal"
	case Warm:
		return "warm"
	default:
		return l.Color
	}
}

func (l *LightSetting) SetColorCold() {
	l.SetColorTemp(Cold)
}

func (l *LightSetting) SetColorNormal() {
	l.SetColorTemp(Normal)
}

func (l *LightSetting) SetColorWarm() {
	l.SetColorTemp(Warm)
}

func (l *LightSetting) HasColorTemperature() bool {
	n := l.GetColorName()
	return n == "cold" || n == "normal" || n == "warm"
}
