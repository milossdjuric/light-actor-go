package actor

type Actor interface {
	Receive(ctx ActorContext)
}
