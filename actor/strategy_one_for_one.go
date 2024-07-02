package actor

type restartOneStrategy struct{}

type stopOneStrategy struct{}

type escalateStrategy struct{}

type resumeOneStrategy struct{}

func NewRestartOneStrategy() *restartOneStrategy {
	return &restartOneStrategy{}
}

func NewStopOneStrategy() *stopOneStrategy {
	return &stopOneStrategy{}
}

func NewEscalateStrategy() *escalateStrategy {
	return &escalateStrategy{}
}

func NewResumeOneStrategy() *resumeOneStrategy {
	return &resumeOneStrategy{}
}

func (strategy *restartOneStrategy) HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure) {

	// fmt.Println("Restarting actor")
	switch failure.Reason.(type) {
	case NotPanic:
		children := failure.ActorContext.Children()
		for _, child := range children {
			actorSystem.Stop(*child)
		}
		actorSystem.Restart(failure.Who)
	// default case is for panic
	default:
		children := failure.ActorContext.Children()
		for _, child := range children {
			actorSystem.Stop(*child)
		}
		actorSystem.RespawnActor(failure.Actor, failure.Who, failure.ActorChan, *failure.ActorContext.Props())
	}
}

func (strategy *stopOneStrategy) HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure) {

	// fmt.Println("Stopping actor")
	switch failure.Reason.(type) {
	case NotPanic:
		actorSystem.Stop(failure.Who)
		supervisor.RemoveChild(failure.Who)
	// default case is for panic
	default:
		actorSystem.RemoveActor(failure.Who, SystemMessage{Type: DeleteMailbox})
		supervisor.RemoveChild(failure.Who)
		children := failure.ActorContext.Children()
		for _, child := range children {
			supervisor.RemoveChild(*child)
			actorSystem.Stop(*child)
		}
	}
}

func (strategy *escalateStrategy) HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure) {

	// fmt.Println("Escalating failure to upper supervisor")
	switch failure.Reason.(type) {
	// default case is for panic
	case NotPanic:
		supervisor.EscalateFailure(failure)
	default:
		actorSystem.RemoveActor(failure.Who, SystemMessage{Type: DeleteMailbox})
		supervisor.RemoveChild(failure.Who)
		supervisor.EscalateFailure(failure)
	}
}

func (strategy *resumeOneStrategy) HandleFailure(actorSystem *ActorSystem, supervisor Supervisor, failure Failure) {

	// fmt.prinlnt("Resuming actor")
	switch failure.Reason.(type) {
	// default case, shouldn't be for panic
	default:
		actorSystem.SendSystemMessage(failure.Who, SystemMessage{Type: ResumeMailboxAll})
	}
}
