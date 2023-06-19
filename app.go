package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/tkanos/gonfig"
	"log"
	"os"
	"strconv"
	"time"
)

type Configuration struct {
	MqttUrl          string
	MqttTopic        string
	MqttMetricsTopic string
}

type Attributes struct {
	BuildId string `json:"buildId"`
	Status  string `json:"status"`
}

type Message struct {
	MessageId   string     `json:"messageId"`
	PublishTime string     `json:"publishTime"`
	Attributes  Attributes `json:"attributes"`
	Data        string     `json:"data"`
}

type Push struct {
	Subscription string  `json:"subscription"`
	Message      Message `json:"message"`
	PublishTime  string  `json:"publishTime"`
}

type Payload struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

func main() {
	cloud_build_statuses := []string{"QUEUED", "WORKING", "FAILURE", "TIMEOUT", "CANCELLED", "SUCCESS"}
	latest_published_time := time.Now()

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
		log.Print(fmt.Sprintf("Received status: %s", payload))

		// Parse the JSON payload
		var push Push
		err = json.Unmarshal(payload, &push)
		if err != nil {
			log.Print("Could not parse message")
			return
		}

		// Extract the payload
		var base64Data = push.Message.Data
		var dataJson, err = base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			log.Print("Could not decode base64 payload")
		}

		var data Payload
		err = json.Unmarshal(dataJson, &data)

		current_status := data.Status
		// For our proposes a timeout is a failure
		if current_status == "TIMEOUT" {
			current_status = "FAILURE"
		}
		log.Print("Status is: ", current_status)

		log.Print("published time string is: " + push.PublishTime)
		var published_time, derr = time.Parse(time.RFC3339, push.PublishTime)
		if derr != nil {
			log.Print("Could not parse published date")
			return
		}
		log.Print("Published time was: ", published_time.String(), latest_published_time.String())
		if published_time.Before(latest_published_time) {
			log.Print("Ignoring out of order message: ", published_time.String())
		}
		latest_published_time = published_time

		for _, status := range cloud_build_statuses {
			is_current := status == current_status
			publishedMessage := "cloudbuild_" + status + ":" + strconv.FormatBool(is_current)
			publish(client, mqtt_metrics_topic, publishedMessage)
			log.Print(fmt.Sprintf("Published message: %s to topic %s", publishedMessage, mqtt_metrics_topic))
		}
	}

	var subscribeToCloudBuildTopic = func(client mqtt.Client) {
		println("Subscribing to:", mqtt_topic)
		if token := client.Subscribe(mqtt_topic, 0, messageHandler); token.Wait() && token.Error() != nil {
			log.Print(fmt.Sprintf("Subscription error", token.Error()))
			os.Exit(1)
		}
	}

	var logConnection mqtt.OnConnectHandler = func(client mqtt.Client) {
		log.Print("Connected")
		subscribeToCloudBuildTopic(client)
	}

	var logConnectionLost mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		log.Print("Connection lost")
	}

	var logReconnecting mqtt.ReconnectHandler = func(client mqtt.Client, opts *mqtt.ClientOptions) {
		log.Print("Reconnecting")
	}

	opts := mqtt.NewClientOptions().AddBroker(mqtt_url)
	opts.SetOnConnectHandler(logConnection)
	opts.SetConnectionLostHandler(logConnectionLost)
	opts.SetReconnectingHandler(logReconnecting)
	opts.SetCleanSession(true)
	opts.SetClientID("google-cloud-build-status")

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)

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
