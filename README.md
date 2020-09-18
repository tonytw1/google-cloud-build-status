Translates Google Cloud Build status pushes arriving on the input MQTT topic into a radio group of on/off messages on the output topic.

ie.
```
topic1      topic2
-------     ------
WORKING ->  QUEUED:false
            WORKING:true
            FAILED:false
            SUCCESS:false
```

This is used to drive one of our dashboards.

Dependencies

```
go get github.com/eclipse/paho.mqtt.golang

go get github.com/gorilla/websocket
go get golang.org/x/net/proxy
```
