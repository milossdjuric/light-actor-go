package remote

import (
	"fmt"
	"light-actor-go/actor"
	"log"
)

type Remote struct {
	remoteReciever RemoteReceiver
	actorSystem    *actor.ActorSystem
}

func NewRemote(remoteConfing RemoteConfig, actorSystem *actor.ActorSystem) *Remote {
	defer log.Println("[DEBUG REMOTE] Actor system: ", actorSystem)
	return &Remote{
		remoteReciever: *NewRemoteReceiver(&remoteConfing, actorSystem),
		actorSystem:    actorSystem,
	}
}

func (r *Remote) Listen() {
	go r.remoteReciever.startServer()
}

func (r *Remote) SpawnRemoteActor(address string, name string) (actor.PID, error) {
	newPID, err := actor.NewPID()
	if err != nil {
		return newPID, nil
	}

	remoteSender := NewRemoteSender(address)
	envelopeChan := make(chan actor.Envelope, 10)

	go func() {
		for {
			envelope := <-envelopeChan
			err := remoteSender.SendMessage(envelope.Message, name)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	r.actorSystem.AddRemoteActor(newPID, envelopeChan)
	return newPID, nil
}

func (r *Remote) SpawnRemoteClusterActor(address string, name string) (actor.PID, error) {
	newPID, err := actor.NewPID()
	if err != nil {
		return newPID, nil
	}

	remoteSender := NewRemoteSender(address)
	envelopeChan := make(chan actor.Envelope, 10)

	go func() {
		defer func() {
			r.actorSystem.RemoveRemoteActor(newPID)
			close(envelopeChan)
		}()

		for envelope := range envelopeChan {
			switch msg := envelope.Message.(type) {
			case actor.SystemMessage:
				if msg.Type == actor.SystemMessageStop || msg.Type == actor.SystemMessageGracefulStop {
					r.actorSystem.RemoveRemoteActor(newPID)
					return
				}
			}

			err := remoteSender.SendMessage(envelope.Message, name)
			if err != nil {
				fmt.Println("SpawnRemoteClusterActor Error: ", err)
			}
		}
	}()
	if r.ActorSystem() == nil {
		fmt.Println("SpawnRemoteClusterActor Error: Actor system is nil")
	}
	r.ActorSystem().AddRemoteActor(newPID, envelopeChan)

	return newPID, nil
}

func (r *Remote) MakeActorDiscoverable(actorPID actor.PID, name string) error {
	return r.remoteReciever.AddRemoteActor(name, actorPID)
}

func (r *Remote) ActorSystem() *actor.ActorSystem {
	return r.actorSystem
}

func (r *Remote) Address() string {
	return r.remoteReciever.config.Addr
}

// func (r *Remote) findActorName(actorPID actor.PID) string {
// 	return r.remoteActorRegistry.Find(actorPID)
// }
