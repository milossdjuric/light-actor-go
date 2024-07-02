package actor

type mailboxState int32

const (
	mailboxRunning mailboxState = iota
	mailboxSuspended
)

type Mailbox struct {
	actorChan      chan Envelope
	mailboxChan    chan Envelope
	queue          []Envelope
	suspendedQueue []Envelope
	state          mailboxState
}

func NewMailbox(actorChan chan Envelope) *Mailbox {
	m := &Mailbox{
		actorChan:      actorChan,
		mailboxChan:    make(chan Envelope),
		queue:          make([]Envelope, 0),
		suspendedQueue: make([]Envelope, 0),
		state:          mailboxSuspended,
	}
	return m
}

func (m *Mailbox) buffer(envelope Envelope) {
	if m.state == mailboxRunning {
		m.queue = append(m.queue, envelope)
		return
	} else if m.state == mailboxSuspended {
		switch envelope.Message.(type) {
		case SystemMessage:
			m.queue = append(m.queue, envelope)
			return
		default:
			// not adding envelope to buffer
		}
	}

}

func (m *Mailbox) getEnvelope() Envelope {
	defer func() {
		m.queue = m.queue[1:]
	}()
	return m.queue[0]
}

func (m *Mailbox) Start() {
	m.state = mailboxRunning
	var newEnvelope Envelope
	var haveReady bool = false
	for {
		for haveReady {
			if m.state == mailboxSuspended {

				select {
				case m.actorChan <- newEnvelope:
					if len(m.queue) > 0 {
						newEnvelope = m.getEnvelope()
					} else {
						haveReady = false
					}
				case envelope := <-m.mailboxChan:
					switch msg := envelope.Message.(type) {
					case SystemMessage:
						if msg.Type == DeleteMailbox {
							m.delete()
							return
						}
						m.buffer(envelope)
					default:
						// add to suspended queue
						m.suspendedQueue = append(m.suspendedQueue, envelope)
					}
				}

			} else {
				if len(m.suspendedQueue) > 0 {
					m.queue = append(m.queue, m.suspendedQueue...)
					m.suspendedQueue = m.suspendedQueue[1:]
				}
				select {
				case m.actorChan <- newEnvelope:
					if len(m.queue) > 0 {
						newEnvelope = m.getEnvelope()
					} else {
						haveReady = false
					}
				case envelope := <-m.mailboxChan:
					switch msg := envelope.Message.(type) {
					case SystemMessage:
						if msg.Type == DeleteMailbox {
							// fmt.Println("Delete mailbox")
							m.delete()
							return
						} else if msg.Type == SuspendMailbox || msg.Type == SuspendMailboxAll || msg.Type == SystemMessageGracefulStop {
							m.state = mailboxSuspended
							// fmt.Println("Suspend mailbox: ", m.state)
						} else if msg.Type == ResumeMailbox || msg.Type == ResumeMailboxAll {
							m.state = mailboxRunning
							// fmt.Println("Resume mailbox: ", m.state)
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
				// fmt.Println("Delete mailbox")
				m.delete()
				return
			} else if msg.Type == SuspendMailbox || msg.Type == SuspendMailboxAll || msg.Type == SystemMessageGracefulStop {
				m.state = mailboxSuspended
				// fmt.Println("Suspend mailbox: ", m.state)
			} else if msg.Type == ResumeMailbox || msg.Type == ResumeMailboxAll {
				m.state = mailboxRunning
				// fmt.Println("Resume mailbox: ", m.state)
			}
		}
		if m.state == mailboxRunning {
			haveReady = true
		} else {
			switch newEnvelope.Message.(type) {
			case SystemMessage:
				haveReady = true
			default:
				m.suspendedQueue = append(m.suspendedQueue, newEnvelope)
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
}
