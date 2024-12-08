package main

import (
	"fmt"
	"light-actor-go/actor"
	"light-actor-go/cluster"
	"light-actor-go/remote"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	InitialNodeAddress = "127.0.0.1:8100"
	TotalNodes         = 150
	LeaveProbability   = 1 // Probability that a node will leave each cycle (100%)
)

func main() {
	// Start the initial cluster node
	remoteConfig := remote.NewRemoteConfig(InitialNodeAddress)
	initialNode := cluster.NewClusterNode(actor.NewActorSystem(), *remoteConfig)
	initialNode.Start()
	fmt.Println("[CLUSTER] Initial node started at:", InitialNodeAddress)

	// Create and start additional nodes
	var nodes []*cluster.ClusterNode
	for i := 1; i < 5; i++ {
		nodeAddr := "127.0.0.1:" + strconv.Itoa(8100+i)
		remoteConfig := remote.NewRemoteConfig(nodeAddr)
		node := cluster.NewClusterNode(actor.NewActorSystem(), *remoteConfig)
		node.Start()
		nodes = append(nodes, node)

		// Join the new node to the cluster
		node.JoinCluster("127.0.0.1", "8100")
		log.Println("[CLUSTER] Node %s joined the cluster\n", nodeAddr)

	}

	time.Sleep(15 * time.Second)

	for i := 5; i <= TotalNodes; i++ {
		nodeAddr := "127.0.0.1:" + strconv.Itoa(8100+i)
		remoteConfig := remote.NewRemoteConfig(nodeAddr)
		node := cluster.NewClusterNode(actor.NewActorSystem(), *remoteConfig)
		node.Start()
		nodes = append(nodes, node)

		// Join the new node to the cluster
		node.JoinCluster("127.0.0.1", "8100")
		fmt.Printf("[CLUSTER] Node %s joined the cluster\n", nodeAddr)

		// Allow nodes to initialize
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("[CLUSTER] Total nodes running: %d\n", len(nodes)+1)

	// Periodically check and remove random nodes
	go func() {
		for {
			time.Sleep(500 * time.Millisecond) // Check every second
			if rand.Float64() < LeaveProbability && len(nodes) > 0 {
				// Select a random node to leave
				nodeIndex := rand.Intn(len(nodes))
				if nodes[nodeIndex] == nil {
					continue
				}
				nodeToLeave := nodes[nodeIndex]

				// Have the node leave the cluster
				nodeToLeave.LeaveCluster("127.0.0.1", "8100")
				fmt.Printf("[CLUSTER] Node %s left the cluster\n", nodeToLeave)

				// Remove the node from the list
				nodes = append(nodes[:nodeIndex], nodes[nodeIndex+1:]...)
			}
		}
	}()

	// Handle shutdown signals gracefully
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	fmt.Println("[CLUSTER] Shutting down cluster nodes")
}
