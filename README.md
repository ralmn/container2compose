# container2compose
Create docker compose file from docker running container

## Requirements

* Docker
* Docker-compose v1.20
* go v1.10

## Install

`go get github.com/ralmn/container2compose`

## Build

`go build -o container2compose`

## Usage

`container2compose container1 container2...`

### Options

`-output` (alias `-o`) : Set docker-compose file name