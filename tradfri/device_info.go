package tradfri

import "strings"

type DeviceInfo struct {
	Manufacturer string `json:"0"`
	Model        string `json:"1"`
	SerialNumber string `json:"2"`
	Firmware     string `json:"3"`
	Power        int    `json:"6"`
	Battery      int    `json:"9"`
}

func (d *DeviceInfo) IsRGBModel() bool {
	return strings.Contains(d.Model, " CWS ")
}
