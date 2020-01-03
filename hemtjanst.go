package sladdlos

import (
	"context"
	"hemtjan.st/sladdlos/tradfri"
	"lib.hemtjan.st/device"
	"strconv"
	"strings"
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

func topicFor(a tradfri.Instance, t ...string) string {
	return strings.Join(t, "/") + "-" + strconv.Itoa(a.GetInstanceID())
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
		topic := topicFor(grp, "light", "grp")
		if _, ok := h.devices[topic]; ok {
			continue
		}
		dev := NewHemtjanstGroup(h, topic, grp)
		h.devices[topic] = dev

		if grp.Members != nil {
			for _, member := range grp.Members {
				ownerGroup[member] = grp.GetInstanceID()
			}
		}
	}

	for _, accessory := range h.accessories {
		var topic string
		if accessory.IsLight() {
			topic = topicFor(accessory, "light", "bulb")
		} else if accessory.IsPlug() {
			topic = topicFor(accessory, "outlet", "plug")
		} else if accessory.IsBlind() {
			topic = topicFor(accessory, "windowCovering", "blind")
		} else if accessory.IsRemote() {
			topic = topicFor(accessory, "remote", "remote")
		} else {
			topic = topicFor(accessory, "unknown", "unknown")
		}

		if _, ok := h.devices[topic]; ok {
			continue
		}
		var ownerDev *HemtjanstDevice

		var owner *tradfri.Group
		if grpId, ok := ownerGroup[accessory.GetInstanceID()]; ok {
			if owner, ok = h.groups[grpId]; !ok {
				continue
			}
		} else {
			// Wait until we have the group
			continue
		}
		ownerTopic := topicFor(owner, "light", "grp")
		var ok bool
		if ownerDev, ok = h.devices[ownerTopic]; !ok {
			continue
		}

		dev := NewHemtjanstAccessory(h, topic, accessory, ownerDev)
		h.devices[topic] = dev
		if ownerDev != nil {
			ownerDev.AddMember(dev)
		}
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
