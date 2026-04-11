package engine

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// SOCKS5 protocol constants.
const (
	socks5Version    = 0x05
	socks5AuthNone   = 0x00
	socks5CmdConnect = 0x01
	socks5AtypIPv4   = 0x01
	socks5AtypDomain = 0x03
	socks5AtypIPv6   = 0x04
	socks5RepSuccess = 0x00
	socks5RepFail    = 0x01
)

// serveSocks5 runs a SOCKS5 proxy on the given listener, routing all
// connections through the SSH client (like ssh -D).
func serveSocks5(listener net.Listener, sshClient *ssh.Client, tunnelName string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return // Listener closed.
		}
		go handleSocks5(conn, sshClient, tunnelName)
	}
}

func handleSocks5(conn net.Conn, sshClient *ssh.Client, tunnelName string) {
	defer conn.Close()

	// 1. Auth negotiation.
	buf := make([]byte, 258)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	if buf[0] != socks5Version {
		return
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return
	}
	// We only support no-auth.
	conn.Write([]byte{socks5Version, socks5AuthNone})

	// 2. Request.
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[0] != socks5Version || buf[1] != socks5CmdConnect {
		sendSocks5Reply(conn, socks5RepFail)
		return
	}

	addrType := buf[3]
	var host string
	switch addrType {
	case socks5AtypIPv4:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case socks5AtypDomain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		domainLen := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:domainLen]); err != nil {
			return
		}
		host = string(buf[:domainLen])
	case socks5AtypIPv6:
		if _, err := io.ReadFull(conn, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	default:
		sendSocks5Reply(conn, socks5RepFail)
		return
	}

	// Read port (2 bytes, big-endian).
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(buf[:2])
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))

	// 3. Connect through SSH.
	remote, err := sshClient.Dial("tcp", target)
	if err != nil {
		log.Debug().Err(err).Str("tunnel", tunnelName).Str("target", target).Msg("socks5: dial failed")
		sendSocks5Reply(conn, socks5RepFail)
		return
	}

	// Send success reply.
	sendSocks5Reply(conn, socks5RepSuccess)

	// 4. Relay.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(remote, conn)
		if tc, ok := remote.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(conn, remote)
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()
	wg.Wait()
	remote.Close()
}

func sendSocks5Reply(conn net.Conn, rep byte) {
	// Minimal SOCKS5 reply: version, rep, rsv, atyp=ipv4, bind_addr=0.0.0.0:0
	reply := []byte{socks5Version, rep, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0}
	conn.Write(reply)
}

