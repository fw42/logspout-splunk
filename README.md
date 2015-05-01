# logspout-splunk

Simple logspout module to forward Docker logstreams to either a Splunk input.

This is work in progress and not tested at scale. Use at own risk.

## Splunk

Put this in your Splunk "inputs.conf" (or add a new TCP input via the web UI):
```
[tcp://1234]
sourcetype = my_source_type
```

## Build the logspout-splunk container

Run `./build.sh`:
```
Sending build context to Docker daemon 90.11 kB
Sending build context to Docker daemon
Step 0 : FROM gliderlabs/logspout:master
...
Successfully built b356b141ddc2
```

## Start the logspout-splunk container

```
sudo docker run --env DEBUG=1 --name="logspout" \
	--volume=/var/run/docker.sock:/tmp/docker.sock \
	--publish=0.0.0.0:8002:80 b356b141ddc2
```

(use container id from above)

## Add a route for your applications

```
curl http://localhost:8002/routes -d '{
	"adapter": "splunk",
	"filter_sources": ["stdout" ,"stderr"],
	"address": "my-splunk-host:1234"
}'
```

## Add a route for a specific container name only

```
curl http://localhost:8002/routes -d '{
	"id": "unicorn",
	"adapter": "splunk",
	"filter_name": "*unicorn*",
	"filter_sources": ["stdout" ,"stderr"],
	"address": "my-splunk-host:1234"
}'
```
