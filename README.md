cowcheck
========

A microservice for checking the health of a Rancher node. 
Presents an HTTP interface on port `5050` for querying health status.
Will return `200 OK` when healthy and `503 Service Unavailable` when one
or more of its checks were unhealthy in the most recent evaluation cycle.

## How to use
Run this as a container on each Rancher host that runs your containers. It will assert 
basic functionality of the Rancher stack is working such as DNS, Metadata API. When combined
with a fleet management service such as AWS Auto Scale Groups or Google Cloud Deployment Manager 
you can replace nodes automatically when they fail. Alternatively you can just monitor and alert 
by polling the endpoint periodically.  
                                                              

## Configuration options

* `POLL_INTERVAL`: Time in seconds between evaluating checks
* `LOG_LEVL`: Level of logging verbosity

## Building

To build the binary:
`make build`

To create a Docker image: 
`make package`


## Running

`docker run --net=host -p 5050:5050 [image name]`

Or schedule with [Rancher](http://rancher.com) to run on all hosts as a 
[Global Service](https://docs.rancher.com/rancher/v1.6/en/cattle/scheduling/#global-service) (cattle) 
or [Daemonset](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)(kubernetes). 

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
