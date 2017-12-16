package main

import (
	"flag"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/hemtjanst/hemtjanst/messaging"
	"github.com/hemtjanst/hemtjanst/messaging/flagmqtt"
	"github.com/hemtjanst/sladdlos"
	"github.com/hemtjanst/sladdlos/tradfri"
	"github.com/hemtjanst/sladdlos/transport"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	cleanUpHemtjanst = flag.Bool("hemtjanst.cleanup", false, "Clean up Hemtjänst MQTT Topics")
	cleanUpTradfri   = flag.Bool("tradfri.cleanup", false, "Clean up Trådfri MQTT Topics")
)

func main() {
	flag.Parse()

	if *cleanUpHemtjanst || *cleanUpTradfri {
		clean()
		return
	}

	id := flagmqtt.NewUniqueIdentifier()

	tr := transport.NewTransport(id)
	tree := tradfri.NewTree(tr)
	tr.SetTree(tree)

	ht := sladdlos.NewHemtjanstClient(tree, id)

	var messenger messaging.PublishSubscriber

	mqClient, err := flagmqtt.NewPersistentMqtt(flagmqtt.ClientConfig{
		WillTopic:   "leave",
		WillPayload: id,
		WillRetain:  false,
		WillQoS:     1,
		ClientID:    "sladdlos-" + id,
		OnConnectHandler: func(client mqtt.Client) {
			if messenger == nil {
				messenger = messaging.NewMQTTMessenger(client)
			}
			tr.OnConnect(client)
			ht.OnConnect(messenger)
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Connecting to MQTT")
	token := mqClient.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Fatal(err)
	}
	log.Print("Connected")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f := "index.tmpl"
		fm := template.FuncMap{
			"GetDevice": func(aId int) *tradfri.Accessory {
				if dev, ok := tree.Devices[aId]; ok {
					return dev
				}
				return nil
			},
		}
		t, err := template.New(f).
			Funcs(fm).
			ParseFiles("./templates/" + f)
		if err != nil {
			log.Print(err)
		}
		t.Execute(w, tree)
	})
	h := &http.Server{
		Addr:              ":7995",
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Fatal(h.ListenAndServe())
}

func clean() {
	exit := make(chan bool)
	id := flagmqtt.NewUniqueIdentifier()
	mqClient, err := flagmqtt.NewPersistentMqtt(flagmqtt.ClientConfig{
		ClientID: "sladdlos-cleaner-" + id,
		OnConnectHandler: func(client mqtt.Client) {

		},
	})

	log.Print("Connecting to MQTT")
	token := mqClient.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Fatal(err)
	}
	log.Print("Connected")

	if *cleanUpTradfri {
		mqClient.Subscribe("tradfri-raw/#", 1, func(client mqtt.Client, message mqtt.Message) {
			if message.Retained() {
				log.Printf("Deleting contents of topic %s", message.Topic())
				client.Publish(message.Topic(), 1, true, []byte{})
			}
		})
	}

	if *cleanUpHemtjanst {
		mqClient.Subscribe("announce/light/+", 1, func(client mqtt.Client, message mqtt.Message) {
			if message.Retained() {
				sp := strings.Split(message.Topic(), "/")
				if len(sp) == 3 && strings.Index(sp[2], "grp-") == 0 || strings.Index(sp[2], "bulb-") == 0 {
					log.Printf("Deleting contents of topic %s", message.Topic())
					client.Publish(message.Topic(), 1, true, []byte{})
				}
			}
		})
		mqClient.Subscribe("light/+/+/get", 1, func(client mqtt.Client, message mqtt.Message) {
			if message.Retained() {
				sp := strings.Split(message.Topic(), "/")
				if len(sp) == 4 && strings.Index(sp[1], "grp-") == 0 || strings.Index(sp[1], "bulb-") == 0 {
					log.Printf("Deleting contents of topic %s", message.Topic())
					client.Publish(message.Topic(), 1, true, []byte{})
				}
			}
		})
	}

	<-exit
}
