package tradfri

import "math"

type Dimmable struct {
	On  *YesNo `json:"5850,omitempty"`
	Dim *uint8 `json:"5851,omitempty"`
}

func (d *Dimmable) dimPercent() float64 {
	if d.Dim == nil {
		return 0
	}
	// Non-linear scaling for better precision.
	if *d.Dim <= 10 {
		return float64(*d.Dim)
	}
	if *d.Dim <= 69 {
		return float64(*d.Dim)/2 + 11
	}
	return (float64(*d.Dim)-69)/3.1 + 40
}

func (d *Dimmable) DimPercent() float64 {
	return math.Floor((d.dimPercent()+.5)*100) / 100
}

func (d *Dimmable) DimInt() int {
	return int(d.DimPercent())
}

func (d *Dimmable) IsOn() bool {
	return d.On != nil && *d.On == Yes
}

// Converts a percentage (0-100) into a value between 0 and 255
func calcDim(dim int) uint8 {
	newDim := 0
	if dim < 0 {
		newDim = 0
	} else if dim <= 10 {
		newDim = dim
	} else if dim <= 40 {
		newDim = dim*2 - 11
	} else {
		dimf := float64(dim-40)*3.1 + 69
		if dimf > 254 {
			newDim = 254
		} else {
			newDim = int(dimf)
		}
	}
	return uint8(newDim)
}
