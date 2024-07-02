package main

import (
	"fmt"
	"light-actor-go/actor"
	"light-actor-go/examples/remote/messages"
	"light-actor-go/remote"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"
)

type StringMessage struct {
	Value string
}

type PingActor struct{}

func (p *PingActor) Receive(context actor.ActorContext) {
	switch msg := context.Message().(type) {
	case *anypb.Any:
		stringMsg := &messages.StringMessage{}
		if err := msg.UnmarshalTo(stringMsg); err != nil {
			fmt.Printf("Error unmarshalling message: %v\n", err)
			return
		}
		fmt.Printf("PingActor received: %s\n", string(stringMsg.Value))
		id, err := uuid.Parse(string(stringMsg.Value))
		if err != nil {
			fmt.Printf("Error parsing UUID: %v\n", err)
			return
		}
		// Respond with a pong message
		context.Send(&messages.StringMessage{Value: "Ping"}, actor.PID{ID: id})
	case *messages.StringMessage:

	default:
		//fmt.Printf("PingActor received an unknown message type: %T\n", msg)
	}
}

type PongActor struct{}

func (p *PongActor) Receive(context actor.ActorContext) {
	switch context.Message().(type) {
	case *anypb.Any:
		fmt.Println("PongActor received ping")
		fmt.Println("Ping Pong interaction completed")
	default:
		//fmt.Println("Unknown message type in pong", msg)
	}
}

func main() {
	// Setup Actor Systems and Remote Communication
	pingSystem := actor.NewActorSystem()
	remote1 := remote.NewRemote(*remote.NewRemoteConfig("127.0.0.1:8091"), pingSystem)
	remote1.Listen()

	pingActor := &PingActor{}
	pingActorID, err := pingSystem.SpawnActor(pingActor)
	if err != nil {
		fmt.Println("Error spawning PingActor:", err)
		return
	}
	remote1.MakeActorDiscoverable(pingActorID, "PingActor")

	pongSystem := actor.NewActorSystem()
	remote2 := remote.NewRemote(*remote.NewRemoteConfig("127.0.0.1:8092"), pongSystem)
	remote2.Listen()

	pongActor := &PongActor{}
	pongActorID, err := pongSystem.SpawnActor(pongActor)
	if err != nil {
		fmt.Println("Error spawning PongActor:", err)
		return
	}
	remote2.MakeActorDiscoverable(pongActorID, "PongActor")

	// Wait for actors to be ready
	time.Sleep(time.Second * 1)

	// Initiate Ping Pong Interaction
	remoteID, _ := remote2.SpawnRemoteActor("127.0.0.1:8091", "PingActor")

	remoteID2, _ := remote1.SpawnRemoteActor("127.0.0.1:8092", "PongActor")
	pingMessage := &messages.StringMessage{Value: remoteID2.ID.String()}

	// Send initial ping from remote1 to remote2
	pongSystem.Send(actor.NewEnvelope(pingMessage, remoteID))

	// Wait for completion (optional)
	time.Sleep(time.Second * 3)

	// Shutdown actors and remote pingSystems (deferred)
	defer pingSystem.GracefulStop(pingActorID)
	defer pongSystem.GracefulStop(pongActorID)
}
