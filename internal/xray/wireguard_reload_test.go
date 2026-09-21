package xray

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xtls/xray-core/common/buf"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/outbound"
	xwg "github.com/xtls/xray-core/proxy/wireguard"
	"github.com/xtls/xray-core/transport"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
)

// Exercise the real encrypted data path and retain an outbound reference across
// reload, just as an already-dispatched inbound request can do. No production
// keys, external service, kernel TUN, privileges, or fixed host ports are needed.
func TestWireGuardReload(t *testing.T) {
	clientKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	peerKey, port := startWireGuardPeer(t, clientKey.PublicKey())
	config := fmt.Sprintf(`{
		"log":{"loglevel":"none"},
		"outbounds":[{"tag":"wg","protocol":"wireguard","settings":{
			"secretKey":%q,"address":["10.77.0.2/32"],"mtu":1280,
			"noKernelTun":true,"peers":[{"publicKey":%q,
			"endpoint":%q,"allowedIPs":["10.77.0.1/32"]}]
		}}]
	}`, base64.StdEncoding.EncodeToString(clientKey.Bytes()),
		base64.StdEncoding.EncodeToString(peerKey.Bytes()), fmt.Sprintf("127.0.0.1:%d", port))
	engine := New(RuntimeConfig{})
	if err := engine.Apply(config, "generation-0", true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := engine.Apply(config, "final", false); err != nil {
			t.Error(err)
		}
	})
	current := func() *core.Instance {
		engine.mu.Lock()
		defer engine.mu.Unlock()
		return engine.instance.(*core.Instance)
	}
	dest := xnet.TCPDestination(xnet.ParseAddress("10.77.0.1"), 8080)
	if err := wireGuardEcho(current(), dest); err != nil {
		t.Fatalf("initial WireGuard traffic: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for ctx.Err() == nil {
				// Interrupted traffic is expected while its core is being replaced.
				_ = wireGuardEcho(current(), dest)
			}
		})
	}
	t.Cleanup(func() { cancel(); workers.Wait() })

	const reloads = 10
	for generation := 1; generation <= reloads; generation++ {
		// The peer rejects fresh handshakes within its flood-protection window.
		// Keep traffic running, but space restarts beyond that protocol limit.
		time.Sleep(2 * device.HandshakeInitationRate)
		old := current()
		retired := old.GetFeature(outbound.ManagerType()).(outbound.Manager).GetHandler("wg")
		// A distinct hash forces the same stop/start path used for a new spec.
		if err := engine.Apply(config, fmt.Sprintf("generation-%d", generation), true); err != nil {
			t.Fatalf("reload %d: %v", generation, err)
		}
		if err := wireGuardEcho(current(), dest); err != nil {
			t.Fatalf("traffic after reload %d: %v", generation, err)
		}
		// A late dispatch on the old handler must return, never recreate the
		// tunnel or close its bind a second time and panic in this goroutine.
		done := make(chan struct{})
		feedback := &reloadErrorFeedback{}
		go func() {
			defer close(done)
			lateCtx, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			lateCtx = session.ContextWithOutbounds(lateCtx, []*session.Outbound{{Target: dest}})
			lateCtx = session.TrackedConnectionError(lateCtx, feedback)
			retired.Dispatch(lateCtx, &transport.Link{
				Reader: buf.NewReader(bytes.NewReader(nil)),
				Writer: buf.NewWriter(io.Discard),
			})
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("retired handler blocked after reload %d", generation)
		}
		if feedback.err == nil || !strings.Contains(feedback.err.Error(), "proxy/wireguard: closed") {
			t.Fatalf("retired handler error = %v, want WireGuard closed error", feedback.err)
		}
	}
	cancel()
	workers.Wait()
	t.Logf("%d reloads with four traffic workers, successful encrypted echo after every reload, and late dispatch on every retired handler", reloads)
}

type reloadErrorFeedback struct{ err error }

func (f *reloadErrorFeedback) SubmitError(err error) { f.err = err }

func wireGuardEcho(instance *core.Instance, dest xnet.Destination) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := core.Dial(ctx, instance, dest)
	if err != nil {
		return err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	payload := []byte("farang-wireguard-reload")
	if _, err := c.Write(payload); err != nil {
		return err
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(c, got); err != nil {
		return err
	}
	if !bytes.Equal(got, payload) {
		return fmt.Errorf("echo = %q, want %q", got, payload)
	}
	return nil
}

func startWireGuardPeer(t *testing.T, clientKey *ecdh.PublicKey) (*ecdh.PublicKey, int) {
	t.Helper()
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tun, _, stack, err := xwg.CreateNetTUN([]netip.Addr{netip.MustParseAddr("10.77.0.1")}, nil, 1280, true)
	if err != nil {
		t.Fatal(err)
	}
	peer := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))
	t.Cleanup(peer.Close)
	if err := peer.IpcSet(fmt.Sprintf("private_key=%x\nlisten_port=0\npublic_key=%x\nallowed_ip=10.77.0.2/32\n", key.Bytes(), clientKey.Bytes())); err != nil {
		t.Fatal(err)
	}
	if err := peer.Up(); err != nil {
		t.Fatal(err)
	}
	settings, err := peer.IpcGet()
	if err != nil {
		t.Fatal(err)
	}
	port := 0
	for _, line := range strings.Split(settings, "\n") {
		if value, ok := strings.CutPrefix(line, "listen_port="); ok {
			port, err = strconv.Atoi(value)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if port == 0 {
		t.Fatal("WireGuard peer did not bind a UDP port")
	}
	listener, err := gonet.ListenTCP(stack, tcpip.FullAddress{
		NIC: 1, Addr: tcpip.AddrFrom4([4]byte{10, 77, 0, 1}), Port: 8080,
	}, ipv4.ProtocolNumber)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}()
		}
	}()
	return key.PublicKey(), port
}
