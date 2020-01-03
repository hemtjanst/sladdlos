package sladdlos

import (
	"fmt"
	"github.com/lucasb-eyer/go-colorful"
	"hemtjan.st/sladdlos/tradfri"
	"lib.hemtjan.st/client"
	"lib.hemtjan.st/device"
	"lib.hemtjan.st/feature"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	lTypeNone = iota
	lTypeTemp
	lTypeRgb
)

type HemtjanstDevice struct {
	sync.RWMutex
	client         *HemtjanstClient
	Topic          string
	isRunning      bool
	isGroup        bool
	accessory      *tradfri.Accessory
	members        []*HemtjanstDevice
	group          *tradfri.Group
	device         client.Device
	features       map[string]client.Feature
	lastHue        *int
	lastSaturation *int
	blind          *blindInfo
}
type blindInfo struct {
	sync.RWMutex
	lastPosition   *int
	targetPosition *int
	direction      blindDirection
	timer          *time.Timer
}
type blindDirection int

const (
	blindClosing blindDirection = 0
	blindOpening blindDirection = 1
	blindStopped blindDirection = 2
)

func NewHemtjanstAccessory(client *HemtjanstClient, topic string, accessory *tradfri.Accessory, group *HemtjanstDevice) *HemtjanstDevice {
	h := &HemtjanstDevice{
		Topic:     topic,
		client:    client,
		isRunning: false,
		isGroup:   false,
		accessory: accessory,
	}
	if group != nil {
		h.members = []*HemtjanstDevice{group}
	}
	h.init()
	return h
}

func NewHemtjanstGroup(client *HemtjanstClient, topic string, group *tradfri.Group) *HemtjanstDevice {
	h := &HemtjanstDevice{
		Topic:     topic,
		client:    client,
		isRunning: false,
		isGroup:   true,
		members:   []*HemtjanstDevice{},
		group:     group,
	}
	return h
}

func (h *HemtjanstDevice) shouldSkip() bool {
	return h.isGroup && h.client.SkipGroup ||
		!h.isGroup && h.accessory.IsLight() && h.client.SkipBulb
}

func (h *HemtjanstDevice) AddMember(member *HemtjanstDevice) {
	h.Lock()
	defer h.Unlock()
	if h.members == nil {
		h.members = []*HemtjanstDevice{}
	}
	h.members = append(h.members, member)
	h.init()
}

func (h *HemtjanstDevice) init() {
	if h.isRunning {
		return
	}
	if h.client == nil {
		return
	}
	var dev *device.Info

	lType := lTypeNone
	if h.isGroup {
		if h.group == nil {
			return
		}
		if h.group.Members == nil || len(h.group.Members) != len(h.members) {
			return
		}

		dev = &device.Info{
			Topic:        h.Topic,
			Name:         h.group.Name,
			Manufacturer: "IKEA",
			Model:        "Trådfri Group",
			SerialNumber: strconv.Itoa(h.group.GetInstanceID()),
			Features:     map[string]*feature.Info{},
		}

		hasLight := false

		for _, d := range h.members {
			if d.accessory != nil {
				if d.accessory.IsLight() {
					hasLight = true
				} else {
					continue
				}
				if l := d.accessory.Light(); l != nil {
					if l.HasColorTemperature() {
						lType = lTypeTemp
					}
				}
				if d.accessory.DeviceInfo.IsRGBModel() {
					lType = lTypeRgb
					break
				}
			}
		}

		if hasLight {
			dev.Type = "lightbulb"
			dev.Features["on"] = &feature.Info{}
			dev.Features["brightness"] = &feature.Info{}
		}
	} else {
		if h.accessory == nil || len(h.members) == 0 {
			return
		}
		owner := h.members[0]
		if owner.group == nil {
			return
		}

		dev = &device.Info{
			Topic:        h.Topic,
			Name:         h.accessory.Name,
			Manufacturer: h.accessory.DeviceInfo.Manufacturer,
			Model:        h.accessory.DeviceInfo.Model,
			SerialNumber: strconv.Itoa(h.accessory.GetInstanceID()),
			Features:     map[string]*feature.Info{},
		}
		if h.accessory.IsLight() {
			dev.Type = "lightbulb"
			dev.Features["on"] = &feature.Info{}
			dev.Features["brightness"] = &feature.Info{}
			if h.accessory.Light().HasColorTemperature() {
				lType = lTypeTemp
			}
			if h.accessory.DeviceInfo.IsRGBModel() {
				lType = lTypeRgb
			}
		} else if h.accessory.IsPlug() {
			dev.Type = "outlet"
			dev.Features["on"] = &feature.Info{}
			dev.Features["outletInUse"] = &feature.Info{}
		} else if h.accessory.IsBlind() {
			h.blind = &blindInfo{direction: blindStopped}
			dev.Type = "windowCovering"
			dev.Features["targetPosition"] = &feature.Info{Min: 0, Max: 100, Step: 1}
			dev.Features["currentPosition"] = &feature.Info{Min: 0, Max: 100, Step: 1}
			dev.Features["positionState"] = &feature.Info{Min: 0, Max: 2, Step: 1}
		}
	}

	switch lType {
	case lTypeTemp:
		dev.Features["colorTemperature"] = &feature.Info{}
	case lTypeRgb:
		dev.Features["hue"] = &feature.Info{}
		dev.Features["saturation"] = &feature.Info{}
		dev.Features["color"] = &feature.Info{}
	}

	if dev.Type == "" {
		log.Printf("Unsupported device: %+v", dev)
		return
	}

	h.isRunning = true
	var err error
	if !h.shouldSkip() {
		h.device, err = client.NewDevice(dev, h.client.transport)
		if err != nil {
			log.Printf("Error creating device: %s", err)
		}
		for _, ft := range h.device.Features() {
			ft := ft
			_ = ft.OnSetFunc(func(val string) {
				h.onDeviceSet(ft.Name(), val)
			})
			err := h.publish(ft.Name())
			if err != nil {
				log.Printf("Error publishing to %s: %s", ft.Name(), err)
			}
		}
		log.Printf("[%s] Started", h.Topic)
	}
	if h.isGroup && h.group != nil {
		h.group.Observe(h.onTradfriChange)
	} else if h.accessory != nil {
		h.accessory.Observe(h.onTradfriChange)
	}
}

func (h *HemtjanstDevice) onDeviceSet(feature string, newValue string) {
	log.Printf("[%s] New suggested value for %s: %s", h.Topic, feature, newValue)
	switch feature {
	case "on":
		on := newValue != "0" && strings.ToLower(newValue) != "false"
		if h.isGroup && h.group != nil {
			h.group.SetOn(on)
		} else if h.accessory != nil {
			h.accessory.SetOn(on)
		}
	case "brightness":
		if dim, err := strconv.Atoi(newValue); err == nil {
			if h.isGroup && h.group != nil {
				h.group.SetDim(dim)
			} else if h.accessory != nil {
				h.accessory.SetDim(dim)
			}
		}
	case "colorTemperature":
		if temp, err := strconv.Atoi(newValue); err == nil {
			newTemp := "warm"
			if temp < 150 {
				newTemp = "cold"
			} else if temp < 250 {
				newTemp = "normal"
			}
			if h.isGroup && h.group != nil {
				for _, m := range h.members {
					if m.accessory != nil && m.accessory.Light() != nil {
						if m.accessory.Light().HasColorTemperature() {
							m.accessory.SetColorTemp(newTemp)
						}
					}
				}
			} else if h.accessory != nil {
				h.accessory.SetColorTemp(newTemp)
			}
		}
	case "color":
		h.updateColor(newValue)
	case "hue":
		if hue, err := strconv.Atoi(newValue); err == nil {
			h.lastHue = &hue
			h.updateColor("")
		}
	case "saturation":
		if saturation, err := strconv.Atoi(newValue); err == nil {
			h.lastSaturation = &saturation
			h.updateColor("")
		}
	case "targetPosition":
		if pos, err := strconv.Atoi(newValue); err == nil && pos >= 0 && pos <= 100 {
			if h.isGroup && h.group != nil {
				// Not supported
			} else if h.accessory != nil && h.accessory.IsBlind() {
				pos = 100 - pos
				h.accessory.SetBlindPosition(pos)
				if h.blind == nil {
					h.blind = &blindInfo{direction: blindStopped}
				}
				h.blind.targetPosition = &pos
			}
		}
	}
}

func (h *HemtjanstDevice) updateColor(rgb string) {
	var newColor colorful.Color

	if rgb == "" {
		if h.lastHue == nil || h.lastSaturation == nil {
			return
		}
		hue := *h.lastHue
		sat := *h.lastSaturation

		newColor = colorful.Hsv(float64(hue), float64(sat)/100, float64(1))
	} else {
		if rgb[0] != '#' {
			rgb = "#" + rgb
		}
		var err error
		newColor, err = colorful.Hex(rgb)
		if err != nil {
			return
		}
	}

	if h.isGroup && h.group != nil {
		for _, m := range h.members {
			if m.accessory != nil && m.accessory.Light() != nil {
				if m.accessory.DeviceInfo.IsRGBModel() {
					m.accessory.SetColor(newColor)
				}
			}
		}
	} else if h.accessory != nil {
		h.accessory.SetColor(newColor)
	}

}

func (h *HemtjanstDevice) dimmable() *tradfri.Dimmable {
	if h.isGroup {
		if h.group != nil {
			return &h.group.Dimmable
		}
		return nil
	}
	if h.accessory == nil {
		return nil
	}
	l := h.accessory.Light()
	if l == nil {
		return nil
	}
	return &l.Dimmable
}

func (h *HemtjanstDevice) onOff() *tradfri.OnOff {
	if h.isGroup {
		if h.group != nil {
			return &h.group.OnOff
		}
		return nil
	}
	if h.accessory == nil {
		return nil
	}
	l := h.accessory.Light()
	if l == nil {
		p := h.accessory.Plug()
		if p == nil {
			return nil
		}
		return &p.OnOff
	}
	return &l.OnOff
}

func (h *HemtjanstDevice) lightSetting() *tradfri.LightSetting {
	if h.isGroup {
		return nil
	}
	if h.accessory == nil {
		return nil
	}
	l := h.accessory.Light()
	if l == nil {
		return nil
	}
	return &l.LightSetting
}

func (h *HemtjanstDevice) featureVal(feature string) (string, error) {
	if h.isGroup {
		min := math.MaxInt32
		max := math.MinInt32
		var last string

		for _, m := range h.members {
			val, err := m.featureVal(feature)
			if err != nil {
				continue
			}
			if val != "" {
				last = val
			}
			ival, err := strconv.Atoi(val)
			if err != nil {
				continue
			}
			if ival < min {
				min = ival
			}
			if ival > max {
				max = ival
			}
		}

		switch feature {
		case "on":
			if max == 1 {
				return "1", nil
			}
			return "0", nil
		case "brightness":
			if max != math.MinInt32 {
				return strconv.Itoa(max), nil
			}
			return "0", nil
		case "colorTemperature":
			return last, nil
		case "reachable":
			if min == 0 {
				return "0", nil
			}
			return "1", nil
		case "hue", "saturation", "color":
			if last == "" {
				return "", fmt.Errorf("device doesn't support %s", feature)
			}
			return last, nil
		case "outletInUse":
			// Currently no way of detecting
			return "1", nil
		case "currentPosition", "targetPosition", "positionState":
			return "2", nil
		}
	}
	switch feature {
	case "on":
		onoff := h.onOff()
		if onoff == nil {
			return "", fmt.Errorf("device doesn't support %s", feature)
		}
		if onoff.IsOn() {
			return "1", nil
		}
		return "0", nil
	case "brightness":
		dim := h.dimmable()
		if dim == nil {
			return "", fmt.Errorf("device doesn't support %s", feature)
		}
		return strconv.Itoa(dim.DimInt()), nil
	case "colorTemperature":
		ls := h.lightSetting()
		if ls == nil || !ls.HasColorTemperature() {
			return "", fmt.Errorf("device doesn't support %s", feature)
		}
		switch ls.GetColorName() {
		case "cold":
			return "111", nil
		case "warm":
			return "400", nil
		default:
			return "222", nil
		}
	case "reachable":
		if h.isGroup || h.accessory == nil {

		}
		if h.accessory.IsAlive() {
			return "1", nil
		}
		return "0", nil
	case "hue":
		ls := h.lightSetting()
		if ls == nil {
			return "", fmt.Errorf("device doesn't support %s", feature)
		}
		hue, _, _ := ls.GetColor().Hsv()
		return strconv.Itoa(int(hue)), nil
	case "saturation":
		ls := h.lightSetting()
		if ls == nil {
			return "", fmt.Errorf("device doesn't support %s", feature)
		}
		_, sat, _ := ls.GetColor().Hsv()
		return strconv.Itoa(int(sat * 100)), nil
	case "color":
		ls := h.lightSetting()
		if ls == nil {
			return "", fmt.Errorf("device doesn't support %s", feature)
		}
		return ls.GetColor().Hex(), nil
	case "outletInUse":
		// Currently no way of detecting
		return "1", nil
	case "currentPosition":
		if bl := h.accessory.Blind(); bl != nil {
			log.Printf("currentPosition reported as %d", 100-bl.Pos())
			return strconv.Itoa(100 - bl.Pos()), nil
		}
	case "targetPosition":
		if bl := h.accessory.Blind(); bl != nil {
			r := 100 - bl.Pos()
			if h.blind != nil {
				if h.blind.targetPosition != nil {
					r = 100 - *h.blind.targetPosition
				} else if h.blind.direction == blindOpening {
					r = 0
				} else if h.blind.direction == blindClosing {
					r = 100
				}
			}
			log.Printf("targetPosition reported as %d", r)
			return strconv.Itoa(r), nil
		}
	case "positionState":
		if h.blind != nil {
			log.Printf("positionState reported as %d", h.blind.direction)
			return strconv.Itoa(int(h.blind.direction)), nil
		}
	}
	return "", fmt.Errorf("device doesn't support %s", feature)
}

func (h *HemtjanstDevice) publish(feature string) error {
	var err error
	if !h.isGroup && len(h.members) == 1 {
		if err := h.members[0].publish(feature); err != nil {
			return err
		}
	}
	newVal, err := h.featureVal(feature)
	if err != nil {
		return err
	}
	if h.device == nil {
		return fmt.Errorf("no device created")
	}

	return h.device.Feature(feature).Update(newVal)
}

func unptr(i interface{}) interface{} {
	if rf := reflect.ValueOf(i); !rf.IsNil() && rf.Type().Kind() == reflect.Ptr {
		return rf.Elem().Interface()
	}
	return i
}

func (h *HemtjanstDevice) onTradfriChange(change []*tradfri.ObservedChange) {
	colorUpdated := false
	for _, ch := range change {
		log.Printf("[%s] %s changed from %v to %v", h.Topic, ch.Field, unptr(ch.OldValue), unptr(ch.NewValue))
		switch ch.Field {
		case "Dim":
			h.publish("brightness")
		case "On":
			h.publish("on")
		case "Color", "ColorX", "ColorY", "Hue", "Saturation":
			if !colorUpdated {
				colorUpdated = true
				h.publish("colorTemperature")
				h.publish("hue")
				h.publish("saturation")
				h.publish("color")
			}
		case "Alive":
			h.publish("reachable")
		case "Position":
			h.publish("currentPosition")
			if h.blind != nil {
				h.blind.onUpdate(h.accessory.Blind(), h.publish)
			}
		}

	}
}

func (b *blindInfo) onUpdate(blind *tradfri.Blind, cb func(string) error) {
	b.Lock()
	defer b.Unlock()
	pos := blind.Pos()
	if b.lastPosition == nil {
		b.lastPosition = &pos
		b.direction = blindStopped
		_ = cb("positionState")
		_ = cb("targetPosition")
	}
	if b.timer == nil {
		tmr := time.NewTimer(3 * time.Second)
		b.timer = tmr
		go func() {
			<-tmr.C
			b.Lock()
			defer b.Unlock()
			if b.timer != tmr {
				return
			}
			b.timer = nil
			b.direction = blindStopped
			b.targetPosition = nil
			go func() {
				_ = cb("positionState")
				_ = cb("targetPosition")
			}()
		}()
		return
	}
	dir := b.direction
	lastPos := *b.lastPosition
	b.lastPosition = &pos
	if lastPos < pos {
		dir = blindClosing
	} else if lastPos > pos {
		dir = blindOpening
	}
	report := map[string]bool{}

	if dir != b.direction {
		b.direction = dir
		report["positionState"] = true
		report["targetPosition"] = true
	}
	if b.targetPosition != nil && *b.targetPosition == pos ||
		(dir == blindClosing && pos == 100) ||
		(dir == blindOpening && pos == 0) {
		b.direction = blindStopped
		b.targetPosition = nil
		b.timer = nil
		report["positionState"] = true
		report["targetPosition"] = true
	} else {
		b.timer.Reset(1500 * time.Millisecond)
	}
	if len(report) > 0 {
		go func() {
			for k, r := range report {
				if r {
					_ = cb(k)
				}
			}
		}()
	}
}
