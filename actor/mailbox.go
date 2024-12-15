package actor

type mailboxState int32

const (
	mailboxRunning mailboxState = iota
	mailboxSuspended
)

type Mailbox struct {
	actorChan   chan Envelope
	mailboxChan chan Envelope
	systemQueue []Envelope
	queue       []Envelope
	state       mailboxState
}

func NewMailbox(actorChan chan Envelope) *Mailbox {
	m := &Mailbox{
		actorChan:   actorChan,
		mailboxChan: make(chan Envelope),
		systemQueue: make([]Envelope, 0),
		queue:       make([]Envelope, 0),
		state:       mailboxSuspended,
	}
	return m
}

func (m *Mailbox) buffer(envelope Envelope) {

	switch envelope.Message.(type) {
	case SystemMessage:
		m.systemQueue = append(m.systemQueue, envelope)
	default:
		if m.state == mailboxRunning {
			m.queue = append(m.queue, envelope)
		}
	}
}

func (m *Mailbox) getEnvelopeFromSystemQueue() Envelope {
	defer func() {
		m.systemQueue = m.systemQueue[1:]
	}()
	return m.systemQueue[0]
}

func (m *Mailbox) getEnvelopeFromQueue() Envelope {
	defer func() {
		m.queue = m.queue[1:]
	}()
	return m.queue[0]
}

// modified old version with sys queue
func (m *Mailbox) Start() {
	m.state = mailboxRunning
	var newEnvelope Envelope
	var haveReady bool = false
	for {
		for haveReady {
			if m.state == mailboxSuspended {
				select {
				case m.actorChan <- newEnvelope:
					if len(m.systemQueue) > 0 {
						newEnvelope = m.getEnvelopeFromSystemQueue()
						switch msg := newEnvelope.Message.(type) {
						case SystemMessage:
							if msg.Type == DeleteMailbox {
								m.delete()
								return
							}
							m.buffer(newEnvelope)
						}
					} else {
						haveReady = false
					}
				default:
					//ignore
				}
			} else {
				select {
				case m.actorChan <- newEnvelope:
					if len(m.systemQueue) > 0 {
						newEnvelope = m.getEnvelopeFromSystemQueue()
					} else if len(m.queue) > 0 {
						newEnvelope = m.getEnvelopeFromQueue()
					} else {
						haveReady = false
					}
				case envelope := <-m.mailboxChan:
					switch msg := envelope.Message.(type) {
					case SystemMessage:
						if msg.Type == DeleteMailbox {
							// fmt.Println("DeleteMailbox")
							m.delete()
							return
						} else if msg.Type == SuspendMailbox || msg.Type == SuspendMailboxAll {
							// fmt.Println("Suspend Mailbox: ", m.state)
							m.state = mailboxSuspended
						} else if msg.Type == ResumeMailbox || msg.Type == ResumeMailboxAll {
							// fmt.Println("Resume Mailbox: ", m.state)
							m.state = mailboxRunning
						}
					}
					m.buffer(envelope)
				}
			}
		}
		newEnvelope = <-m.mailboxChan
		switch msg := newEnvelope.Message.(type) {
		case SystemMessage:
			if msg.Type == DeleteMailbox {
				// fmt.Println("DeleteMailbox")
				m.delete()
				return
			} else if msg.Type == SuspendMailbox || msg.Type == SuspendMailboxAll {
				// fmt.Println("Suspend Mailbox: ", m.state)
				m.state = mailboxSuspended
			} else if msg.Type == ResumeMailbox || msg.Type == ResumeMailboxAll {
				// fmt.Println("Resume Mailbox: ", m.state)
				m.state = mailboxRunning
			}
		}
		if m.state == mailboxRunning {
			haveReady = true
		} else {
			switch newEnvelope.Message.(type) {
			case SystemMessage:
				haveReady = true
			default:
				//ignore
			}
		}
	}
}

func (m *Mailbox) GetChan() chan Envelope {
	return m.mailboxChan
}

func (m *Mailbox) delete() {
	close(m.actorChan)
	clear(m.queue)
	clear(m.systemQueue)
}
