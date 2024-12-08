package cluster

import (
	"fmt"
	"light-actor-go/actor"
	"light-actor-go/remote"
	"log"
	"math/big"
	"time"

	"google.golang.org/protobuf/types/known/anypb"
)

const (
	gossipThreshold = 10
)

type ClusterActor struct {
	id         string
	hostname   string
	port       string
	dht        *KademliaDHT
	ticker     *time.Ticker
	remote     *remote.Remote
	inCluster  bool
	stopGossip chan struct{}
}

func NewClusterActor(id, hostname, port string, remote *remote.Remote) *ClusterActor {
	dht := NewKademliaDHT(hostname, port)

	fmt.Printf("[DEBUG] Creating ClusterActor with ID: %s, Hostname: %s, Port: %s\n", id, hostname, port)
	return &ClusterActor{
		id:        id,
		hostname:  hostname,
		port:      port,
		dht:       dht,
		remote:    remote,
		inCluster: false,
	}
}

func (c *ClusterActor) Receive(ctx actor.ActorContext) {
	fmt.Printf("[DEBUG] ClusterActor %s Received message type: %T\n", c.id, ctx.Message())

	// Check if the message is of type proto.Any (dynamic protobuf message)
	switch msg := ctx.Message().(type) {
	case actor.SystemMessage:
		switch msg.Type {
		case actor.SystemMessageStart:
			fmt.Printf("[DEBUG] ClusterActor %s Received SystemMessageStart, starting cluster actor.\n", c.id)
		case actor.SystemMessageRestart:
			fmt.Printf("[DEBUG] ClusterActor %s Received SystemMessageRestart, restarting cluster actor.\n", c.id)
		case actor.SystemMessageStop:
			fmt.Printf("[DEBUG] ClusterActor %s Received SystemMessageStop, stopping gossip.", c.id)
			c.stopGossip <- struct{}{}
		case actor.SystemMessageGracefulStop:
			fmt.Printf("[DEBUG] ClusterActor %s Received SystemMessageGracefulStop, stopping gossip.", c.id)
			c.stopGossip <- struct{}{}
		}
	case *anypb.Any:
		// Try to unmarshal the Any type message into a specific protobuf message
		if err := c.handleProtoAny(ctx, msg); err != nil {
			fmt.Printf("[ERROR] ClusterActor %s Failed to unmarshal proto.Any: %v\n", c.id, err)
		}
	case *JoinCluster:
		fmt.Printf("[DEBUG] ClusterActor %s Received JoinCluster, for cluster %s:%s\n", c.id, msg.Hostname, msg.Port)
		c.storeNode(ctx, msg.Hostname, msg.Port, NodeHealthy, time.Now())
		c.startGossip(ctx)
		c.inCluster = true
	case *LeaveCluster:
		fmt.Printf("[DEBUG] ClusterActor %s Received LeaveCluster, for cluster %s:%s\n", c.id, msg.Hostname, msg.Port)
		c.inCluster = false
		c.disownSelf(ctx)
	case *Gossip:
		fmt.Printf("[DEBUG] ClusterActor %s Received Gossip\n", c.id)
		c.gossip(ctx, c.remote)
	case *GossipBatch:
		fmt.Printf("[DEBUG] ClusterActor %s Received GossipBatch\n", c.id)
		c.gossipBatch(ctx, c.remote)
	case *SwimPing:
		fmt.Printf("[DEBUG] ClusterActor %s Received SwimPing\n", c.id)
		if c.inCluster {
			c.handleGossipPing(ctx)
		}
	case *SwimAck:
		fmt.Printf("[DEBUG] ClusterActor %s Received SwimAck\n", c.id)
		if c.inCluster {
			c.handleGossipAck(ctx)
		}
	case *SwimPingReq:
		fmt.Printf("[DEBUG] ClusterActor %s Received SwimPingReq\n", c.id)
		if c.inCluster {
			c.handleGossipPingReq(ctx)
		}
	case *SwimAckReq:
		fmt.Printf("[DEBUG] ClusterActor %s Received SwimAckReq\n", c.id)
		if c.inCluster {
			c.handleGossipAckReq(ctx)
		}
	case *SuspectCheck:
		fmt.Printf("[DEBUG] ClusterActor %s Received SuspectCheck\n", c.id)
		if c.inCluster {
			c.suspectCheck(ctx, c.remote)
		}
	default:
		fmt.Printf("[DEBUG] ClusterActor %s Received unknown message type: %T\n", c.id, msg)
	}
}

func (c *ClusterActor) handleProtoAny(ctx actor.ActorContext, msg *anypb.Any) error {
	fmt.Printf("[DEBUG] Received proto.Any with TypeUrl: %s\n", msg.TypeUrl) // Log TypeUrl to help debug

	switch msg.TypeUrl {
	// Match the fully-qualified TypeUrl for each message
	case "type.googleapis.com/cluster.SwimPing":
		swimPing := &SwimPing{}
		if err := msg.UnmarshalTo(swimPing); err != nil {
			return fmt.Errorf("error unmarshalling SwimPing: %w", err)
		}
		fmt.Printf("[DEBUG] PbAny ClusterActor %s Received SwimPing from %s\n", c.id, swimPing.Sender)
		ctx.Send(swimPing, *ctx.Self())

	case "type.googleapis.com/cluster.SwimAck":
		swimAck := &SwimAck{}
		if err := msg.UnmarshalTo(swimAck); err != nil {
			return fmt.Errorf("error unmarshalling SwimAck: %w", err)
		}
		fmt.Printf("[DEBUG] PbAny ClusterActor %s Received SwimAck from %s\n", c.id, swimAck.Sender)
		ctx.Send(swimAck, *ctx.Self())

	case "type.googleapis.com/cluster.SwimPingReq":
		swimPingReq := &SwimPingReq{}
		if err := msg.UnmarshalTo(swimPingReq); err != nil {
			return fmt.Errorf("error unmarshalling SwimPingReq: %w", err)
		}
		fmt.Printf("[DEBUG] PbAny ClusterActor %s Received SwimPingReq from %s\n", c.id, swimPingReq.Sender)
		ctx.Send(swimPingReq, *ctx.Self())

	case "type.googleapis.com/cluster.SwimAckReq":
		swimAckReq := &SwimAckReq{}
		if err := msg.UnmarshalTo(swimAckReq); err != nil {
			return fmt.Errorf("error unmarshalling SwimAckReq: %w", err)
		}
		fmt.Printf("[DEBUG] PbAny ClusterActor %s Received SwimAckReq from %s\n", c.id, swimAckReq.Sender)
		ctx.Send(swimAckReq, *ctx.Self())
	default:
		return fmt.Errorf("unknown proto.Any typeUrl: %s", msg.TypeUrl)
	}
	return nil
}

func (c *ClusterActor) storeNode(ctx actor.ActorContext, address, port string, state KademliaNodeState, lastPing time.Time) {
	fmt.Printf("[DEBUG] ClusterActor %s Storing node %s:%s in DHT\n", c.id, address, port)
	c.dht.Store(address, port, state, lastPing)
}

func (c *ClusterActor) startGossip(ctx actor.ActorContext) {
	if c.ticker != nil {
		fmt.Println("Gossip already started")
		return
	}

	log.Println("[DEBUG] ClusterActor %s STARTING GOSSIP", c.id)
	c.ticker = time.NewTicker(10 * time.Millisecond)
	go func() {
		for {
			select {
			case <-c.ticker.C:
				ctx.Send(&GossipBatch{}, *ctx.Self())
				time.Sleep(10 * time.Millisecond)
				ctx.Send(&SuspectCheck{}, *ctx.Self())
			case <-c.stopGossip:
				c.ticker.Stop()
				return
			}
		}
	}()
}

func (c *ClusterActor) gossip(ctx actor.ActorContext, remote *remote.Remote) {
	// node := c.dht.RandomNode()
	closestNodesStringified := c.dht.FindStringified(c.dht.nodeId)
	node := c.dht.RendezvousNode(closestNodesStringified)
	if node.hostname == "" && node.port == "" && node.nodeId == "" {
		fmt.Println("[DEBUG] No nodes in DHT to gossip to")
		return
	}

	fmt.Printf("[DEBUG] Random node selected: %s:%s\n", node.hostname, node.port)

	msg := ctx.Message().(*Gossip)

	swimPing := &SwimPing{
		Sender: HostnamePortToAddress(c.hostname, c.port),
		Target: HostnamePortToAddress(node.hostname, node.port),
	}
	if msg.Disowned != nil {
		swimPing.Disowned = &DisownedEntry{
			Target:    msg.Disowned.Target,
			SeenCount: msg.Disowned.SeenCount + 1,
		}
	}

	fmt.Printf("[DEBUG] ClusterActor %s Gossiping to %s:%s\n", c.id, node.hostname, node.port)

	if c.dht.port == "8100" || c.dht.port == "8010" {
		fmt.Println("[STARTER NODE] Sending gossip message to node:", node.hostname, node.port)
	}

	err := c.sendMessageToRemoteActor(ctx,
		HostnamePortToAddress(node.hostname, node.port),
		"cluster-actor-"+HostnamePortToAddress(node.hostname, node.port),
		swimPing,
	)
	if err != nil {
		fmt.Println("Error sending message to remote actor:", err)
	}
}

func (c *ClusterActor) gossipBatch(ctx actor.ActorContext, remote *remote.Remote) {
	// nodes := c.dht.RandomNodesBatch()
	closestNodesStringified := c.dht.FindStringified(c.dht.nodeId)
	nodes := c.dht.RendezvousNodeBatch(closestNodesStringified)
	if len(nodes) == 0 {
		fmt.Println("[DEBUG] No nodes in DHT to gossip to")
		return
	}

	msg := ctx.Message().(*GossipBatch)
	for _, node := range nodes {
		swimPing := &SwimPing{
			Sender:   HostnamePortToAddress(c.hostname, c.port),
			Target:   HostnamePortToAddress(node.hostname, node.port),
			Disowned: msg.Disowned,
		}
		if msg.Disowned != nil {
			swimPing.Disowned = &DisownedEntry{
				Target:    msg.Disowned.Target,
				SeenCount: msg.Disowned.SeenCount + 1,
			}
		}
		err := c.sendMessageToRemoteActor(ctx,
			HostnamePortToAddress(node.hostname, node.port),
			"cluster-actor-"+HostnamePortToAddress(node.hostname, node.port),
			swimPing,
		)
		if err != nil {
			fmt.Println("Error sending message to remote actor:", err)
		}
	}
}

func (c *ClusterActor) disownSelf(ctx actor.ActorContext) {
	c.gossipDisownSelf(ctx)
	c.stopGossip <- struct{}{}
}

func (c *ClusterActor) gossipDisownSelf(ctx actor.ActorContext) {

	randomNodes := c.dht.RendezvousNodeBatch(c.dht.nodeId)
	if len(randomNodes) == 0 {
		fmt.Println("[DEBUG] No nodes in DHT to gossip to")
		return
	}
	for _, node := range randomNodes {
		err := c.sendMessageToRemoteActor(
			ctx,
			HostnamePortToAddress(node.hostname, node.port),
			"cluster-actor-"+HostnamePortToAddress(node.hostname, node.port),
			&SwimPing{
				Sender:   HostnamePortToAddress(c.hostname, c.port),
				Target:   HostnamePortToAddress(node.hostname, node.port),
				Disowned: &DisownedEntry{Target: HostnamePortToAddress(c.hostname, c.port), SeenCount: 0},
			},
		)
		if err != nil {
			fmt.Println("Error sending message to remote actor:", err)
		}
	}
	fmt.Printf("[DEBUG] ClusterActor %s Disowning self\n", c.id)
}

func (c *ClusterActor) suspectCheck(ctx actor.ActorContext, remote *remote.Remote) {
	// node := c.dht.RandomNode()
	closestNodesStringified := c.dht.FindStringified(c.dht.nodeId)
	node := c.dht.RendezvousNode(closestNodesStringified)
	if node.hostname == "" && node.port == "" && node.nodeId == "" {
		fmt.Println("[DEBUG] No nodes in DHT to gossip to")
		return
	}

	fmt.Printf("[DEBUG] ClusterActor %s Checking suspect node %s:%s\n", c.id, node.hostname, node.port)
	if node.state == NodeHealthy && node.lastPing.Add(5*time.Second).Before(time.Now()) {
		c.dht.Store(node.hostname, node.port, NodeSuspect, node.lastPing)

		err := c.sendMessageToRemoteActor(ctx, HostnamePortToAddress(node.hostname, node.port),
			"cluster-actor-"+HostnamePortToAddress(node.hostname, node.port),
			&SwimPing{
				Sender: HostnamePortToAddress(c.hostname, c.port),
				Target: HostnamePortToAddress(node.hostname, node.port),
			})
		if err != nil {
			fmt.Println("Error sending message to remote actor:", err)
		}
		fmt.Printf("[DEBUG] ClusterActor %s Health check successful for %s:%s\n", c.id, node.hostname, node.port)

		nodeAddress := HostnamePortToAddress(node.hostname, node.port)
		closestNodes := c.dht.Find(node.nodeId)

		for _, closeNode := range closestNodes {

			err := c.sendMessageToRemoteActor(ctx, HostnamePortToAddress(closeNode.hostname, closeNode.port),
				"cluster-actor-"+HostnamePortToAddress(closeNode.hostname, closeNode.port),
				&SwimPingReq{
					Sender: HostnamePortToAddress(c.hostname, c.port),
					Target: nodeAddress,
				})
			if err != nil {
				fmt.Println("Error sending message to remote actor:", err)
			}
		}
		fmt.Printf("[DEBUG] ClusterActor %s Suspect check failed for %s:%s, setting suspect\n", c.id, node.hostname, node.port)

		return
	} else if node.state == NodeSuspect && node.lastPing.Add(10*time.Second).After(time.Now()) {
		nodeAddress := HostnamePortToAddress(node.hostname, node.port)
		closestNodes := c.dht.Find(node.nodeId)
		for _, closeNode := range closestNodes {

			err := c.sendMessageToRemoteActor(ctx, HostnamePortToAddress(closeNode.hostname, closeNode.port),
				"cluster-actor-"+HostnamePortToAddress(closeNode.hostname, closeNode.port),
				&SwimPingReq{
					Sender: HostnamePortToAddress(c.hostname, c.port),
					Target: nodeAddress,
				})
			if err != nil {
				fmt.Println("Error sending message to remote actor:", err)
			}
		}
		fmt.Printf("[DEBUG] ClusterActor %s Suspect check failed for %s:%s, setting suspect\n", c.id, node.hostname, node.port)
		return
	} else if node.state == NodeSuspect && node.lastPing.Add(10*time.Second).Before(time.Now()) {

		c.dht.Remove(node.hostname, node.port)
		// randomNodes := c.dht.RandomNodesBatch()
		randomNodes := c.dht.RendezvousNodeBatch(node.nodeId)
		if len(randomNodes) == 0 {
			fmt.Println("[DEBUG] No nodes in DHT to gossip to")
			return
		}
		// disseminate disown message to random nodes
		for _, randomNode := range randomNodes {
			err := c.sendMessageToRemoteActor(ctx, HostnamePortToAddress(randomNode.hostname, randomNode.port),
				"cluster-actor-"+HostnamePortToAddress(randomNode.hostname, randomNode.port),
				&SwimPing{
					Sender:   HostnamePortToAddress(c.hostname, c.port),
					Target:   HostnamePortToAddress(randomNode.hostname, randomNode.port),
					Disowned: &DisownedEntry{Target: HostnamePortToAddress(node.hostname, node.port), SeenCount: 0},
				})
			if err != nil {
				fmt.Println("Error sending message to remote actor:", err)
			}
		}
		fmt.Printf("[DEBUG] ClusterActor %s Suspect check failed for %s:%s, disowning\n", c.id, node.hostname, node.port)
	}
}

func (c *ClusterActor) handleGossipPing(ctx actor.ActorContext) {
	msg := ctx.Message().(*SwimPing)
	senderAddress := msg.Sender
	hostAddress := HostnamePortToAddress(c.hostname, c.port)

	hostname, port := AddressToHostnamePort(senderAddress)
	senderNodeId := Hash(hostname + port)
	c.dht.Store(hostname, port, NodeHealthy, time.Now())

	if disownedEntry := msg.Disowned; disownedEntry != nil {
		disownedHostname, disownedPort := AddressToHostnamePort(disownedEntry.Target)
		c.dht.Remove(disownedHostname, disownedPort)
		if msg.Disowned.SeenCount < gossipThreshold {
			newDisownedEntry := &DisownedEntry{
				Target:    disownedEntry.Target,
				SeenCount: disownedEntry.SeenCount + 1,
			}
			gossipMsg := &Gossip{Disowned: newDisownedEntry}
			// send to itself message to gossip disowned entry
			ctx.Send(gossipMsg, *ctx.Self())
		}
	}

	ackMsg := &SwimAck{Sender: senderAddress}
	remoteActorPID, err := c.remote.SpawnRemoteClusterActor(senderAddress, "cluster-actor-"+senderAddress)
	if err != nil {
		fmt.Println("Error spawning remote sender actor:", err)
		return
	}
	ctx.Send(ackMsg, remoteActorPID)

	nodes := c.dht.Find(senderNodeId)

	for _, node := range nodes {
		if distance := getXORDistance(senderNodeId, node.nodeId); distance == big.NewInt(0) {
			//if in nodes in dht, continue since we already sent ack
			continue
		} else {
			// if not in nodes in dht, send ack req to closer nodes, still is idempotent
			err := c.sendMessageToRemoteActor(ctx, HostnamePortToAddress(node.hostname, node.port),
				"cluster-actor-"+HostnamePortToAddress(node.hostname, node.port),
				&SwimAckReq{
					Sender: hostAddress,
					Target: senderAddress,
				})
			if err != nil {
				fmt.Printf("[DEBUG] Error sending message from cluster-actor %s to remote actor: %v\n", hostAddress, err)
				fmt.Println("Error sending message to remote actor:", err)
			}
		}
	}
}

func (c *ClusterActor) handleGossipPingReq(ctx actor.ActorContext) {
	msg := ctx.Message().(*SwimPingReq)
	targetAddress := msg.Target
	senderAddress := msg.Sender
	hostname, port := AddressToHostnamePort(targetAddress)
	targetNodeId := Hash(hostname + port)

	// Handle DisownedEntry if it exists
	var newDisownedEntry *DisownedEntry
	if msg.Disowned != nil {
		disownedHostname, disownedPort := AddressToHostnamePort(msg.Disowned.Target)
		c.dht.Remove(disownedHostname, disownedPort)
		if msg.Disowned.SeenCount < gossipThreshold {
			newDisownedEntry = &DisownedEntry{
				Target:    msg.Disowned.Target,
				SeenCount: msg.Disowned.SeenCount + 1,
			}
		}
	}

	nodes := c.dht.Find(targetNodeId)
	for _, node := range nodes {
		nodeAddress := HostnamePortToAddress(node.hostname, node.port)

		var msgToSend interface{}
		if getXORDistance(targetNodeId, node.nodeId).Cmp(big.NewInt(0)) == 0 {
			msgToSend = &SwimPing{
				Sender:   senderAddress,
				Target:   targetAddress,
				Disowned: newDisownedEntry,
			}
		} else {
			msgToSend = &SwimPingReq{
				Sender:   senderAddress,
				Target:   targetAddress,
				Disowned: newDisownedEntry,
			}
		}

		remoteActorPID, err := c.remote.SpawnRemoteClusterActor(nodeAddress, "cluster-actor-"+nodeAddress)
		if err != nil {
			fmt.Println("Error spawning remote actor:", err)
			continue
		}

		ctx.Send(msgToSend, remoteActorPID)
	}
}

func (c *ClusterActor) handleGossipAck(ctx actor.ActorContext) {
	msg := ctx.Message().(*SwimAck)
	targetAddress := msg.Sender
	hostname, port := AddressToHostnamePort(targetAddress)
	c.dht.Store(hostname, port, NodeHealthy, time.Now())
}

func (c *ClusterActor) handleGossipAckReq(ctx actor.ActorContext) {
	msg := ctx.Message().(*SwimAckReq)
	targetAddress := msg.Target
	senderAddress := msg.Sender

	hostname, port := AddressToHostnamePort(targetAddress)
	targetNodeId := Hash(hostname + port)

	c.dht.Store(hostname, port, NodeHealthy, time.Now())
	nodes := c.dht.Find(targetNodeId)

	for _, node := range nodes {
		nodeAddress := HostnamePortToAddress(node.hostname, node.port)

		var msgToSend interface{}
		if getXORDistance(targetNodeId, node.nodeId).Cmp(big.NewInt(0)) == 0 {
			msgToSend = &SwimAck{
				Sender: senderAddress,
				Target: targetAddress,
			}
		} else {
			msgToSend = &SwimAckReq{
				Sender: senderAddress,
				Target: targetAddress,
			}
		}

		remoteActorPID, err := c.remote.SpawnRemoteClusterActor(nodeAddress, "cluster-actor-"+nodeAddress)
		if err != nil {
			fmt.Println("Error spawning remote actor:", err)
			continue
		}

		ctx.Send(msgToSend, remoteActorPID)
	}
}

func (c *ClusterActor) sendMessageToRemoteActor(ctx actor.ActorContext, remoteAddress, actorName string, message interface{}) error {
	log.Printf("[DEBUG] ClusterActor %s Sending message to remote actor %s, actor name: %s\n", c.id, remoteAddress, actorName)
	remoteActorPID, err := c.remote.SpawnRemoteClusterActor(remoteAddress, actorName)
	if err != nil {
		return fmt.Errorf("error spawning remote actor: %w", err)
	}

	log.Printf("[DEBUG] Remote actor spawned with PID: %s\n", remoteActorPID)

	hostNodeAddress := fmt.Sprintf("%s:%s", c.hostname, c.port)
	err = c.remote.MakeActorDiscoverable(*ctx.Self(), "cluster-actor-"+hostNodeAddress)
	if err != nil {
		return fmt.Errorf("error making actor discoverable: %w", err)
	}

	log.Printf("[DEBUG] Making actor discoverable with PID: %s\n", *ctx.Self())

	ctx.Send(message, remoteActorPID)
	time.Sleep(5 * time.Millisecond)
	ctx.Send(actor.SystemMessage{Type: actor.SystemMessageGracefulStop}, remoteActorPID)
	return nil
}
