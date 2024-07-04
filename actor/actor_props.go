package actor

type ActorProps struct {
	Parent              *PID
	rootStrategy        FailureStrategy
	supervisionStrategy FailureStrategy
}

func NewActorProps(parent *PID) *ActorProps {
	return &ActorProps{
		Parent:              parent,
		rootStrategy:        defaultRootStrategy,
		supervisionStrategy: defaultSupervisionStrategy,
	}
}

func NewActorPropsWithStrategies(parent *PID, rootStrategy, supervisionStrategy FailureStrategy) *ActorProps {
	return &ActorProps{
		Parent:              parent,
		rootStrategy:        rootStrategy,
		supervisionStrategy: supervisionStrategy,
	}
}

func ConfigureActorProps(props ...ActorProps) *ActorProps {
	if len(props) > 0 {
		return &props[0]
	}
	return defaultConfig()
}

func (prop *ActorProps) AddParent(Parent *PID) {
	prop.Parent = Parent
}

func (prop *ActorProps) SetRootStrategy(strategy FailureStrategy) {
	prop.rootStrategy = strategy
}

func (prop *ActorProps) SetSupervisionStrategy(strategy FailureStrategy) {
	prop.supervisionStrategy = strategy
}

func (prop *ActorProps) RootStrategy() FailureStrategy {
	if prop.rootStrategy == nil {
		return defaultRootStrategy
	}
	return prop.rootStrategy
}

func (prop *ActorProps) SupervisionStrategy() FailureStrategy {
	if prop.supervisionStrategy == nil {
		return defaultSupervisionStrategy
	}
	return prop.supervisionStrategy
}

func defaultConfig() *ActorProps {
	return &ActorProps{
		Parent:              nil,
		rootStrategy:        defaultRootStrategy,
		supervisionStrategy: defaultSupervisionStrategy,
	}
}
