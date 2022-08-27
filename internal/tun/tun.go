// Copyright (C) 2022 Alexander Sowitzki
//
// This program is free software: you can redistribute it and/or modify it under the terms of the
// GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied
// warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License along with this program.
// If not, see <https://www.gnu.org/licenses/>.

// Package tun interfaces with the linux kernel to provide access to net tuns.
package tun

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// requestFlag is a flag for the ioctl that requests a tun attachment.
type requestFlag int

const (
	// IfaceNameMaxLen is the maximum allowed length of tun interface names in bytes (not runes).
	IfaceNameMaxLen = 16
	// requestLen is the length in bytes of a ioctl request payload.
	requestLen = IfaceNameMaxLen + 2
	// tunFlag indicates to the kernel that we want a tun device (not tap).
	tunFlag requestFlag = 0x0001
	// noPiFlag tells the kernel that we do not want to have packet info prepended to every message coming out of the dev.
	noPiFlag requestFlag = 0x1000
	// ioctlNumber is the number of the ioctl we are doing to get a run.
	ioctlNumber uintptr = 0x400454ca
	// tunPath is the file we do the ioctl on.
	tunPath = "/dev/net/tun"
)

var (
	// ErrMTU indicates that a packet is too large for the tun MTU.
	ErrMTU  = errors.New("packet too large for MTU")
	errName = fmt.Errorf("tun name longer than %d bytes", IfaceNameMaxLen)
)

// request is the ioctl request payload.
type request struct {
	// Name of the tun to attach to.
	name [IfaceNameMaxLen]byte
	// flags for the kernel regarding the attachment.
	flags uint16
}

// newTunRequest creates a new tun request marshalled as bytes.
// It returns an error if the given tun name is too long.
func newTunRequest(name string, flags requestFlag) ([]byte, error) {
	request := request{[IfaceNameMaxLen]byte{}, uint16(flags)}
	nameBytes := []byte(name)

	if len(nameBytes) > len(request.name) {
		return nil, fmt.Errorf("new request: %w: %d", errName, len(nameBytes))
	}

	copy(request.name[:], nameBytes)

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.LittleEndian, &request); err != nil {
		panic(fmt.Errorf("new request: %w", err))
	}

	data := buf.Bytes()

	if len(data) != requestLen {
		panic(fmt.Sprintf("new request: unexpected len %d", requestLen))
	}

	return data, nil
}

// Tun is a handle for a linux Tun device that allows reading an writing frames.
type Tun struct {
	io.ReadWriteCloser
	iface string
}

// New creates a new tun handle for the tun named by ifaceName.
func New(ifaceName string) (*Tun, error) {
	tunFD, err := unix.Open(tunPath, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("new tun: %w", err)
	}

	req, err := newTunRequest(ifaceName, tunFlag|noPiFlag)
	if err != nil {
		return nil, fmt.Errorf("new tun: %w", err)
	}

	// Unholy-ish magic to get a C pointer for ioctl call.
	reqPtr := uintptr(unsafe.Pointer(&req[0]))

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(tunFD), ioctlNumber, reqPtr)

	if errno != 0 {
		if closeErr := unix.Close(tunFD); closeErr != nil {
			return nil, fmt.Errorf("new tun: [%w; %v]", errno, closeErr)
		}

		return nil, fmt.Errorf("new tun: %w", errno)
	}

	return &Tun{os.NewFile(uintptr(tunFD), tunPath), ifaceName}, nil
}

// MTU returns the current MTU size of the interface in bytes.
func (t *Tun) MTU() int {
	iface, err := net.InterfaceByName(t.iface)
	if err != nil {
		panic("tun dev is gone")
	}

	return iface.MTU
}
