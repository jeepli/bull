package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ikenchina/bull/democlient"
)

// test
// run consul : consul agent --dev
func main() {
	go democlient.TestServer("", 40001)
	go democlient.TestServer("", 40002)

	time.Sleep(1 * time.Second)

	cc := &democlient.Client{}
	err := cc.Build()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		ret, err := cc.Add(int32(i), 2)
		fmt.Println(ret, err)
		time.Sleep(2 * time.Second)
	}
}
