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
	Color      string `json:"5706,omitempty"`
	Hue        int    `json:"5707,omitempty"`
	Saturation int    `json:"5708,omitempty"`
	ColorX     int    `json:"5709,omitempty"`
	ColorY     int    `json:"5710,omitempty"`

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
	h, s, _ := c.Hsv()
	l.Hue = int(h*(65535/360) + 0.5)
	if l.Hue >= 65535 {
		l.Hue = 65535
	}
	l.Saturation = int(s*65279 + 0.5)
	if l.Saturation >= 65279 {
		l.Saturation = 65279
	}
}

func (l *LightSetting) GetColor() colorful.Color {
	hue := float64(l.Hue) / (65535 / 360)
	sat := float64(l.Saturation) / 65279
	return colorful.Hsv(hue, sat, 1)
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
