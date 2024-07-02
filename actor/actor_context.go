package actor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Define the actorState constants
type actorState int32

const (
	actorStart actorState = iota
	actorStop
	actorStopping
)

// ActorContext holds the state and context of an actor
type ActorContext struct {
	actor       Actor
	actorSystem *ActorSystem
	ctx         context.Context
	props       *ActorProps
	envelope    Envelope
	state       actorState
	children    map[PID]bool
	self        PID
	mu          sync.RWMutex
	actorChan   chan Envelope
}

// NewActorContext creates and initializes a new actorContext
func NewActorContext(actor Actor, ctx context.Context, actorSystem *ActorSystem, props *ActorProps, self PID, actorChan chan Envelope) *ActorContext {
	context := new(ActorContext)
	context.actor = actor
	context.ctx = ctx
	context.props = props
	context.state = actorStart
	context.actorSystem = actorSystem
	context.self = self
	context.children = make(map[PID]bool) // Initialize children as a map
	context.actorChan = actorChan
	return context
}

// Adds envelope to the current actor context
func (ctx *ActorContext) AddEnvelope(envelope Envelope) {
	ctx.envelope = envelope
}

// Spawns child actor
func (ctx *ActorContext) SpawnActor(actor Actor, Props ...ActorProps) (PID, error) {
	prop := ConfigureActorProps(Props...)
	prop.AddParent(&ctx.self)

	// pid, err := NewPID()
	// if err != nil {
	// 	return PID{}, err
	// }

	pid, err := ctx.actorSystem.SpawnActor(actor, *prop)
	if err != nil {
		return PID{}, err
	}

	ctx.mu.Lock()
	ctx.children[pid] = true
	ctx.mu.Unlock()

	return pid, nil
}

// Send message
func (ctx *ActorContext) Send(message interface{}, receiver PID) {
	sendEnvelope := NewEnvelope(message, receiver)
	ctx.actorSystem.Send(sendEnvelope)
}

func (ctx *ActorContext) Message() interface{} {
	return ctx.envelope.Message
}

// Context returns the context attached to the actorContext
func (ctx *ActorContext) Context() context.Context {
	return ctx.ctx
}

// Message returns the current message being processed
func (ctx *ActorContext) Envelope() Envelope {
	return ctx.envelope
}

// State returns the current state of the actor
func (ctx *ActorContext) State() actorState {
	return ctx.state
}

func (ctx *ActorContext) Self() *PID {
	return &ctx.self
}

func (ctx *ActorContext) ActorSystem() *ActorSystem {
	return ctx.actorSystem
}

func (ctx *ActorContext) Parent() *PID {
	return ctx.props.Parent
}

func (ctx *ActorContext) Children() []*PID {
	ctx.mu.RLock()

	children := make([]*PID, 0, len(ctx.children))
	for child := range ctx.children {
		children = append(children, &child)
	}
	ctx.mu.RUnlock()

	return children
}

func (ctx *ActorContext) RemoveChild(child PID) {
	ctx.mu.Lock()
	delete(ctx.children, child)
	ctx.mu.Unlock()
}

func (ctx *ActorContext) Props() *ActorProps {
	return ctx.props
}

func (ctx *ActorContext) HandleSystemMessage(msg SystemMessage) {
	switch msg.Type {
	case SystemMessageStart:
		ctx.Start()
	case SystemMessageStop:
		ctx.Stop()
	case SystemMessageGracefulStop:
		ctx.GracefulStop()
	case SystemMessageChildTerminated:
		if child, ok := msg.Extras.(PID); ok {
			ctx.ChildTerminated(child)
		} else {
			fmt.Println("System message extras not a PID")
		}
	case SystemMessageFailure:
		if failure, ok := msg.Extras.(Failure); ok {
			ctx.HandleFailure(failure)
		} else {
			fmt.Println("System message extras not a Failure")
		}
	case SystemMessageEscalateFailure:
		if failure, ok := msg.Extras.(Failure); ok {
			ctx.EscalateFailure(failure)
		} else {
			fmt.Println("System message extras not a Failure")
		}
	case SystemMessageRestart:
		ctx.Restart()
	case SuspendMailbox:
		//ignore
		// fmt.Println("Suspend mailbox: ", ctx.self)
	case ResumeMailbox:
		//ignore
		// fmt.Println("Resume mailbox: ", ctx.self)
	case SuspendMailboxAll:
		ctx.SuspendChildren()
		// fmt.Println("Suspend all mailboxes: ", ctx.self)
	case ResumeMailboxAll:
		ctx.ResumeChildren()
		// fmt.Println("Resume all mailboxes: ", ctx.self)
	default:
		// fmt.Println("System message unknown")
	}
}

func (ctx *ActorContext) Start() {
	ctx.state = actorStart
	// fmt.Println("System message start:", ctx.self)
}

func (ctx *ActorContext) Restart() {

	ctx.mu.RLock()
	if len(ctx.children) > 0 {
		for child, _ := range ctx.children {
			ctx.actorSystem.SendSystemMessage(child, SystemMessage{Type: SystemMessageStop})
		}
		ctx.mu.RUnlock()
	} else {
		ctx.mu.RUnlock()
	}

	ctx.state = actorStop
	ctx.actorSystem.RespawnActor(ctx.actor, ctx.self, ctx.actorChan, *ctx.props)
}

func (ctx *ActorContext) Stop() {

	ctx.mu.RLock()
	if len(ctx.children) > 0 {
		for child, _ := range ctx.children {
			ctx.actorSystem.SendSystemMessage(child, SystemMessage{Type: SystemMessageStop})
		}
		ctx.mu.RUnlock()
	} else {
		ctx.mu.RUnlock()
	}

	ctx.actorSystem.RemoveActor(ctx.self, SystemMessage{Type: DeleteMailbox})
	ctx.state = actorStop
	// fmt.Println("System message stop", ctx.self)
}

func (ctx *ActorContext) GracefulStop() {
	ctx.state = actorStopping
	// fmt.Println("System message stopping", ctx.self)

	time.Sleep(1 * time.Second)

	ctx.mu.RLock()

	if len(ctx.children) > 0 {
		for child, _ := range ctx.children {
			ctx.actorSystem.SendSystemMessage(child, SystemMessage{Type: SystemMessageGracefulStop})
			// fmt.Println("Sending graceful stop to child:", child)
		}
		ctx.mu.RUnlock()
	} else {
		ctx.mu.RUnlock()
		if ctx.props.Parent != nil {
			ctx.actorSystem.SendSystemMessage(*ctx.props.Parent, SystemMessage{Type: SystemMessageChildTerminated, Extras: ctx.self})
		}
		// fmt.Println("No children, actor stopped", ctx.self)
		ctx.actorSystem.RemoveActor(ctx.self, SystemMessage{Type: DeleteMailbox})
		ctx.state = actorStop
	}
}

func (ctx *ActorContext) ChildTerminated(child PID) {
	// fmt.Println("System message child terminated:", child)

	ctx.mu.Lock()
	delete(ctx.children, child)
	ctx.mu.Unlock()

	ctx.mu.RLock()
	if len(ctx.children) == 0 && ctx.state == actorStopping {
		ctx.mu.RUnlock()
		if ctx.props.Parent != nil {
			ctx.actorSystem.SendSystemMessage(*ctx.props.Parent, SystemMessage{Type: SystemMessageChildTerminated, Extras: ctx.self})
		}
		ctx.actorSystem.RemoveActor(ctx.self, SystemMessage{Type: DeleteMailbox})
		ctx.state = actorStop
		// fmt.Println("No more children left, actor stopping", ctx.self)
	} else {
		ctx.mu.RUnlock()
	}
}

func (ctx *ActorContext) SuspendChildren() {
	ctx.mu.RLock()
	if len(ctx.children) > 0 {
		for child, _ := range ctx.children {
			ctx.actorSystem.SendSystemMessage(child, SystemMessage{Type: SuspendMailboxAll})
		}
		ctx.mu.RUnlock()
	} else {
		ctx.mu.RUnlock()
	}
}

func (ctx *ActorContext) ResumeChildren() {
	ctx.mu.RLock()
	if len(ctx.children) > 0 {
		for child, _ := range ctx.children {
			ctx.actorSystem.SendSystemMessage(child, SystemMessage{Type: ResumeMailboxAll})
		}
		ctx.mu.RUnlock()
	} else {
		ctx.mu.RUnlock()
	}
}

func (ctx *ActorContext) EscalateFailure(failure Failure) {
	supervisorFailure := Failure{Reason: failure.Reason, Who: ctx.self, Actor: ctx.actor, ActorContext: ctx, ActorChan: ctx.actorChan}

	switch failure.Reason.(type) {
	case NotPanic:
		ctx.actorSystem.SendSystemMessage(ctx.self, SystemMessage{Type: SuspendMailbox})
		ctx.SuspendChildren()

		if ctx.props.Parent != nil {
			ctx.actorSystem.SendSystemMessage(*ctx.props.Parent, SystemMessage{Type: SystemMessageFailure, Extras: supervisorFailure})
		} else {
			ctx.props.RootStrategy().HandleFailure(ctx.actorSystem, ctx, supervisorFailure)
		}
	default:
		//default for failure escalation is panic
		panic("Panic")
	}
}

func (ctx *ActorContext) HandleFailure(failure Failure) {
	ctx.props.SupervisionStrategy().HandleFailure(ctx.ActorSystem(), ctx, failure)
}

func (ctx *ActorContext) HandleRootFailure(failure Failure) {
	ctx.props.RootStrategy().HandleFailure(ctx.ActorSystem(), ctx, failure)
}
