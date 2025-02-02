package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	. "github.com/smartfog/nec-fogflow/common/ngsi"
)

var StartTime = time.Now()

func main() {
	configurationFile := flag.String("f", "config.json", "A configuration file")
	myPort := flag.Int("p", 8050, "the port of this agent")
	num := flag.Int("n", 2, "number of updates")

	flag.Parse()

	config := CreateConfig(*configurationFile)

	config.MyPort = *myPort

	startAgent(&config)
	sid := subscribe(&config)

	time.Sleep(10 * time.Second)

	StartTime = time.Now()

	// create the input entities
	for i := 1; i <= *num; i++ {
		createEntity(&config, i)
		StartTime = time.Now()

		time.Sleep(5 * time.Second)

		StartTime = time.Now()
		deleteEntity(&config, i)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c

	unsubscribe(&config, sid)

	// delete the input entities
	for i := 1; i <= *num; i++ {
		deleteEntity(&config, i)
	}

	time.Sleep(5 * time.Second)
}

func startAgent(config *Config) {
	agent := NGSIAgent{Port: config.MyPort}
	agent.Start()
	agent.SetContextNotifyHandler(HandleNotifyContext)
}

func HandleNotifyContext(notifyCtxReq *NotifyContextRequest) {
	INFO.Println("===========RECEIVE NOTIFY CONTEXT=========")
	INFO.Println(notifyCtxReq)

	for _, v := range notifyCtxReq.ContextResponses {
		ctxObj := CtxElement2Object(&(v.ContextElement))
		INFO.Println(ctxObj)

		var latency = (time.Now().UnixNano() - StartTime.UnixNano()) / int64(time.Millisecond)
		if ctxObj.IsEmpty() == true {
			// terminate task
			fmt.Printf("TERMINATE TASK: %s latency: %d \r\n", ctxObj.Entity.ID, latency)
		} else {
			// launch task
			fmt.Printf("LAUNCH TASK: %s latency: %d \r\n", ctxObj.Entity.ID, latency)
		}

		//currentTime := time.Now().UnixNano() / 1000000
		//latency := currentTime - ctxObj.Attributes["time"].Value.(int64)
		//num := ctxObj.Attributes["no"].Value.(int64)
		//fmt.Printf("No. %d, latency: %d \r\n", num, latency)

		/*
			for _, attr := range v.ContextElement.Attributes {
				INFO.Println(attr.Name)
				INFO.Println(attr.Type)
				INFO.Println(attr.Value)
				if attr.Name == "time" {
					INFO.Println("time to send: ", attr.Value)
					currentTime := time.Now().UnixNano() / 1000000
					INFO.Println("time to receive: ", currentTime)
					latency := currentTime - attr.Value.(int64)
					fmt.Println("latency: ", latency)
				}
			} */
	}
}

func subscribe(config *Config) string {
	subscription := SubscribeContextRequest{}

	newEntity := EntityId{}
	newEntity.Type = "Car"
	newEntity.IsPattern = true
	subscription.Entities = make([]EntityId, 0)
	subscription.Entities = append(subscription.Entities, newEntity)

	subscription.Reference = "http://" + config.MyIP + ":" + strconv.Itoa(config.MyPort)

	client := NGSI10Client{IoTBrokerURL: config.SubscribeBrokerURL}
	sid, err := client.SubscribeContext(&subscription, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(sid)

	return sid
}

func unsubscribe(config *Config, sid string) {
	// unsubscribe context
	fmt.Println("=============unsubscribe===================")
	client := NGSI10Client{IoTBrokerURL: config.SubscribeBrokerURL}
	err := client.UnsubscribeContext(sid)
	if err != nil {
		fmt.Println(err)
	}
}

func createEntity(config *Config, i int) {
	ctxObj := ContextObject{}

	ctxObj.Entity.ID = "Car." + strconv.Itoa(i)
	ctxObj.Entity.Type = "Car"
	ctxObj.Entity.IsPattern = false

	ctxObj.Attributes = make(map[string]ValueObject)
	ctxObj.Attributes["no"] = ValueObject{Type: "integer", Value: i}

	currentTime := time.Now().UnixNano() / 1000000
	ctxObj.Attributes["time"] = ValueObject{Type: "integer", Value: currentTime}

	client := NGSI10Client{IoTBrokerURL: config.UpdateBrokerURL}
	err := client.UpdateContextObject(&ctxObj)
	if err != nil {
		fmt.Println(err)
	}

	INFO.Println("create entity ", i)
}

func deleteEntity(config *Config, i int) {
	eid := EntityId{}
	eid.ID = "Car." + strconv.Itoa(i)

	client := NGSI10Client{IoTBrokerURL: config.UpdateBrokerURL}
	err := client.DeleteContext(&eid)
	if err != nil {
		fmt.Println(err)
	}

	INFO.Println("delete entity ", i)
}
