package cluster

import (
	"light-actor-go/actor"
	"light-actor-go/remote"
	"log"
)

type ClusterNode struct {
	remote    *remote.Remote
	nodeActor *actor.PID
}

func NewClusterNode(actorSystem *actor.ActorSystem, remoteConfig remote.RemoteConfig) *ClusterNode {

	clusterNode := &ClusterNode{
		remote: remote.NewRemote(remoteConfig, actorSystem),
	}

	log.Printf("[CLUSTER NODE] Custer node: %v\n", clusterNode)

	clusterProps := actor.NewActorPropsWithStrategies(nil, actor.NewRestartAllStrategy(), actor.NewRestartAllStrategy())
	clusterActorName := "cluster-actor-" + remoteConfig.Addr
	hostname, port := AddressToHostnamePort(remoteConfig.Addr)
	log.Println("[CLUSTER NODE] Actor System: ", clusterNode.remote.ActorSystem())
	nodeActorPID, err := clusterNode.remote.ActorSystem().SpawnActor(NewClusterActor(clusterActorName, hostname, port, clusterNode.remote), *clusterProps)
	if err != nil {
		panic(err)
	}

	clusterNode.remote.MakeActorDiscoverable(nodeActorPID, clusterActorName)
	clusterNode.nodeActor = &nodeActorPID

	return clusterNode
}

func (c *ClusterNode) Start() {
	c.remote.Listen()
}

func (c *ClusterNode) JoinCluster(hostname, port string) {
	c.remote.ActorSystem().Send(actor.NewEnvelope(&JoinCluster{Hostname: hostname, Port: port}, *c.nodeActor))
}

func (c *ClusterNode) LeaveCluster(hostname, port string) {
	c.remote.ActorSystem().Send(actor.NewEnvelope(&LeaveCluster{Hostname: hostname, Port: port}, *c.nodeActor))
}

func (c *ClusterNode) Address() string {
	return c.remote.Address()
}
