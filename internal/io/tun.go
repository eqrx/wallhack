// Copyright (C) 2021 Alexander Sowitzki
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

package io

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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
	// errIoctl indicates that our ioctl failed.
	errIoctl = errors.New("ioctl failed")
	// errNameTooLong indicates that the given tun name is too long.
	errNameTooLong = fmt.Errorf("tun name is longer than %d bytes", IfaceNameMaxLen)
	// ErrMTU indicates that a packet is too large for the tun MTU.
	ErrMTU = errors.New("packet too large for MTU")
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
		return nil, fmt.Errorf("%w: %d", errNameTooLong, len(nameBytes))
	}

	copy(request.name[:], nameBytes)

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.LittleEndian, &request); err != nil {
		panic(fmt.Sprintf("failed to serialize request: %v", err))
	}

	data := buf.Bytes()

	if len(data) != requestLen {
		panic(fmt.Sprintf("unexpected request len %v", requestLen))
	}

	return data, nil
}

// tun is a handle for a linux tun device that allows reading an writing frames.
type tun struct {
	// f is the file handle that is attached to the tun.
	f *os.File
	// mtu is the maximum frame size the tun can handle.
	mtu int
}

// newTun creates a new tun handle for the tun named by ifaceName.
func newTun(ifaceName string) (*tun, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("tun device %s needs to be created by the network manager: %w", ifaceName, err)
	}

	tunFD, err := unix.Open(tunPath, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open tun device: %w", err)
	}

	req, err := newTunRequest(ifaceName, tunFlag|noPiFlag)
	if err != nil {
		return nil, fmt.Errorf("create tun request: %w", err)
	}

	// Unholy-ish magic to get a C pointer for ioctl call.
	reqPtr := uintptr(unsafe.Pointer(&req[0])) //nolint:gosec // Needed for ioctl.

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(tunFD), ioctlNumber, reqPtr)

	if errno != 0 {
		if closeErr := unix.Close(tunFD); closeErr != nil {
			return nil, fmt.Errorf("%w: code %v. close tun device after error: %v", errIoctl, errno, closeErr)
		}

		return nil, fmt.Errorf("%w: code %v", errIoctl, errno)
	}

	return &tun{os.NewFile(uintptr(tunFD), tunPath), iface.MTU}, nil
}

// Close closes the underlying tun file.
func (t *tun) Close() error {
	if err := t.f.Close(); err != nil {
		return fmt.Errorf("close tun fd: %w", err)
	}

	return nil
}

// readIPFrame reads a complete ip frame from the tun and returnes any io error.
func (t *tun) readIPFrame() ([]byte, error) {
	packet := make([]byte, t.mtu+1)

	bytesRead, err := t.f.Read(packet)
	if err != nil {
		return nil, fmt.Errorf("read from tun: %w", err)
	}

	if bytesRead == len(packet)+1 {
		panic("tun read packet that is larger than MTU")
	}

	return packet[:bytesRead], nil
}

// writeIPFrame writes an IP frame to the tun and returns any io error.
func (t *tun) writeIPFrame(packet []byte) error {
	if len(packet) > t.mtu {
		return fmt.Errorf("%w: MTU is %d, packet size is %d", ErrMTU, t.mtu, len(packet))
	}

	bytesWritten, err := t.f.Write(packet)
	if err != nil {
		return fmt.Errorf("write packet to tun: %w", err)
	}

	if bytesWritten != len(packet) {
		panic(fmt.Sprintf("tun write with len %d with MTU %d returned %d written", len(packet), t.mtu, bytesWritten))
	}

	return nil
}
