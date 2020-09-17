Translates Google Cloud Build status pushes arriving on our MQTT topic into a radio group of on/offs messages.

ie.
```
WORKING ->  QUEUED:false
            WORKING:true
            FAILED:false
            SUCCESS:false
```

This is used to drive one of or dashboards.

