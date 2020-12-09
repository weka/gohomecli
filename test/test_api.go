package main

import (
	"fmt"
	"home.weka.io/cli"
)

const hostname = "api.fries.home.weka.io"
const token = "Onjb8DYnP9DDVSargu11TrvEvpQEwS"

func main() {
	client := cli.NewClient(hostname, token)
	fmt.Printf("%v", client)
}
