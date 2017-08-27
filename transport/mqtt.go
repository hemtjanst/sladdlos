package transport

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hemtjanst/sladdlos/tradfri"
	"log"
	"strings"
	"sync"
)

type Transport struct {
	sync.RWMutex
	client  mqtt.Client
	id      string
	tree    *tradfri.Tree
	waiting map[string]chan *tradfriReply
}

func NewTransport(id string) *Transport {
	m := &Transport{
		id:      id,
		waiting: map[string]chan *tradfriReply{},
	}
	return m
}

func (t *Transport) SetTree(tree *tradfri.Tree) {
	t.Lock()
	defer t.Unlock()
	if tree == nil {
		t.tree = nil
		if t.client != nil {
			t.client.Unsubscribe("tradfri-raw/#")
		}
		return
	}
	t.tree = tree
	if t.client != nil {
		t.subscribe()
	}
}

func (t *Transport) OnConnect(client mqtt.Client) {
	t.Lock()
	defer t.Unlock()
	t.client = client
	if t.tree != nil {
		t.subscribe()
	}
}

func (t *Transport) onMessage(c mqtt.Client, msg mqtt.Message) {
	t.RLock()
	defer t.RUnlock()
	if t.tree == nil {
		return
	}
	topic := strings.Split(msg.Topic(), "/")
	if len(topic) < 2 || topic[0] != "tradfri-raw" {
		return
	}
	go func() {
		err := t.tree.Populate(topic[1:], msg.Payload())

		if err != nil {
			log.Print(err)
			log.Printf("^- While parsing from %s: %s", msg.Topic(), string(msg.Payload()))
		}
	}()
}

func (t *Transport) subscribe() {
	t.client.Subscribe("tradfri-raw/#", 1, t.onMessage)
	t.client.Subscribe("tradfri-reply/"+t.id, 1, t.onReply)
}
