package socket

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"syscall"
)

// Unix operating systems represent sockets as files.
// We'll need ot know the file descriptor to interact with the
// socket.
type Socket struct {
	FileDescriptor int
}

// To read, write and close with our socket we'll use
// the following interfaces:
//  - io.Reader
//  - io.Writer
//  - io.Closer

var _ io.Reader = (*Socket)(nil)
var _ io.Writer = (*Socket)(nil)
var _ io.Closer = (*Socket)(nil)

func (socket Socket) Read(bytes []byte) (int, error) {
	if len(bytes) == 0 {
		return 0, nil
	}
	bytesRead, err := syscall.Read(socket.FileDescriptor, bytes)
	if err != nil {
		bytesRead = 0
	}

	return bytesRead, err
}

func (socket Socket) Write(bytes []byte) (int, error) {
	if len(bytes) == 0 {
		return 0, nil
	}
	numBytesWritten, err := syscall.Write(socket.FileDescriptor, bytes)
	if err != nil {
		numBytesWritten = 0
	}
	return numBytesWritten, err
}

func (socket *Socket) Close() error {
	return syscall.Close(socket.FileDescriptor)
}

// For meaningful log messages, we can implement the fmt.Stringer
// interface. Representing a socket by it's file descriptor.
func (s *Socket) String() string {
	return strconv.Itoa(s.FileDescriptor)
}

func Listen(ip string, port int) (*Socket, error) {
	socket := &Socket{}

	socketFileDescriptor, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create a socket (%v)", err)
	}

	socket.FileDescriptor = socketFileDescriptor

	socketAddress := &syscall.SockaddrInet4{Port: port}
	// Create a slice referencing the entire socketAddress.Addr array, copy
	// expects a slice. they're both implemented with the pointer method
	// so it'll reference one another.
	copy(socketAddress.Addr[:], net.ParseIP(ip))

	if err = syscall.Bind(socket.FileDescriptor, socketAddress); err != nil {
		return nil, fmt.Errorf("failed to bind socket (%v)", err)
	}

	if err = syscall.Listen(socket.FileDescriptor, syscall.SOMAXCONN); err != nil {
		return nil, fmt.Errorf("failed to listen on socket (%v)", err)
	}

	return socket, nil
}
