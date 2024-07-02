package main

import (
	"fmt"
	"light-actor-go/actor"
	"time"
)

// ParentActor is a top-level actor that spawns a child actor
type ParentActor struct{}

func (a *ParentActor) Receive(ctx actor.ActorContext) {
	// fmt.Println("Parent actor received message:", ctx.Message())

	switch msg := ctx.Message().(type) {
	case string:
		if msg == "SpawnChild" {
			self := ctx.Self()
			childProps := actor.NewActorProps(self) // Child actor will not have a parent

			actorSystem := ctx.ActorSystem()

			childPID, _ := ctx.SpawnActor(&ChildActor{}, *childProps)

			time.Sleep(1 * time.Second)

			actorSystem.Send(actor.NewEnvelope("SpawnGrandchild", childPID))
		}
	case actor.SystemMessage:
		switch msg.Type {
		case actor.SystemMessageStop:
			fmt.Println("Parent actor received stop message")
		case actor.SystemMessageGracefulStop:
			fmt.Println("Parent actor received graceful stop message")
		case actor.SystemMessageChildTerminated:
			fmt.Println("Parent actor received child terminated message")
		case actor.SystemMessageStart:
			fmt.Println("Parent actor received start message")
		}
	}

}

// ChildActor is spawned by ParentActor and spawns another actor
type ChildActor struct{}

func (a *ChildActor) Receive(ctx actor.ActorContext) {
	// fmt.Println("Child actor received message:", ctx.Message())

	switch msg := ctx.Message().(type) {
	case string:
		fmt.Println("Child actor received:", msg)
		if msg == "SpawnGrandchild" {
			self := ctx.Self()
			grandChildProps := actor.NewActorProps(self)
			ctx.SpawnActor(&GrandChildActor{}, *grandChildProps)

		}
	case actor.SystemMessage:
		switch msg.Type {
		case actor.SystemMessageStop:
			fmt.Println("Child actor received stop message")
		case actor.SystemMessageGracefulStop:
			fmt.Println("Child actor received graceful stop message")
		case actor.SystemMessageChildTerminated:
			fmt.Println("Child actor received child terminated message")
		case actor.SystemMessageStart:
			fmt.Println("Child actor received start message")
		}
	}
}

// GrandChildActor is spawned by ChildActor
type GrandChildActor struct{}

func (a *GrandChildActor) Receive(ctx actor.ActorContext) {
	// fmt.Println("Grandchild actor received message:", ctx.Message())

	switch msg := ctx.Message().(type) {
	case string:
		fmt.Println("Grandchild actor received:", msg)
	case actor.SystemMessage:
		switch msg.Type {
		case actor.SystemMessageStop:
			fmt.Println("Grandchild actor received stop message")
		case actor.SystemMessageGracefulStop:
			fmt.Println("Grandchild actor received graceful stop message")
		case actor.SystemMessageChildTerminated:
			fmt.Println("Grandchild actor received child terminated message")
		case actor.SystemMessageStart:
			fmt.Println("Grandchild actor received start message")
		}
	}
}

func main() {
	actorSystem := actor.NewActorSystem()

	// Spawn the top-level parent actor
	parentPID, err := actorSystem.SpawnActor(&ParentActor{})
	if err != nil {
		fmt.Println("Error spawning parent actor:", err)
		return
	}

	time.Sleep(1 * time.Second)

	actorSystem.Send(actor.NewEnvelope("SpawnChild", parentPID))

	time.Sleep(3 * time.Second)

	// Trigger graceful stop of parent actor
	actorSystem.GracefulStop(parentPID)

	time.Sleep(2 * time.Second)

	// Hello is to be ignored as parent actor is in stopping state
	actorSystem.Send(actor.NewEnvelope("Hellooooo", parentPID))

	time.Sleep(10 * time.Second)
}
