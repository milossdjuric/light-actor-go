package actor

type restartAllStrategy struct{}

type stopAllStrategy struct{}

func NewRestartAllStrategy() *restartAllStrategy {
	return &restartAllStrategy{}
}

func NewStopAllStrategy() *stopAllStrategy {
	return &stopAllStrategy{}
}

func (strategy *restartAllStrategy) HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure) {

	// fmt.Println("Restarting all actors")

	children := supervisor.Children()

	switch failure.Reason.(type) {
	case NotPanic:
		failureChildren := failure.ActorContext.Children()
		for _, child := range failureChildren {
			actorSystem.Stop(*child)
		}
		for _, child := range children {
			actorSystem.Restart(*child)
		}
	// default case is for panic
	default:
		actorSystem.RespawnActor(failure.Actor, failure.Who, failure.ActorChan, *failure.ActorContext.Props())
		failureChildren := failure.ActorContext.Children()
		for _, child := range failureChildren {
			actorSystem.Stop(*child)
		}
		for _, child := range children {
			if *child != failure.Who {
				actorSystem.Restart(*child)
			}
		}
	}
}

func (strategy *stopAllStrategy) HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure) {

	// fmt.Println("Stopping all actors")

	children := supervisor.Children()

	switch failure.Reason.(type) {
	case NotPanic:
		for _, child := range children {
			supervisor.RemoveChild(*child)
			actorSystem.Send(NewEnvelope(SystemMessage{Type: SystemMessageStop}, *child))
		}
	// default case is for panic
	default:
		actorSystem.RemoveActor(failure.Who, SystemMessage{Type: DeleteMailbox})
		supervisor.RemoveChild(failure.Who)
		failureChildren := failure.ActorContext.Children()
		for _, child := range failureChildren {
			actorSystem.Stop(*child)
		}
		for _, child := range children {
			if *child != failure.Who {
				supervisor.RemoveChild(*child)
				actorSystem.Send(NewEnvelope(SystemMessage{Type: SystemMessageStop}, *child))
			}
		}
	}
}
