package sladdlos

import (
	"github.com/hemtjanst/hemtjanst/messaging"
	"github.com/hemtjanst/sladdlos/tradfri"
	"log"
	"strconv"
	"sync"
	//"time"
)

const (
	colorCold   = 90
	colorNormal = 200
	colorWarm   = 400
)

type HemtjanstClient struct {
	sync.RWMutex
	MQTT         messaging.PublishSubscriber
	Id           string
	Announce     bool
	tree         *tradfri.Tree
	devices      map[string]*HemtjanstDevice
	groups       map[int]*tradfri.Group
	accessories  map[int]*tradfri.Accessory
	newDevChan   chan *tradfri.Accessory
	newGroupChan chan *tradfri.Group
}

func NewHemtjanstClient(tree *tradfri.Tree, id string) *HemtjanstClient {
	h := &HemtjanstClient{
		tree:         tree,
		Id:           id,
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

func (h *HemtjanstClient) start() {
	//tick := time.NewTicker(5 * time.Second)
	for {
		select {
		case d := <-h.newDevChan:
			go func(d *tradfri.Accessory) {
				h.Lock()
				defer h.Unlock()
				if _, ok := h.accessories[d.GetInstanceID()]; !ok {
					h.accessories[d.GetInstanceID()] = d
				}
				h.ensureDevices()
			}(d)
		case g := <-h.newGroupChan:
			go func(g *tradfri.Group) {
				h.Lock()
				defer h.Unlock()
				if _, ok := h.groups[g.GetInstanceID()]; !ok {
					h.groups[g.GetInstanceID()] = g
				}
				h.ensureDevices()
			}(g)
			/*case <-tick.C:
			log.Print("Tick")
			go func() {
				h.Lock()
				defer h.Unlock()
				log.Print("ensureDevices")
				h.ensureDevices()
			}()*/
		}
	}
}

func (h *HemtjanstClient) ensureDevices() {
	ownerGroup := map[int]int{}

	for _, grp := range h.groups {
		hasLight := false
		if grp.Members != nil {
			for _, member := range grp.Members {
				ownerGroup[member] = grp.GetInstanceID()
				if l, ok := h.accessories[member]; ok {
					if l.IsLight() {
						hasLight = true
					}
				}
			}
		}

		topic := topicFor("grp", grp)
		if !hasLight {
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
		if _, ok := h.devices[topic]; ok {
			continue
		}
		var owner *tradfri.Group
		if grpId, ok := ownerGroup[light.GetInstanceID()]; ok {
			if owner, ok = h.groups[grpId]; !ok {
				log.Printf("[%s] Owner group not initialized, continuing", topic)
				continue
			}
		} else {
			// Wait until we have the group
			log.Printf("[%s] Owner group not found yet, continuing", topic)
			continue
		}
		ownerTopic := topicFor("grp", owner)
		var ownerDev *HemtjanstDevice
		var ok bool
		if ownerDev, ok = h.devices[ownerTopic]; !ok {
			log.Printf("[%s] Owner group not found as %s", topic, ownerTopic)
			continue
		}

		dev := NewHemtjanstAccessory(h, topic, light, ownerDev)
		h.devices[topic] = dev
		ownerDev.AddMember(dev)
	}

}

func (h *HemtjanstClient) OnConnect(client messaging.PublishSubscriber) {
	go func() {
		if h.MQTT == nil {
			h.MQTT = client
			go h.start()
		}
		h.MQTT.Subscribe("discover", 1, h.onDiscover)
		for _, d := range h.devices {
			d.OnConnect()
		}
	}()
}

func (h *HemtjanstClient) onDiscover(message messaging.Message) {
	h.Announce = true
	go func() {
		h.RLock()
		defer h.RUnlock()
		for _, d := range h.devices {
			d.OnDiscover()
		}
	}()
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
