package kqueue

import (
	"fmt"
	"syscall"

	"github.com/masonictemple4/kserver/socket"
)

type EventLoop struct {
	KqueueFileDescriptor int
	SocketFileDescriptor int
}

func NewEventLoop(s *socket.Socket) (*EventLoop, error) {
	// Creates a new kernel event queue and returns the
	// its file descriptor. This enables us to interact with
	// with this Kqueue with the kevent system call.
	kQueue, err := syscall.Kqueue()
	if err != nil {
		return nil, fmt.Errorf("failed to create a kqueue file descriptor (%v)", err)
	}

	// kevent provides two functionalities:
	//  - subscribing to new events
	//  - and polling.

	// To sub to incoming connection events, we can pass a kevent struct
	// to kevent systemcall. Containing the following:
	//  - ident: file descriptor set to the socket file.
	//  - A filter that processes the event. Set to EVFILT_READ, which,
	//  when used while listening to a socket indicates we're interested in
	//  incoming connection events.
	//  - Flags that indicate actions to perform with this event. (EV_ADD add to queue,
	//  i.e subbing to it, and enable EV_ENABLE). Flags can be combined using bitwise or.
	changeEvent := syscall.Kevent_t{
		Ident:  uint64(s.FileDescriptor),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}

	changeEventRegistered, err := syscall.Kevent(
		kQueue,
		[]syscall.Kevent_t{changeEvent},
		nil,
		nil,
	)

	if err != nil || changeEventRegistered == -1 {
		return nil, fmt.Errorf("failed to register change event (%v)", err)
	}

	return &EventLoop{
		KqueueFileDescriptor: kQueue,
		SocketFileDescriptor: s.FileDescriptor,
	}, nil

}

type Handler func(s *socket.Socket)

func (eventLoop *EventLoop) Handle(handler Handler) {
	println("we made it to he event loop")
	for {
		newEvents := make([]syscall.Kevent_t, 10)
		numNewEvents, err := syscall.Kevent(
			eventLoop.KqueueFileDescriptor,
			nil,
			newEvents,
			nil,
		)
		if err != nil {
			continue
		}

		for i := 0; i < numNewEvents; i++ {
			currentEvent := newEvents[i]
			eventFileDescriptor := int(currentEvent.Ident)

			if currentEvent.Flags&syscall.EV_EOF != 0 {
				// client closed the connection.
				syscall.Close(eventFileDescriptor)
			} else if eventFileDescriptor == eventLoop.SocketFileDescriptor {
				// new inbound connection
				// Pops the connection request off the kqueue of pending TCP connections
				// and creates a new socket and file for that descriptor.
				socketConnection, _, err := syscall.Accept(eventFileDescriptor)
				if err != nil {
					println("an error occured recieving a socket connection", err)
					continue
				}
				socketEvent := syscall.Kevent_t{
					Ident:  uint64(socketConnection),
					Filter: syscall.EVFILT_READ,
					Flags:  syscall.EV_ADD,
					Fflags: 0,
					Data:   0,
					Udata:  nil,
				}

				// Sub to a new EVFILT_READ event. On accept sockets,
				// EVFILT_READ events happen whenever there is data to read on
				// the socket.
				socketEventRegistered, err := syscall.Kevent(
					eventLoop.KqueueFileDescriptor,
					[]syscall.Kevent_t{socketEvent},
					nil,
					nil,
				)

				if err != nil || socketEventRegistered == -1 {
					println("might be a problem here", err)
					continue
				}
			} else if currentEvent.Filter&syscall.EVFILT_READ != 0 {
				// Lastly, we handle events that contain the file descriptor of the client socket
				// wrapping it in a socket and passint it to the handler function.
				handler(&socket.Socket{FileDescriptor: int(eventFileDescriptor)})
			}
		}

	}
}
