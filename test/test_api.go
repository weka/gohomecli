package main

import (
	"fmt"

	"github.com/weka/gohomecli/pkg/client"
)

const (
	hostname = "api.fries.home.weka.io"
	token    = "Onjb8DYnP9DDVSargu11TrvEvpQEwS"
)

func main() {
	client := client.NewClient(hostname, token)
	fmt.Printf("%v", client)
}
