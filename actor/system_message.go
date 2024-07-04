package actor

type SystemMessageType int

const (
	SystemMessageStart SystemMessageType = iota
	SystemMessageStop
	SystemMessageGracefulStop
	SystemMessageChildTerminated
	DeleteMailbox
	SystemMessageFailure
	SystemMessageRestart
	SuspendMailbox
	ResumeMailbox
	SuspendMailboxAll
	ResumeMailboxAll
	SystemMessageEscalateFailure
)

type SystemMessage struct {
	Type   SystemMessageType
	Extras interface{}
}

type Failure struct {
	Who          PID
	Reason       interface{}
	Actor        Actor
	ActorContext *ActorContext
	ActorChan    chan Envelope
}

type NotPanic struct {
	Reason interface{}
}
