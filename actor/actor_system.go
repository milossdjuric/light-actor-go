package actor

import (
	"context"
	"fmt"
)

type ActorSystem struct {
	registry *Registry
}

// Creates new actor system that can only be used localy
func NewActorSystem() *ActorSystem {
	return &ActorSystem{registry: NewRegistry()}
}

func (system *ActorSystem) SpawnActor(a Actor, props ...ActorProps) (PID, error) {
	prop := ConfigureActorProps(props...)

	actorChan := make(chan Envelope)
	mailbox := NewMailbox(actorChan)

	mailboxChan := mailbox.GetChan()
	mailboxPID, err := NewPID()
	if err != nil {
		return mailboxPID, err
	}

	startMailbox(mailbox)

	startActor(a, system, prop, mailboxPID, actorChan)

	//Put mailbox chanel in registry
	err = system.registry.Add(mailboxPID, mailboxChan)
	if err != nil {
		return mailboxPID, err
	}

	return mailboxPID, nil
}

func (system *ActorSystem) RespawnActor(a Actor, mailboxPID PID, actorChan chan Envelope, props ...ActorProps) (PID, error) {
	prop := ConfigureActorProps(props...)

	if actorChan == nil {
		fmt.Println("ActorChan not found")
	}

	actorContext := NewActorContext(a, context.Background(), system, prop, mailboxPID, actorChan)

	//Start actor in separate gorutine
	startActorWithContext(a, system, actorContext, actorChan)

	system.SendSystemMessage(actorContext.self, SystemMessage{Type: ResumeMailboxAll})
	system.SendSystemMessage(actorContext.self, SystemMessage{Type: SystemMessageStart})

	return mailboxPID, nil
}

func startActor(a Actor, system *ActorSystem, prop *ActorProps, mailboxPID PID, actorChan chan Envelope) {
	go func() {

		//Setup basic actor context
		actorContext := NewActorContext(a, context.Background(), system, prop, mailboxPID, actorChan)

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Actor recovered, need error handling:", r)
				system.SendSystemMessage(actorContext.self, SystemMessage{Type: SuspendMailbox})
				actorContext.SuspendChildren()
				if actorContext.Parent() != nil {
					failure := Failure{Reason: r, Who: *actorContext.Self(), Actor: a, ActorContext: actorContext, ActorChan: actorChan}
					system.SendSystemMessage(*actorContext.props.parent, SystemMessage{Type: SystemMessageFailure, Extras: failure})
				} else {
					actorContext.props.rootStrategy.HandleFailure(system, actorContext, Failure{Reason: r, Who: *actorContext.Self(), Actor: a, ActorContext: actorContext, ActorChan: actorChan})
				}
			}
		}()

		system.SendSystemMessage(actorContext.self, SystemMessage{Type: SystemMessageStart})

		for {
			envelope := <-actorChan
			//Set only message and send
			actorContext.AddEnvelope(envelope)
			a.Receive(*actorContext)

			if msg, ok := envelope.Message.(SystemMessage); ok {
				actorContext.HandleSystemMessage(msg)
				if actorContext.state == actorStop {
					return
				}
			}

		}
	}()
}

func startActorWithContext(a Actor, system *ActorSystem, actorContext *ActorContext, actorChan chan Envelope) {
	go func() {

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Actor recovered, need error handling:", r)
				system.SendSystemMessage(actorContext.self, SystemMessage{Type: SuspendMailbox})
				if actorContext.Parent() != nil {
					failure := Failure{Reason: r, Who: *actorContext.Self(), Actor: a, ActorContext: actorContext, ActorChan: actorChan}

					actorContext.SuspendChildren()
					system.SendSystemMessage(*actorContext.props.parent, SystemMessage{Type: SystemMessageFailure, Extras: failure})
				} else {
					actorContext.props.rootStrategy.HandleFailure(system, actorContext, Failure{Reason: r, Who: *actorContext.Self(), Actor: a, ActorContext: actorContext, ActorChan: actorChan})
				}
			}
		}()

		for {
			envelope := <-actorChan

			actorContext.AddEnvelope(envelope)
			a.Receive(*actorContext)

			if msg, ok := envelope.Message.(SystemMessage); ok {
				actorContext.HandleSystemMessage(msg)
				if actorContext.state == actorStop {
					return
				}
			}
		}
	}()
}

func startMailbox(mailbox *Mailbox) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Mailbox recovered, need restarting:", r)
			}
		}()
		mailbox.Start()
	}()
}

func (system *ActorSystem) Send(envelope Envelope) {
	// fmt.Printf("Send message: %v to receiver: %v\n", envelope.Message, envelope.Receiver())
	ch := system.registry.Find(*envelope.Receiver())
	if ch == nil {
		// fmt.Println("Channel is nil")
		return
	}
	ch <- envelope
}

func (system *ActorSystem) AddRemoteActor(remoteActorPID PID, senderChan chan Envelope) {
	system.registry.Add(remoteActorPID, senderChan)
}

func (system *ActorSystem) SendSystemMessage(receiver PID, msg SystemMessage) {
	envelope := NewEnvelope(msg, receiver)
	// fmt.Println("Send system message:", msg)
	// fmt.Println("Send system message to:", receiver)
	system.Send(envelope)
}

func (system *ActorSystem) RemoveActor(receiver PID, msg SystemMessage) {
	system.SendSystemMessage(receiver, msg)
	system.registry.Remove(receiver)
}

func (system *ActorSystem) Stop(pid PID) {
	envelope := NewEnvelope(SystemMessage{Type: SystemMessageStop}, pid)
	system.Send(envelope)
}

func (system *ActorSystem) GracefulStop(pid PID) {
	envelope := NewEnvelope(SystemMessage{Type: SystemMessageGracefulStop}, pid)
	system.Send(envelope)
}

func (system *ActorSystem) Restart(pid PID) {
	envelope := NewEnvelope(SystemMessage{Type: SystemMessageRestart}, pid)
	system.Send(envelope)
}

func (system *ActorSystem) EscalateFailureNotPanic(reason interface{}, pid PID) {
	notPanic := NotPanic{Reason: reason}
	failure := Failure{Reason: notPanic, Who: pid}

	envelope := NewEnvelope(SystemMessage{Type: SystemMessageEscalateFailure, Extras: failure}, pid)
	system.Send(envelope)
}

func (system *ActorSystem) EscalateFailurePanic(reason interface{}, pid PID) {
	failure := Failure{Reason: reason, Who: pid}

	envelope := NewEnvelope(SystemMessage{Type: SystemMessageEscalateFailure, Extras: failure}, pid)
	system.Send(envelope)
}
