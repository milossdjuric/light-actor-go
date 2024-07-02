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

			childProps := actor.NewActorPropsWithStrategies(self, actor.NewResumeOneStrategy(), actor.NewResumeOneStrategy()) // Child actor will not have a parent

			actorSystem := ctx.ActorSystem()

			childPID, _ := ctx.SpawnActor(&ChildActor{}, *childProps)
			childPID2, _ := ctx.SpawnActor(&ChildActor{}, *childProps)
			childPID3, _ := ctx.SpawnActor(&ChildActor{}, *childProps)

			fmt.Println("Spawned child actors:", childPID, childPID2, childPID3)

			time.Sleep(1 * time.Second)

			actorSystem.Send(actor.NewEnvelope("SpawnGrandchild", childPID))
			actorSystem.Send(actor.NewEnvelope("SpawnGrandchild", childPID2))
			actorSystem.Send(actor.NewEnvelope("SpawnGrandchild", childPID3))

			time.Sleep(1 * time.Second)

		} else if msg == "Panic" {
			panic("ParentPanic")
		} else if msg == "Hellooo" {
			actorSystem := ctx.ActorSystem()
			fmt.Println("Len of children:", len(ctx.Children()))
			for _, child := range ctx.Children() {
				actorSystem.Send(actor.NewEnvelope("Hellooo", *child))
			}
		}
	case actor.SystemMessage:
		switch msg.Type {
		case actor.SystemMessageFailure:

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
			grandChildPID, _ := ctx.SpawnActor(&GrandChildActor{}, *grandChildProps)

			fmt.Println("Spawned grandchild actor:", grandChildPID)

			time.Sleep(1 * time.Second)

		} else if msg == "Panic" {
			panic("Panic")
		} else if msg == "Hellooo" {
			actorSystem := ctx.ActorSystem()
			for _, child := range ctx.Children() {
				fmt.Println("Sending hello from child to grandchild:", child)
				actorSystem.Send(actor.NewEnvelope("Hello from child", *child))
			}
		}

	case actor.SystemMessage:
		switch msg.Type {
		case actor.SystemMessageFailure:
			failure := msg.Extras.(actor.Failure)
			fmt.Printf("Child actor received failure: %v from %v\n", failure.Reason, failure.Who)
		case actor.SystemMessageStop:
			fmt.Println("Child actor received stop message")
		case actor.SystemMessageGracefulStop:
			fmt.Println("Child actor received graceful stop message")
		case actor.SystemMessageChildTerminated:
			fmt.Println("Child actor received child terminated message")
		case actor.SystemMessageStart:
			fmt.Println("Child actor received start message")
		case actor.SuspendMailbox:
			fmt.Println("Child actor received suspend mailbox message")
		case actor.ResumeMailbox:
			fmt.Println("Child actor received resume mailbox message")
		case actor.SuspendMailboxAll:
			fmt.Println("Child actor received suspend mailbox all message")
		case actor.ResumeMailboxAll:
			fmt.Println("Child actor received resume mailbox all message")
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
		case actor.SystemMessageFailure:
			failure := msg.Extras.(actor.Failure)
			fmt.Printf("Grandchild actor received failure: %v from %v\n", failure.Reason, failure.Who)
		case actor.SystemMessageStop:
			fmt.Println("Grandchild actor received stop message")
		case actor.SystemMessageGracefulStop:
			fmt.Println("Grandchild actor received graceful stop message")
		case actor.SystemMessageChildTerminated:
			fmt.Println("Grandchild actor received child terminated message")
		case actor.SystemMessageStart:
			fmt.Println("Grandchild actor received start message")
		case actor.SuspendMailbox:
			fmt.Println("Grandchild actor received suspend mailbox message")
		case actor.ResumeMailbox:
			fmt.Println("Grandchild actor received resume mailbox message")
		case actor.SuspendMailboxAll:
			fmt.Println("Grandchild actor received suspend mailbox all message")
		case actor.ResumeMailboxAll:
			fmt.Println("Grandchild actor received resume mailbox all message")
		}
	}
}

func main() {
	actorSystem := actor.NewActorSystem()

	//We use Resume strategy for the parent actor, parent will continue to work, as well as its children
	props := actor.NewActorPropsWithStrategies(nil, actor.NewResumeOneStrategy(), actor.NewResumeOneStrategy())

	// Spawn the top-level parent actor
	parentPID, err := actorSystem.SpawnActor(&ParentActor{}, *props)
	if err != nil {
		fmt.Println("Error spawning parent actor:", err)
		return
	}

	time.Sleep(3 * time.Second)

	actorSystem.Send(actor.NewEnvelope("SpawnChild", parentPID))

	time.Sleep(3 * time.Second)

	actorSystem.EscalateFailureNotPanic("Not Panic", parentPID)

	time.Sleep(5 * time.Second)

	actorSystem.Send(actor.NewEnvelope("Hellooo", parentPID))

	time.Sleep(5 * time.Second)
}
