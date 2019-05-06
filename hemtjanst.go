package sladdlos

import (
	"context"
	"hemtjan.st/sladdlos/tradfri"
	"lib.hemtjan.st/device"
	"strconv"
	"sync"
)

type HemtjanstClient struct {
	sync.RWMutex
	Id           string
	Announce     bool
	SkipGroup    bool
	SkipBulb     bool
	transport    device.Transport
	tree         *tradfri.Tree
	devices      map[string]*HemtjanstDevice
	groups       map[int]*tradfri.Group
	accessories  map[int]*tradfri.Accessory
	newDevChan   chan *tradfri.Accessory
	newGroupChan chan *tradfri.Group
}

func NewHemtjanstClient(tree *tradfri.Tree, transport device.Transport, id string) *HemtjanstClient {
	h := &HemtjanstClient{
		tree:         tree,
		transport:    transport,
		Id:           id,
		SkipBulb:     false,
		SkipGroup:    false,
		devices:      map[string]*HemtjanstDevice{},
		newDevChan:   make(chan *tradfri.Accessory),
		newGroupChan: make(chan *tradfri.Group),
		groups:       map[int]*tradfri.Group{},
		accessories:  map[int]*tradfri.Accessory{},
	}
	tree.AddCallback(h)
	return h
}

func topicFor(t string, a tradfri.Instance) string {
	return "light/" + t + "-" + strconv.Itoa(a.GetInstanceID())
}

func topicForPlug(t string, a tradfri.Instance) string {
	return "outlet/" + t + "-" + strconv.Itoa(a.GetInstanceID())
}

func (h *HemtjanstClient) Start(ctx context.Context) {
	for {
		select {
		case d, op := <-h.newDevChan:
			if !op {
				return
			}
			go func(d *tradfri.Accessory) {
				h.Lock()
				defer h.Unlock()
				if _, ok := h.accessories[d.GetInstanceID()]; !ok {
					h.accessories[d.GetInstanceID()] = d
				}
				h.ensureDevices()
			}(d)
		case g, op := <-h.newGroupChan:
			if !op {
				return
			}
			go func(g *tradfri.Group) {
				h.Lock()
				defer h.Unlock()
				if _, ok := h.groups[g.GetInstanceID()]; !ok {
					h.groups[g.GetInstanceID()] = g
				}
				h.ensureDevices()
			}(g)
		case <-ctx.Done():
			return
		}
	}
}

func (h *HemtjanstClient) ensureDevices() {
	ownerGroup := map[int]int{}

	for _, grp := range h.groups {
		hasLight := false
		hasPlug := false
		if grp.Members != nil {
			for _, member := range grp.Members {
				ownerGroup[member] = grp.GetInstanceID()
				if l, ok := h.accessories[member]; ok {
					if l.IsLight() {
						hasLight = true
					}
					if l.IsPlug() {
						hasPlug = true
					}
				}
			}
		}

		var topic string
		if hasLight {
			topic = topicFor("grp", grp)
		} else if hasPlug {
			topic = topicForPlug("grp", grp)
		} else {
			continue
		}
		if _, ok := h.devices[topic]; ok {
			continue
		}
		dev := NewHemtjanstGroup(h, topic, grp)
		h.devices[topic] = dev
	}

	for _, light := range h.accessories {
		topic := topicFor("bulb", light)
		if light.IsPlug() {
			topic = topicForPlug("plug", light)
		}

		if _, ok := h.devices[topic]; ok {
			continue
		}
		var owner *tradfri.Group
		if grpId, ok := ownerGroup[light.GetInstanceID()]; ok {
			if owner, ok = h.groups[grpId]; !ok {
				continue
			}
		} else {
			// Wait until we have the group
			continue
		}
		ownerTopic := topicFor("grp", owner)
		var ownerDev *HemtjanstDevice
		var ok bool
		if ownerDev, ok = h.devices[ownerTopic]; !ok {
			// Try with outlet topic
			ownerTopic = topicForPlug("grp", owner)
			if ownerDev, ok = h.devices[ownerTopic]; !ok {
				continue
			}
		}

		dev := NewHemtjanstAccessory(h, topic, light, ownerDev)
		h.devices[topic] = dev
		ownerDev.AddMember(dev)
	}

}

func (h *HemtjanstClient) OnNewAccessory(d *tradfri.Accessory) {
	go func() {
		h.newDevChan <- d
	}()
}

func (h *HemtjanstClient) OnNewGroup(g *tradfri.Group) {
	go func() {
		h.newGroupChan <- g
	}()
}

func (h *HemtjanstClient) OnNewScene(g *tradfri.Group, s *tradfri.Scene) {

}
