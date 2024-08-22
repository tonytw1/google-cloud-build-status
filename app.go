package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/tkanos/gonfig"
	"log"
	"os"
	"os/signal"
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

type Callback struct {
	Subscription string  `json:"subscription"`
	Message      Message `json:"message"`
}

type Data struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

func main() {
	cloudBuildStatuses := []string{"QUEUED", "WORKING", "FAILURE", "TIMEOUT", "CANCELLED", "SUCCESS"}
	latestPublishedTime := time.Now()

	configuration := Configuration{}
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		panic(err)
	}

	mqttUrl := configuration.MqttUrl
	mqttTopic := configuration.MqttTopic
	mqttMetricsTopic := configuration.MqttMetricsTopic

	var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		payload := msg.Payload()
		log.Print(fmt.Sprintf("Received status: %s", payload))

		// Parse the JSON payload
		var push Callback
		err := json.Unmarshal(payload, &push)
		if err != nil {
			log.Print("Could not parse message")
			return
		}

		// Extract the payload
		var base64Data = push.Message.Data
		dataJson, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			log.Print("Could not decode base64 payload")
		}

		var data Data
		err = json.Unmarshal(dataJson, &data)

		currentStatus := data.Status
		// For our proposes a timeout is a failure
		if currentStatus == "TIMEOUT" {
			currentStatus = "FAILURE"
		}
		log.Print("Status is: ", currentStatus)

		log.Print("published time string is: " + push.Message.PublishTime)
		publishTime, err := time.Parse(time.RFC3339, push.Message.PublishTime)
		if err != nil {
			log.Print("Could not parse published date")
			return
		}
		log.Print("Published time was: ", publishTime.String(), latestPublishedTime.String())
		if publishTime.Before(latestPublishedTime) {
			log.Print("Ignoring out of order message: ", publishTime.String())
			return
		}

		latestPublishedTime = publishTime

		for _, status := range cloudBuildStatuses {
			isCurrent := status == currentStatus
			publishedMessage := status + ":" + strconv.FormatBool(isCurrent)
			publish(client, mqttMetricsTopic, publishedMessage)
			log.Print(fmt.Sprintf("Published message: %s to topic %s", publishedMessage, mqttMetricsTopic))
		}
	}

	var subscribeToCloudBuildTopic = func(client mqtt.Client) {
		log.Print("Subscribing to:", mqttTopic)
		if token := client.Subscribe(mqttTopic, 0, messageHandler); token.Wait() && token.Error() != nil {
			log.Print(fmt.Sprintf("Subscription error: %s", token.Error()))
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

	opts := mqtt.NewClientOptions().AddBroker(mqttUrl)
	opts.SetOnConnectHandler(logConnection)
	opts.SetConnectionLostHandler(logConnectionLost)
	opts.SetReconnectingHandler(logReconnecting)
	opts.SetCleanSession(true)
	opts.SetClientID("google-cloud-build-status")

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)

	log.Print("Connecting to: ", mqttUrl)
	c := mqtt.NewClient(opts)

	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer c.Disconnect(250)

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt)
	<-exitSignal
}

func publish(c mqtt.Client, topic string, message string) {
	token := c.Publish(topic, 0, false, message)
	token.WaitTimeout(time.Second * 1)
}
