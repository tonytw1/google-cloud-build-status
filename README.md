Translates Google Cloud Build status pushes arriving on a MQTT topic into a radio group of on/off messages.

ie.
```
WORKING ->  QUEUED:false
            WORKING:true
            FAILED:false
            SUCCESS:false
```

This is used to drive one of our dashboards.

