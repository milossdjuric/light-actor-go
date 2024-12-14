package main

import (
	"fmt"
	"light-actor-go/actor"
	"light-actor-go/cluster"
	"light-actor-go/remote"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	// Create node configurations

	node1 := cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8060"))
	node2 := cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8061"))

	node1.Start()
	node2.Start()

	time.Sleep(3 * time.Second)

	node1.JoinCluster("127.0.0.1", "8060")
	node2.JoinCluster("127.0.0.1", "8060")

	time.Sleep(2 * time.Second)

	node2.LeaveCluster("127.0.0.1", "8060")
	// go simulateNodeFailures(nodes)

	// Handle shutdown signals to gracefully terminate the cluster
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	fmt.Println("Shutting down cluster nodes")
}
