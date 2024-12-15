package cluster

import (
	"fmt"
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
	nodes map[string]KademliaNode
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

	delete(bucket.nodes, nodeId)
}

func (dht *KademliaDHT) RemoveAll() {
	for bucket := range dht.buckets {
		dht.buckets[bucket].nodes = make(map[string]KademliaNode)
	}
}

func (dht *KademliaDHT) Store(hostname, port string, state KademliaNodeState, lastPing time.Time) string {
	newNodeId := Hash(hostname + port)
	newNode := KademliaNode{nodeId: newNodeId, hostname: hostname, port: port, state: state, lastPing: lastPing}
	distance := getXORDistance(dht.nodeId, newNodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	// Idempotent put
	if _, exists := bucket.nodes[newNodeId]; exists {
		bucket.nodes[newNodeId] = newNode
		return newNodeId
	}

	// Handle case when adding a new node to a non-full bucket
	if len(bucket.nodes) < bucketSize {
		bucket.nodes[newNodeId] = newNode
	} else {
		// Handle the case when the bucket is full
	}

	return newNodeId
}

func (dht *KademliaDHT) Find(nodeId string) []KademliaNode {
	distance := getXORDistance(dht.nodeId, nodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	closestNodes := make([]KademliaNode, 0)
	for _, node := range bucket.nodes {
		closestNodes = append(closestNodes, node)
		if len(closestNodes) >= bucketSize {
			break
		}
	}

	return closestNodes
}

func (dht *KademliaDHT) FindStringified(nodeId string) string {
	distance := getXORDistance(dht.nodeId, nodeId)
	bucketIndex := getBucketIndex(distance)
	bucket := dht.buckets[bucketIndex]

	closestNodes := ""
	for _, node := range bucket.nodes {
		closestNodes = fmt.Sprintf("%s%s, ", closestNodes, node.nodeId)
	}

	return closestNodes
}

func (dht *KademliaDHT) RandomNode() *KademliaNode {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		bucketIndex := r.Intn(len(dht.buckets))
		bucket := dht.buckets[bucketIndex]

		if len(bucket.nodes) > 0 {
			nodes := make([]KademliaNode, 0, len(bucket.nodes))
			for _, node := range bucket.nodes {
				nodes = append(nodes, node)
			}

			randomIndex := r.Intn(len(nodes))
			return &nodes[randomIndex]
		}
	}
}

// Return a collection of nodes in the random bucket
func (dht *KademliaDHT) RandomNodesBatch() []KademliaNode {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		bucketIndex := r.Intn(len(dht.buckets))
		bucket := dht.buckets[bucketIndex]

		if len(bucket.nodes) > 0 {
			nodes := make([]KademliaNode, 0, len(bucket.nodes))
			for _, node := range bucket.nodes {
				nodes = append(nodes, node)
			}

			return nodes
		}
	}
}

// Process of selecting a rendezvous node based on a key
func (dht *KademliaDHT) RendezvousNode(key string) *KademliaNode {
	selectedNode := &KademliaNode{}
	maxWeight := new(big.Int)

	for _, bucket := range dht.buckets {
		for _, node := range bucket.nodes {
			// Skip unhealthy nodes
			if node.state != NodeHealthy {
				continue
			}

			// Calculate the weight for this node
			weight := CalculateWeight(key, node.nodeId)

			// Compare the weight and select the node with the highest weight
			if weight == nil || weight.Cmp(maxWeight) > 0 {
				if node.nodeId == dht.nodeId {
					continue
				}
				maxWeight = weight
				selectedNode = &node
			}
		}
	}

	return selectedNode
}

// Select a batch of rendezvous nodes based on a key
func (dht *KademliaDHT) RendezvousNodeBatch(key string) []KademliaNode {
	scoredNodes := make([]ScoredNode, 0)

	for _, bucket := range dht.buckets {
		for _, node := range bucket.nodes {
			if node.state != NodeHealthy {
				continue
			}

			// Calculate the weight for this node
			weight := CalculateWeight(key, node.nodeId)
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

	return selectedNodes
}

func (node *KademliaNode) NodeId() string {
	return node.nodeId
}
