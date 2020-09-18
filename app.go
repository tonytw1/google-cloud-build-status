package main

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/tkanos/gonfig"
	"os"
	"strconv"
	"time"
)

type Configuration struct {
	MqttUrl          string
	MqttTopic        string
	MqttMetricsTopic string
}

func main() {
	cloud_build_statuses := []string{"QUEUED", "WORKING", "FAILURE", "TIMEOUT", "CANCELLED", "SUCCESS"}

	configuration := Configuration{}
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		panic(err)
	}

	mqtt_url := configuration.MqttUrl
	mqtt_topic := configuration.MqttTopic
	mqtt_metrics_topic := configuration.MqttMetricsTopic

	var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		payload := msg.Payload()
		fmt.Printf("MSG: %s\n", payload)

		current_status := string(payload)

		for _, status := range cloud_build_statuses {
			is_current := status == current_status
			s := status + ":" + strconv.FormatBool(is_current)
			println(s)
			publish(client, mqtt_metrics_topic, s)
		}
	}

	var subscribeToCloudBuildTopic = func(client mqtt.Client) {
		println("Connected")
		println("Subscribing to:", mqtt_topic)
		if token := client.Subscribe(mqtt_topic, 0, messageHandler); token.Wait() && token.Error() != nil {
			fmt.Println("Subscription error", token.Error())
			os.Exit(1)
		}
	}

	opts := mqtt.NewClientOptions().AddBroker(mqtt_url)
	opts.SetKeepAlive(10 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.OnConnect = subscribeToCloudBuildTopic
	opts.SetCleanSession(true)

	println("Connecting to: ", mqtt_url)
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer c.Disconnect(250)

	for {
		time.Sleep(10 * time.Second)
	}
}

func publish(c mqtt.Client, topic string, message string) {
	token := c.Publish(topic, 0, false, message)
	token.Wait()
}
