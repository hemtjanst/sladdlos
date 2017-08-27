package main

import (
	"flag"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hemtjanst/hemtjanst/messaging"
	"github.com/hemtjanst/hemtjanst/messaging/flagmqtt"
	"github.com/hemtjanst/sladdlos"
	"github.com/hemtjanst/sladdlos/tradfri"
	"github.com/hemtjanst/sladdlos/transport"
	"html/template"
	"log"
	"net/http"
	"time"
)

func main() {
	flag.Parse()

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
