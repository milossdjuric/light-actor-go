package cluster

import (
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"sort"
	"time"
)

// maximum bucket size
const bucketSize = 5

// length of node IDs, 160 bits for SHA-1 hash function
const bits = 160

type KademliaDHT struct {
	nodeId   string
	hostname string
	port     string
	buckets  []*KademliaBucket
}

type KademliaNode struct {
	nodeId   string
	hostname string
	port     string
	lastPing time.Time
	state    KademliaNodeState
}

type KademliaNodeState int

type ScoredNode struct {
	node   *KademliaNode
	weight *big.Int
}

const (
	NodeHealthy KademliaNodeState = iota
	NodeSuspect
	NodeDisowned
)

type KademliaBucket struct {
	nodes map[string]KademliaNode // Map of nodeId to KademliaNode
}

func NewKademliaDHT(hostname, port string) *KademliaDHT {
	nodeId := Hash(hostname + port)
	dht := &KademliaDHT{
		nodeId:   nodeId,
		hostname: hostname,
		port:     port,
		buckets:  make([]*KademliaBucket, bits),
	}

	for i := 0; i < bits; i++ {
		dht.buckets[i] = &KademliaBucket{
			nodes: make(map[string]KademliaNode),
		}
	}
	return dht
}

func (dht *KademliaDHT) Remove(hostname, port string) {
	nodeId := Hash(hostname + port)
	distance := getXORDistance(dht.nodeId, nodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	// Log the removal attempt
	log.Printf("[Remove] Removing node with ID: %s from bucket %d (distance: %d)\n", nodeId, bucketIndex, distance)

	// Perform the removal
	delete(bucket.nodes, nodeId)

	// Log the result
	log.Printf("[Remove] Node with ID: %s removed from bucket %d\n", nodeId, bucketIndex)
}

func (dht *KademliaDHT) Store(hostname, port string, state KademliaNodeState, lastPing time.Time) string {
	newNodeId := Hash(hostname + port)
	newNode := KademliaNode{nodeId: newNodeId, hostname: hostname, port: port, state: state, lastPing: lastPing}
	distance := getXORDistance(dht.nodeId, newNodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	// Log the store attempt
	log.Printf("[Store] Storing node with ID: %s in bucket %d (distance: %d)\n", newNodeId, bucketIndex, distance)

	// Idempotent put
	if _, exists := bucket.nodes[newNodeId]; exists {
		// Log the update of an existing node
		log.Printf("[Store] Node with ID: %s already exists, updating its state\n", newNodeId)
		bucket.nodes[newNodeId] = newNode
		return newNodeId
	}

	// Handle case when adding a new node to a non-full bucket
	if len(bucket.nodes) < bucketSize {
		// Log the addition of a new node to the bucket
		log.Printf("[Store] Adding new node with ID: %s to bucket %d\n", newNodeId, bucketIndex)
		bucket.nodes[newNodeId] = newNode
	} else {
		// Handle the case when the bucket is full
		log.Printf("[Store] Bucket %d is full, cannot add new node with ID: %s\n", bucketIndex, newNodeId)
	}

	return newNodeId
}

func (dht *KademliaDHT) Find(nodeId string) []KademliaNode {
	distance := getXORDistance(dht.nodeId, nodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	// Log the find attempt
	log.Printf("[Find] Finding closest nodes to node ID: %s in bucket %d (distance: %d)\n", nodeId, bucketIndex, distance)

	closestNodes := make([]KademliaNode, 0)
	for _, node := range bucket.nodes {
		closestNodes = append(closestNodes, node)
		if len(closestNodes) >= bucketSize {
			break
		}
	}

	// Log the found nodes
	log.Printf("[Find] Found %d closest nodes for node ID: %s\n", len(closestNodes), nodeId)

	return closestNodes
}

func (dht *KademliaDHT) FindStringified(nodeId string) string {
	distance := getXORDistance(dht.nodeId, nodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	// Log the find stringified attempt
	log.Printf("[FindStringified] Finding closest nodes to node ID: %s in bucket %d (distance: %d)\n", nodeId, bucketIndex, distance)

	closestNodes := ""
	for _, node := range bucket.nodes {
		closestNodes = fmt.Sprintf("%s%s, ", closestNodes, node.nodeId)
	}

	// Log the result
	log.Printf("[FindStringified] Found closest nodes for node ID: %s: %s\n", nodeId, closestNodes)

	return closestNodes
}

func (dht *KademliaDHT) RandomNode() *KademliaNode {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Log the start of the random node selection process
	log.Println("[RandomNode] Starting random node selection...")

	for {
		bucketIndex := r.Intn(len(dht.buckets))
		bucket := dht.buckets[bucketIndex]

		// Log the selected bucket index
		log.Printf("[RandomNode] Selected bucket index: %d\n", bucketIndex)

		if len(bucket.nodes) > 0 {
			nodes := make([]KademliaNode, 0, len(bucket.nodes))
			for _, node := range bucket.nodes {
				nodes = append(nodes, node)
			}

			// Log the number of nodes in the selected bucket
			log.Printf("[RandomNode] Found %d nodes in the selected bucket\n", len(nodes))

			randomIndex := r.Intn(len(nodes))
			// Log the selected random node ID
			log.Printf("[RandomNode] Selected random node with ID: %s\n", nodes[randomIndex].nodeId)
			return &nodes[randomIndex]
		}
	}
	// In case no node is found (shouldn't happen in normal conditions)
	log.Println("[RandomNode] No node found, returning empty node")
	return &KademliaNode{}
}

// Return a collection of nodes in the random bucket
func (dht *KademliaDHT) RandomNodesBatch() []KademliaNode {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Log the start of the batch random node selection process
	log.Println("[RandomNodesBatch] Starting random batch node selection...")

	for {
		bucketIndex := r.Intn(len(dht.buckets))
		bucket := dht.buckets[bucketIndex]

		// Log the selected bucket index
		log.Printf("[RandomNodesBatch] Selected bucket index: %d\n", bucketIndex)

		if len(bucket.nodes) > 0 {
			nodes := make([]KademliaNode, 0, len(bucket.nodes))
			for _, node := range bucket.nodes {
				nodes = append(nodes, node)
			}

			// Log the number of nodes found in the selected bucket
			log.Printf("[RandomNodesBatch] Found %d nodes in the selected bucket\n", len(nodes))

			return nodes
		}
	}
}

// Process of selecting a rendezvous node based on a key
func (dht *KademliaDHT) RendezvousNode(key string) *KademliaNode {
	log.Printf("[RendezvousNode] Starting rendezvous process for key: %s\n", key)
	selectedNode := &KademliaNode{}
	maxWeight := new(big.Int)

	// Iterate through the buckets to evaluate nodes
	for _, bucket := range dht.buckets {
		for _, node := range bucket.nodes {
			// Log the current node being evaluated
			log.Printf("[RendezvousNode] Evaluating node ID: %s, State: %s\n", node.nodeId, node.state)

			// Skip unhealthy nodes
			if node.state != NodeHealthy {
				log.Printf("[RendezvousNode] Node %s is not healthy, skipping...\n", node.nodeId)
				continue
			}

			// Calculate the weight for this node
			weight := CalculateWeight(key, node.nodeId)

			// Log the calculated weight
			if weight != nil {
				log.Printf("[RendezvousNode] Calculated weight for node %s: %s\n", node.nodeId, weight.String())
			} else {
				log.Printf("[RendezvousNode] Weight calculation returned nil for node %s\n", node.nodeId)
			}

			// Compare the weight and select the node with the highest weight
			if weight == nil || weight.Cmp(maxWeight) > 0 {
				log.Printf("[RendezvousNode] Node %s selected with weight: %s\n", node.nodeId, weight.String())
				if node.nodeId == dht.nodeId {
					continue
				}
				maxWeight = weight
				selectedNode = &node
			}
		}
	}

	// Log the final selected node
	log.Printf("[RendezvousNode] Final selected node: %s with weight: %s\n", selectedNode.nodeId, maxWeight.String())

	return selectedNode
}

// Select a batch of rendezvous nodes based on a key
func (dht *KademliaDHT) RendezvousNodeBatch(key string) []KademliaNode {
	log.Printf("[RendezvousNodeBatch] Starting batch rendezvous process for key: %s\n", key)
	scoredNodes := make([]ScoredNode, 0)

	// Iterate through the buckets to evaluate nodes
	for _, bucket := range dht.buckets {
		for _, node := range bucket.nodes {
			if node.state != NodeHealthy {
				continue
			}

			// Calculate the weight for this node
			weight := CalculateWeight(key, node.nodeId)

			// Log the node and its weight
			log.Printf("[RendezvousNodeBatch] Node ID: %s, Weight: %s\n", node.nodeId, weight.String())

			scoredNodes = append(scoredNodes, ScoredNode{node: &node, weight: weight})
		}
	}

	// Sort the nodes based on weight
	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].weight.Cmp(scoredNodes[j].weight) > 0
	})

	// Select the top nodes based on score
	selectedNodes := []KademliaNode{}
	for i := 0; i < bucketSize && i < len(scoredNodes); i++ {
		selectedNodes = append(selectedNodes, *scoredNodes[i].node)
	}

	// Log the selected nodes
	log.Printf("[RendezvousNodeBatch] Selected %d nodes for key: %s\n", len(selectedNodes), key)

	return selectedNodes
}

// NodeId method for the KademliaNode struct
func (node *KademliaNode) NodeId() string {
	// Log the retrieval of the node ID
	log.Printf("[NodeId] Retrieving node ID: %s\n", node.nodeId)
	return node.nodeId
}
