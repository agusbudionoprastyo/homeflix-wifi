//go:build linux

package services

import (
	"net"
	"os"
	"syscall"
)

// enableDirectIO applies O_DIRECT on Linux file descriptors to bypass the page cache.
func enableDirectIO(fd int) {
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_SETFL, syscall.O_DIRECT)
}

// mmapSharedPopulateFlags returns Linux mmap flags with pre-population enabled.
func mmapSharedPopulateFlags() int {
	return syscall.MAP_SHARED | syscall.MAP_POPULATE
}

// transferSendfileOrCopy uses Linux sendfile(2) to transfer bytes directly from file to tcpConn.
func transferSendfileOrCopy(tcpConn *net.TCPConn, file *os.File, offset *int64, count int) (int, error) {
	tcpFile, err := tcpConn.File()
	if err != nil {
		return 0, err
	}
	defer tcpFile.Close()
	return syscall.Sendfile(int(tcpFile.Fd()), int(file.Fd()), offset, count)
}
