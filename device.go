package sladdlos

import (
	"github.com/hemtjanst/hemtjanst/device"
	"github.com/hemtjanst/hemtjanst/messaging"
	"github.com/hemtjanst/sladdlos/tradfri"
	"log"
	"strconv"
	"strings"
	"sync"
)

type HemtjanstDevice struct {
	sync.RWMutex
	client    *HemtjanstClient
	mqClient  messaging.PublishSubscriber
	Topic     string
	isRunning bool
	isGroup   bool
	accessory *tradfri.Accessory
	members   []*HemtjanstDevice
	group     *tradfri.Group
	device    *device.Device
	features  map[string]*device.Feature
}

func NewHemtjanstAccessory(client *HemtjanstClient, topic string, accessory *tradfri.Accessory, group *HemtjanstDevice) *HemtjanstDevice {
	h := &HemtjanstDevice{
		Topic:     topic,
		client:    client,
		isRunning: false,
		isGroup:   false,
		members:   []*HemtjanstDevice{group},
		accessory: accessory,
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

func (h *HemtjanstDevice) OnConnect() {
	h.subscribeFeatures()
}

func (h *HemtjanstDevice) OnDiscover() {
	if h.device != nil {
		h.device.PublishMeta()
	}
}

func (h *HemtjanstDevice) subscribeFeatures() {
	if h.device != nil {
		h.device.RLock()
		defer h.device.RUnlock()
		for k, v := range h.device.Features {
			h.handleFeature(k, v)
		}
	}
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
	if h.client == nil || h.client.MQTT == nil {
		return
	}
	var dev *device.Device
	hasColorTemp := false
	if h.isGroup {
		if h.group == nil {
			return
		}
		if h.group.Members == nil || len(h.group.Members) != len(h.members) {
			//log.Printf("[%s] Not enough members yet (%d/%d)", h.Topic, len(h.members), len(h.group.Members))
			return
		}

		dev = device.NewDevice(h.Topic, h.client.MQTT)
		dev.Name = h.group.Name
		dev.Type = "lightbulb"
		dev.Manufacturer = "IKEA"
		dev.Model = "Trådfri Group"
		dev.SerialNumber = strconv.Itoa(h.group.GetInstanceID())
		dev.LastWillID = h.client.Id

		for _, d := range h.members {
			if d.accessory != nil {
				if l := d.accessory.Light(); l != nil {
					if l.HasColorTemperature() {
						hasColorTemp = true
						break
					}
				}
			}
		}
	} else {
		if h.members == nil || h.accessory == nil || len(h.members) == 0 {
			return
		}
		owner := h.members[0]
		if owner.group == nil {
			return
		}

		if !h.accessory.IsLight() {
			return
		}

		dev = device.NewDevice(h.Topic, h.client.MQTT)
		dev.Name = owner.group.Name + ": " + h.accessory.Name
		dev.Type = "lightbulb"
		dev.Manufacturer = h.accessory.DeviceInfo.Manufacturer
		dev.Model = h.accessory.DeviceInfo.Model
		dev.SerialNumber = strconv.Itoa(h.accessory.GetInstanceID())
		dev.LastWillID = h.client.Id

		dev.AddFeature("reachable", &device.Feature{})
		hasColorTemp = h.accessory.Light().HasColorTemperature()
	}

	dev.AddFeature("on", &device.Feature{})
	dev.AddFeature("brightness", &device.Feature{})
	if hasColorTemp {
		dev.AddFeature("colorTemperature", &device.Feature{})
	}
	if dev != nil {
		h.isRunning = true
		h.device = dev
		h.subscribeFeatures()
		if h.client.Announce {
			h.device.PublishMeta()
		}
		if h.isGroup && h.group != nil {
			h.group.Observe(h.onTradfriChange)
		} else if h.accessory != nil {
			h.accessory.Observe(h.onTradfriChange)
		}
		log.Printf("[%s] Started", h.Topic)
	}
}

func (h *HemtjanstDevice) handleFeature(name string, ft *device.Feature) {
	if h.device == nil {
		return
	}
	h.device.RLock()
	defer h.device.RUnlock()

	if h.device.Features == nil {
		return
	}

	for k, ft := range h.device.Features {
		ftName := k
		ft.OnSet(func(msg messaging.Message) {
			h.onDeviceSet(ftName, string(msg.Payload()))
		})
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
							m.accessory.SetColor(newTemp)
						}
					}
				}
			} else if h.accessory != nil {
				h.accessory.SetColor(newTemp)
			}
		}
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

func (h *HemtjanstDevice) onTradfriChange(change []*tradfri.ObservedChange) {
	for _, ch := range change {
		log.Printf("[%s] %s changed from %v to %v", h.Topic, ch.Field, ch.OldValue, ch.NewValue)
		switch ch.Field {
		case "Dim":
			dim := h.dimmable()
			if dim == nil {
				continue
			}
			newDim := dim.DimInt()
			if ft, err := h.device.GetFeature("brightness"); err == nil && ft != nil {
				ft.Update(strconv.Itoa(newDim))
			}
		case "On":
			dim := h.dimmable()
			if dim == nil {
				continue
			}
			val := "0"
			if dim.IsOn() {
				val = "1"
			}
			if ft, err := h.device.GetFeature("on"); err == nil && ft != nil {
				ft.Update(val)
			}
		case "Color":
			ls := h.lightSetting()
			if ls == nil {
				continue
			}
			newVal := ""
			switch ls.GetColorName() {
			case "cold":
				newVal = "111"
			case "normal":
				newVal = "222"
			case "warm":
				newVal = "400"
			}
			if ft, err := h.device.GetFeature("colorTemperature"); err == nil && ft != nil {
				ft.Update(newVal)
			}
		case "Alive":
			if h.isGroup || h.accessory == nil {
				continue
			}
			val := "0"
			if h.accessory.IsAlive() {
				val = "1"
			}
			if ft, err := h.device.GetFeature("reachable"); err == nil && ft != nil {
				ft.Update(val)
			}
		}

	}
}
