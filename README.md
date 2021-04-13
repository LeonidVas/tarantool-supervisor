<a href="http://tarantool.org">
  <img src="https://avatars2.githubusercontent.com/u/2344919?v=2&s=250" align="right">
</a>

# Tvisor

A service that spawns and manages tarantool instances on a given machine.

## Table of contents
* [Getting started](#getting-started)
  * [Prerequisites](#prerequisites)
  * [Download / Build](#download-/-build)
  * [Run tests](#run-tests)
  * [Usage](#usage)
* [Documentation](#documentation)
* [Configuration](#configuration)
* [Args](#args)
* [API](#api)
  * [Start](#start)
  * [Stop](#stop)
  * [Status](#status)
  * [List](#list)
* [Caution](#caution)

## Getting started

### Prerequisites

 * [Go](https://golang.org/doc/install)
 * [Mage](https://magefile.org/)
 * [Testify](https://github.com/stretchr/testify)
 * [Mapstructure](https://github.com/mitchellh/mapstructure)
 * [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

To run tests:
 * [Python3](https://www.python.org/downloads/)

### Download / Build

Using [go get](https://golang.org/cmd/go/#hdr-Legacy_GOPATH_go_get)(GOPATH):
``` bash
go get -u -d github.com/LeonidVas/tvisor
cd $GOPATH/src/github.com/LeonidVas/tvisor
mage build
```

Using Go Modules:
``` bash
git clone https://github.com/LeonidVas/tvisor.git
cd tvisor
mage build
```

### Run tests
``` bash
mage -v test
```

### Usage

See [Args](#args), [Configuration](#configuration) and [API](#api) sections
belows for more details.

Run:
``` bash
./tvisor --cfg="cfg.json" --addr="127.0.0.1:8080"
```

Start an instance:
``` bash
curl --header "Content-Type: application/json" --request POST \
 --data '{"command_name":"start", "params":{"name": "test_instance", "env":["MYVAR=true"]}}' \
 http://127.0.0.1:8080/instance

{"id":1}
```

Stop the instance:
``` bash
curl --header "Content-Type: application/json" --request POST \
 --data '{"command_name":"stop", "params":{"id": 1, "force": true}}' \
 http://127.0.0.1:8080/instance

{"done":true}
```

Get instance status:
``` bash
curl --header "Content-Type: application/json" --request POST \
 --data '{"command_name":"status","params":{"id": 1}}' \
 http://127.0.0.1:8080/instance

{"status":{"name":"test_instance","status":"running","pid":741739,"env":["MYVAR=true"]}}
```

Get a list of instances:
``` bash
curl --header "Content-Type: application/json" --request POST \
 --data '{"command_name":"list"}' \
 http://127.0.0.1:8080/instance

{"instances":{"1":{"name":"test_instance","status":"running","pid":741739,"env":["MYVAR=true"]}}}
```

Graceful terminate by sending SIGINT / SIGTERM:
```bash
2021/03/23 17:56:25 The service has been terminated.
```

## Documentation

To read the documentation use:
* [go doc](https://golang.org/cmd/go/#hdr-Show_documentation_for_package_or_symbol)
* [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc)

## Configuration

For configuration, JSON config is used with the following fields:
* `instances_dir`(string) - directory that stores executable files with `.lua`
 extension for running Instances. Default: `/etc/tarantool/tvisor/instances`
* `termination_timeout`(number) - time (in seconds) to wait for the Instance to
 terminate correctly. After this timeout expires, the SIGKILL signal will be used
 to stop the instance if the force option is true, else an error will be returned.
 Default: `30`

## Args

Arguments of tvisor:
* `-cfg`(string) - path to Tvisor config. Default: `cfg.json`
* `-addr`(string) - address to start the HTTP server(host:port).
 Default: `127.0.0.1:8080`
* `-help` - help.

## API

The HTTP API is used to interact with Tvisor. The request uses JSON
describing the command sent by the POST method. Response - JSON containing the
result of the command.

Command structure:
```json
{
  "command_name": "command_name",
  "params": {
    "param_name1": "a",
    "param_name2": true,
    "param_name3": 2,
    "param_name4": ["a", "b", "c"]
  }
}
```

Now the following commands are available: `start`, `stop`, `status`, `list`.

### Start
Run an instance by name.

Name: `start`

Parametrs:
* `name`(string) - name of instance to run (without `.lua` extension). The
 instance to start will be searched for in the `inst_dir` directory.
* `restartable`(bool) - the setting is responsible for the need to restart the
 instance on failure. Default: `true`.
* `env`(array of strings) - an array of environment variables that will be
 used when starting the instance.

Example:
```json
{
  "command_name": "start",
  "params": {
    "name": "test_instance",
    "restartable": true,
    "env": [
      "MYVAR=true"
    ]
  }
}
```

Response:
* `id`(number) - instance ID. 0 is incorrect.

Example:
```json
{
  "id": 1
}
```

### Stop
Stop the instance by ID.

Name: `stop`

Parametrs:
* `id`(number) - instance ID. 0 is incorrect.
* `force`(bool) - if `true`in case of a graceful termination (`SIGTERM`) of the
 instance fails, a forced termination (`SIGKILL`) will be used. Default: true.

Example:
```json
{
  "command_name": "stop",
  "params": {
    "id": 1,
    "force": true
  }
}
```

Response:
* `done`(bool) - `true` if successful.

Example:
```json
{
  "done": true
}
```

### Status
Returns the status of the instance by ID.

Name: `status`

Parametrs:
* `id`(number) - instance ID. 0 is incorrect.

Example:
```json
{
  "command_name": "status",
  "params": {
    "id": 1,
  }
}
```

Response:
* `status`(JSON Obj) - an object describing the status of the instance.
  * `name`(string) - the name of the instance.
  * `status`(string) - describes the status of the instance.
    Available values: `running` / `terminated`.
  * `pid`(number) - a process ID.
  * `restartable`(bool) - the setting is responsible for the need to restart the
    instance on failure.
  * `env`(array of strings) - describes the environment settled by a client.

Example:
```json
{
  "status": {
    "name": "test_instance",
    "status": "running",
    "pid": 741739,
    "restartable": true,
    "env": [
      "MYVAR=true"
    ]
  }
}
```

### List
Returns a list of instances.

Name: `list`

Example:
```json
{
  "command_name": "list"
}
```

Response:
* `instances`(array of `status` objs) - map of an instance ID to current status.

Example:
```json
{
  "instances": {
    "1": {
      "name": "test_instance",
      "status": "running",
      "pid": 741739,
      "restartable": true,
      "env": [
        "MYVAR=true"
      ]
    }
  }
}
```

## Caution

This service is in early alpha.
