package actor

import (
	"time"
)

type mailboxState int32

const (
	mailboxRunning mailboxState = iota
	mailboxSuspended
)

type Mailbox struct {
	actorChan   chan Envelope
	mailboxChan chan Envelope
	queue       []Envelope
	state       mailboxState
}

func NewMailbox(actorChan chan Envelope) *Mailbox {
	m := &Mailbox{
		actorChan:   actorChan,
		mailboxChan: make(chan Envelope),
		queue:       make([]Envelope, 0),
		state:       mailboxSuspended,
	}
	return m
}

func (m *Mailbox) buffer(msg Envelope) {
	m.queue = append(m.queue, msg)
}

func (m *Mailbox) getEnvelope() Envelope {
	if len(m.queue) > 0 {
		return m.queue[0]
	}
	return Envelope{}
}

func (m *Mailbox) removeFirstEnvelope() {
	if len(m.queue) > 0 {
		m.queue = m.queue[1:]
	}
}

func (m *Mailbox) moveToBack() {
	if len(m.queue) > 0 {
		m.queue = append(m.queue[1:], m.queue[0])
	}
}

func (m *Mailbox) Start() {
	m.state = mailboxRunning

	for {
		select {
		case envelope := <-m.mailboxChan:
			// fmt.Println("Mailbox received message:", envelope.Message)
			m.buffer(envelope)
		default:
			if len(m.queue) > 0 {
				envelope := m.getEnvelope()
				if msg, ok := envelope.Message.(SystemMessage); ok {
					// Handle state transitions before sending the message to actorChan
					switch msg.Type {
					case DeleteMailbox:
						// fmt.Println("Deleting mailbox")
						m.delete()
						return
					case SuspendMailbox, SuspendMailboxAll, SystemMessageGracefulStop:
						m.state = mailboxSuspended
						// fmt.Println("Suspending mailbox state:", m.state)
					case ResumeMailbox, ResumeMailboxAll:
						m.state = mailboxRunning
						// fmt.Println("Resuming mailbox state:", m.state)
					}

					select {
					case m.actorChan <- envelope:
						// fmt.Println("Sending system message to actor channel:", msg)
						m.removeFirstEnvelope()
					case <-time.After(100 * time.Millisecond):
						// fmt.Println("Failed to send system message to actor channel, timed out:", msg)
						m.moveToBack()
					}
				} else if m.state == mailboxRunning {
					select {
					case m.actorChan <- envelope:
						// fmt.Println("Sending to actor channel:", envelope.Message)
						m.removeFirstEnvelope()
					case <-time.After(100 * time.Millisecond):
						// fmt.Println("Failed to send message to actor channel, timed out:", envelope.Message)
						m.moveToBack()
					}
				} else if m.state == mailboxSuspended {
					m.moveToBack()
					time.Sleep(100 * time.Millisecond)
					// fmt.Println("Non-system message moved to back of queue:", envelope.Message)
				}
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
