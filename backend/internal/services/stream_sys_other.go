//go:build !linux

package services

import (
	"io"
	"net"
	"os"
	"syscall"
)

// enableDirectIO is a no-op on platforms without O_DIRECT.
func enableDirectIO(fd int) {}

// mmapSharedPopulateFlags returns portable shared mmap flags without Linux MAP_POPULATE.
func mmapSharedPopulateFlags() int {
	return syscall.MAP_SHARED
}

// transferSendfileOrCopy falls back to io.CopyN on platforms without Linux sendfile(2).
func transferSendfileOrCopy(tcpConn *net.TCPConn, file *os.File, offset *int64, count int) (int, error) {
	var r io.Reader = file
	if offset != nil {
		r = io.NewSectionReader(file, *offset, int64(count))
	}
	written, err := io.CopyN(tcpConn, r, int64(count))
	if offset != nil {
		*offset += written
	}
	return int(written), err
}
