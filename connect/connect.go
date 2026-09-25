package connect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"gabagool/config"

	"github.com/tailscale/tailcat"
)

const tailcatPort = 5713

func Connect(ctx context.Context, route config.RecvRoute) error {
	log.Printf("Connecting to tunnel %s", route.Name)

	client := tailcat.NewClient(tailcat.Addr(route.Server))
	defer client.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	workers := 0

	if route.Tcp {
		workers++

		go func() {
			errCh <- serveTCP(ctx, client, route.Addr)
		}()
	}

	if route.Udp {
		workers++

		go func() {
			errCh <- serveUDP(ctx, client, route.Addr)
		}()
	}

	if workers == 0 {
		return errors.New("route has neither TCP nor UDP enabled")
	}

	var firstErr error

	select {
	case firstErr = <-errCh:
	case <-ctx.Done():
		firstErr = ctx.Err()
	}

	cancel()

	remaining := workers
	if firstErr != ctx.Err() {
		remaining--
	}

	for i := 0; i < remaining; i++ {
		err := <-errCh

		if firstErr == nil && err != nil {
			firstErr = err
		}
	}

	if errors.Is(firstErr, context.Canceled) {
		return nil
	}

	return firstErr
}

func serveTCP(
	ctx context.Context,
	client *tailcat.Client,
	addr string,
) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("TCP listening on %q", addr)

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		stream, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return err
		}

		go func(stream net.Conn) {
			if err := handleTCP(ctx, client, stream); err != nil {
				if ctx.Err() == nil {
					log.Printf(
						"TCP connection %s failed: %s",
						stream.RemoteAddr(),
						err,
					)
				}
			}
		}(stream)
	}
}

func handleTCP(
	ctx context.Context,
	client *tailcat.Client,
	stream net.Conn,
) error {
	defer stream.Close()

	conn, err := client.DialTCPPort(ctx, tailcatPort)
	if err != nil {
		return err
	}
	defer conn.Close()

	errCh := make(chan error, 2)

	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			_ = stream.Close()
			_ = conn.Close()

		case <-done:
		}
	}()

	// Local TCP client -> Tailcat.
	go func() {
		_, err := io.Copy(conn, stream)
		errCh <- err
	}()

	// Tailcat -> local TCP client.
	go func() {
		_, err := io.Copy(stream, conn)
		errCh <- err
	}()

	firstErr := <-errCh

	// One direction ended. Tear down the other direction as well.
	_ = stream.Close()
	_ = conn.Close()

	// Wait for the second io.Copy to wake up and exit.
	secondErr := <-errCh

	close(done)

	if ctx.Err() != nil {
		return nil
	}

	if firstErr != nil {
		return firstErr
	}

	return secondErr
}

type udpSession struct {
	conn tailcat.ConnPacketConn
	peer net.Addr
}

func serveUDP(
	ctx context.Context,
	client *tailcat.Client,
	addr string,
) error {
	sock, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer sock.Close()

	log.Printf("UDP listening on %q", addr)

	var (
		mu       sync.Mutex
		sessions = make(map[string]*udpSession)
	)

	go func() {
		<-ctx.Done()

		_ = sock.Close()

		mu.Lock()
		defer mu.Unlock()

		for _, session := range sessions {
			_ = session.conn.Close()
		}
	}()

	removeSession := func(key string, session *udpSession) {
		mu.Lock()
		defer mu.Unlock()

		if sessions[key] == session {
			delete(sessions, key)
		}
	}

	startReader := func(key string, session *udpSession) {
		go func() {
			defer func() {
				removeSession(key, session)
				_ = session.conn.Close()
			}()

			buf := make([]byte, 65535)

			for {
				n, err := session.conn.Read(buf)
				if err != nil {
					if ctx.Err() == nil {
						log.Printf(
							"UDP session %s read failure: %s",
							session.peer,
							err,
						)
					}

					return
				}

				if _, err := sock.WriteTo(buf[:n], session.peer); err != nil {
					if ctx.Err() == nil {
						log.Printf(
							"UDP session %s write failure: %s",
							session.peer,
							err,
						)
					}

					return
				}
			}
		}()
	}

	buf := make([]byte, 65535)

	for {
		n, peer, err := sock.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return err
		}

		key := peer.String()

		mu.Lock()
		session := sessions[key]
		mu.Unlock()

		if session == nil {
			conn, err := client.DialUDPPort(ctx, tailcatPort)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}

				log.Printf(
					"UDP session %s dial failure: %s",
					peer,
					err,
				)

				continue
			}

			session = &udpSession{
				conn: conn,
				peer: peer,
			}

			mu.Lock()

			if existing := sessions[key]; existing != nil {
				mu.Unlock()

				_ = conn.Close()
				session = existing
			} else {
				sessions[key] = session
				mu.Unlock()

				startReader(key, session)
			}
		}

		if _, err := session.conn.Write(buf[:n]); err != nil {
			removeSession(key, session)
			_ = session.conn.Close()

			if ctx.Err() != nil {
				return nil
			}

			log.Printf(
				"UDP session %s write failure: %s",
				peer,
				err,
			)

			continue
		}
	}
}

func wrapErr(prefix string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", prefix, err)
}
