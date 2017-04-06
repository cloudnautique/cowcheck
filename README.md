cowcheck
========

A microservice for checking the health of a Rancher node. 
Presents an HTTP interface on port `5050` for querying health status.
Will return `200 OK` when healthy and `503 Service Unavailable` when one
or more of its checks were unhealthy in the most recent evaluation cycle.

## Building

`make`


## Running

`./bin/cowcheck`

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
