package actor

type FailureStrategy interface {
	HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure)
}

type Supervisor interface {
	Children() []*PID
	Props() *ActorProps
	EscalateFailure(Failure)
	RemoveChild(child PID)
}

var (
	defaultSupervisionStrategy = NewRestartOneStrategy()
	defaultRootStrategy        = NewRestartOneStrategy()
)
