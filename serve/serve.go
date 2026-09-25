package serve

import (
	"context"
	"gabagool/config"
	"io"
	"log"
	"net"

	"github.com/tailscale/tailcat"
)

func Serve(ctx context.Context, route config.SendRoute) error {
	log.Printf("Serving tunnel %s", route.Name)

	psk, err := route.Auth.Psk()
	if err != nil {
		return err
	}

	node, err := route.Auth.Identity()
	if err != nil {
		return err
	}

	server := tailcat.Server{
		Key:          node,
		PresharedKey: psk,

		OnTCP: func(_ uint16) func(net.Conn) {
			return func(c net.Conn) {
				if !route.Tcp {
					_ = c.Close()
					return
				}

				if err := handleTCP(route.Addr, c); err != nil {
					log.Printf("TCP handler failure: %s", err)
				}
			}
		},

		OnUDP: func(_ uint16) func(tailcat.ConnPacketConn) {
			return func(c tailcat.ConnPacketConn) {
				if !route.Udp {
					_ = c.Close()
					return
				}

				if err := handleUDP(route.Addr, c); err != nil {
					log.Printf("UDP handler failure: %s", err)
				}
			}
		},
	}

	defer server.Close()

	if err := server.Start(); err != nil {
		return err
	}

	log.Printf("Listening on %q...", server.TailcatAddr())

	<-ctx.Done()

	return nil
}

func handleTCP(addr string, conn net.Conn) error {
	defer conn.Close()

	stream, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer stream.Close()

	errCh := make(chan error, 2)

	// tailcat -> TCP destination
	go func() {
		_, err := io.Copy(stream, conn)
		errCh <- err
	}()

	// TCP destination -> tailcat
	go func() {
		_, err := io.Copy(conn, stream)
		errCh <- err
	}()

	firstErr := <-errCh

	_ = stream.Close()
	_ = conn.Close()

	<-errCh

	return firstErr
}

func handleUDP(addr string, conn tailcat.ConnPacketConn) error {
	defer conn.Close()

	sock, err := net.Dial("udp", addr)
	if err != nil {
		return err
	}
	defer sock.Close()

	errCh := make(chan error, 2)

	// tailcat -> UDP destination
	go func() {
		buf := make([]byte, 65535)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- err
				return
			}

			if _, err := sock.Write(buf[:n]); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// UDP destination -> tailcat
	go func() {
		buf := make([]byte, 65535)

		for {
			n, err := sock.Read(buf)
			if err != nil {
				errCh <- err
				return
			}

			if _, err := conn.Write(buf[:n]); err != nil {
				errCh <- err
				return
			}
		}
	}()

	firstErr := <-errCh

	_ = sock.Close()
	_ = conn.Close()

	<-errCh

	return firstErr
}
