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
	nodes := []*cluster.ClusterNode{
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8010")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8011")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8012")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8013")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8014")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8015")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8016")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8017")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8018")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8019")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8020")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8021")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8022")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8023")),
		cluster.NewClusterNode(actor.NewActorSystem(), *remote.NewRemoteConfig("127.0.0.1:8024")),
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	// Wait for nodes to start
	time.Sleep(5 * time.Second)

	// Join all nodes to the cluster managed by Node 1
	for i := 0; i < len(nodes); i++ {
		nodes[i].JoinCluster("127.0.0.1", "8010")
	}

	time.Sleep(1 * time.Second)

	for i := len(nodes) - 1; i >= 0; i-- {
		time.Sleep(1 * time.Second)
		nodes[i].LeaveCluster("127.0.0.1", "8010")
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	fmt.Println("Shutting down cluster nodes")
}
