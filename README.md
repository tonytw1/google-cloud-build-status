## Google Cloud Build status

Listens on a MQTT topic for Google Cloud Build JSON callbacks.

Parses the callback and emits the latest build status onto another MQTT topic as a group of booleans.

ie.
```
input       output
-------     ------
WORKING ->  QUEUED:false
            WORKING:true
            FAILED:false
            SUCCESS:false
```

This is used to drive one of our build status dashboards.
