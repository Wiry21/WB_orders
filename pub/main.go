package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	stan "github.com/nats-io/stan.go"
)

func main() {
	// Open our jsonFile
	jsonFile, err := os.Open("model.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened model.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	clusterID := "test-cluster" // nats cluster id
	clientID := "test-client-1"

	sc, err := stan.Connect(clusterID, clientID)
	if err != nil {
		log.Fatal(err)
	}

	sc.Publish("subj", byteValue)
	log.Printf("Published [%s] : \n", "subj")
}
