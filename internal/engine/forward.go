package engine

import (
	"io"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// forward pipes data bidirectionally between a local TCP connection and a
// remote endpoint through an SSH channel. It handles half-close properly so
// that both sides see EOF.
func forward(localConn net.Conn, sshClient *ssh.Client, remoteAddr string) {
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		log.Debug().Err(err).
			Str("remote", remoteAddr).
			Msg("failed to dial remote via SSH")
		localConn.Close()
		return
	}

	pipeConns(localConn, remoteConn)
}

// forwardReverse pipes data between a connection accepted on the remote SSH
// listener and a locally-dialed TCP connection. This is the data path for
// remote forwards (ssh -R equivalent).
func forwardReverse(remoteConn net.Conn, localAddr string) {
	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		log.Debug().Err(err).
			Str("local", localAddr).
			Msg("failed to dial local target")
		remoteConn.Close()
		return
	}

	pipeConns(localConn, remoteConn)
}

// pipeConns copies data bidirectionally between two connections with proper
// half-close handling.
func pipeConns(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// a → b
	go func() {
		defer wg.Done()
		_, err := io.Copy(b, a)
		if err != nil {
			log.Trace().Err(err).Msg("a→b copy ended")
		}
		if tc, ok := b.(*net.TCPConn); ok {
			tc.CloseWrite()
		} else {
			b.Close()
		}
	}()

	// b → a
	go func() {
		defer wg.Done()
		_, err := io.Copy(a, b)
		if err != nil {
			log.Trace().Err(err).Msg("b→a copy ended")
		}
		if tc, ok := a.(*net.TCPConn); ok {
			tc.CloseWrite()
		} else {
			a.Close()
		}
	}()

	wg.Wait()
	a.Close()
	b.Close()
}
