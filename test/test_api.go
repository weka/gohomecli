package main

import (
	"fmt"
	"github.com/weka/gohomecli/cli"
)

const hostname = "api.fries.home.weka.io"
const token = "Onjb8DYnP9DDVSargu11TrvEvpQEwS"

func main() {
	client := cli.NewClient(hostname, token)
	fmt.Printf("%v", client)
}
