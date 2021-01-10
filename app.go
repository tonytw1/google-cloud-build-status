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
		fmt.Printf("Recieved status: %s", payload)

		current_status := string(payload)

		// For our proposes a timeout is a failure
		if current_status == "TIMEOUT" {
			current_status = "FAILURE"
		}

		for _, status := range cloud_build_statuses {
			is_current := status == current_status
			publishedMessage := "cloudbuild_" + status + ":" + strconv.FormatBool(is_current)
			publish(client, mqtt_metrics_topic, publishedMessage)
			fmt.Printf("Published message: %s", publishedMessage)
		}
	}

	var subscribeToCloudBuildTopic = func(client mqtt.Client) {
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
	opts.SetClientID("google-cloud-build-status")

	println("Connecting to: ", mqtt_url)
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer c.Disconnect(250)

	for {
		time.Sleep(5 * time.Second)
	}
}

func publish(c mqtt.Client, topic string, message string) {
	token := c.Publish(topic, 0, false, message)
	token.WaitTimeout(time.Second * 1)
}
