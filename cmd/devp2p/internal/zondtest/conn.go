package zondtest

// TODO(now.youtrack.cloud/issue/TGZ-6)
/*
import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/theQRL/go-zond/crypto"
	"github.com/theQRL/go-zond/p2p"
	"github.com/theQRL/go-zond/p2p/rlpx"
	"github.com/theQRL/go-zond/rlp"
	"github.com/theQRL/go-zond/zond/protocols/snap"
	"github.com/theQRL/go-zond/zond/protocols/zond"
)

var (
	pretty = spew.ConfigState{
		Indent:                  "  ",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
		SortKeys:                true,
	}
	timeout = 2 * time.Second
)

// dial attempts to dial the given node and perform a handshake, returning the
// created Conn if successful.
func (s *Suite) dial() (*Conn, error) {
	key, _ := crypto.GenerateKey()
	return s.dialAs(key)
}

// dialAs attempts to dial a given node and perform a handshake using the given
// private key.
func (s *Suite) dialAs(key *ecdsa.PrivateKey) (*Conn, error) {
	tcpEndpoint, _ := s.Dest.TCPEndpoint()
	fd, err := net.Dial("tcp", tcpEndpoint.String())
	if err != nil {
		return nil, err
	}
	conn := Conn{Conn: rlpx.NewConn(fd, s.Dest.Pubkey())}
	conn.ourKey = key
	_, err = conn.Handshake(conn.ourKey)
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.caps = []p2p.Cap{
		{Name: "eth", Version: 67},
		{Name: "eth", Version: 68},
	}
	conn.ourHighestProtoVersion = 68
	return &conn, nil
}

// dialSnap creates a connection with snap/1 capability.
func (s *Suite) dialSnap() (*Conn, error) {
	conn, err := s.dial()
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}
	conn.caps = append(conn.caps, p2p.Cap{Name: "snap", Version: 1})
	conn.ourHighestSnapProtoVersion = 1
	return conn, nil
}

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey                     *ecdsa.PrivateKey
	negotiatedProtoVersion     uint
	negotiatedSnapProtoVersion uint
	ourHighestProtoVersion     uint
	ourHighestSnapProtoVersion uint
	caps                       []p2p.Cap
}

// Read reads a packet from the connection.
func (c *Conn) Read() (uint64, []byte, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	code, data, _, err := c.Conn.Read()
	if err != nil {
		return 0, nil, err
	}
	return code, data, nil
}

// ReadMsg attempts to read a devp2p message with a specific code.
func (c *Conn) ReadMsg(proto Proto, code uint64, msg any) error {
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		got, data, err := c.Read()
		if err != nil {
			return err
		}
		if protoOffset(proto)+code == got {
			return rlp.DecodeBytes(data, msg)
		}
	}
}

// Write writes a eth packet to the connection.
func (c *Conn) Write(proto Proto, code uint64, msg any) error {
	c.SetWriteDeadline(time.Now().Add(timeout))
	payload, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(protoOffset(proto)+code, payload)
	return err
}

// ReadEth reads an Eth sub-protocol wire message.
func (c *Conn) ReadEth() (any, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		code, data, _, err := c.Conn.Read()
		if err != nil {
			return nil, err
		}
		if code == pingMsg {
			c.Write(baseProto, pongMsg, []byte{})
			continue
		}
		if getProto(code) != ethProto {
			// Read until eth message.
			continue
		}
		code -= baseProtoLen

		var msg any
		switch int(code) {
		case zond.StatusMsg:
			msg = new(zond.StatusPacket)
		case zond.GetBlockHeadersMsg:
			msg = new(zond.GetBlockHeadersPacket)
		case zond.BlockHeadersMsg:
			msg = new(zond.BlockHeadersPacket)
		case zond.GetBlockBodiesMsg:
			msg = new(zond.GetBlockBodiesPacket)
		case zond.BlockBodiesMsg:
			msg = new(zond.BlockBodiesPacket)
		case zond.TransactionsMsg:
			msg = new(zond.TransactionsPacket)
		case zond.NewPooledTransactionHashesMsg:
			msg = new(zond.NewPooledTransactionHashesPacket)
		case zond.GetPooledTransactionsMsg:
			msg = new(zond.GetPooledTransactionsPacket)
		case zond.PooledTransactionsMsg:
			msg = new(zond.PooledTransactionsPacket)
		default:
			panic(fmt.Sprintf("unhandled eth msg code %d", code))
		}
		if err := rlp.DecodeBytes(data, msg); err != nil {
			return nil, fmt.Errorf("unable to decode eth msg: %v", err)
		}
		return msg, nil
	}
}

// ReadSnap reads a snap/1 response with the given id from the connection.
func (c *Conn) ReadSnap() (any, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		code, data, _, err := c.Conn.Read()
		if err != nil {
			return nil, err
		}
		if getProto(code) != snapProto {
			// Read until snap message.
			continue
		}
		code -= baseProtoLen + ethProtoLen

		var msg any
		switch int(code) {
		case snap.GetAccountRangeMsg:
			msg = new(snap.GetAccountRangePacket)
		case snap.AccountRangeMsg:
			msg = new(snap.AccountRangePacket)
		case snap.GetStorageRangesMsg:
			msg = new(snap.GetStorageRangesPacket)
		case snap.StorageRangesMsg:
			msg = new(snap.StorageRangesPacket)
		case snap.GetByteCodesMsg:
			msg = new(snap.GetByteCodesPacket)
		case snap.ByteCodesMsg:
			msg = new(snap.ByteCodesPacket)
		case snap.GetTrieNodesMsg:
			msg = new(snap.GetTrieNodesPacket)
		case snap.TrieNodesMsg:
			msg = new(snap.TrieNodesPacket)
		default:
			panic(fmt.Errorf("unhandled snap code: %d", code))
		}
		if err := rlp.DecodeBytes(data, msg); err != nil {
			return nil, fmt.Errorf("could not rlp decode message: %v", err)
		}
		return msg, nil
	}
}

// peer performs both the protocol handshake and the status message
// exchange with the node in order to peer with it.
func (c *Conn) peer(chain *Chain, status *zond.StatusPacket) error {
	if err := c.handshake(); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	if err := c.statusExchange(chain, status); err != nil {
		return fmt.Errorf("status exchange failed: %v", err)
	}
	return nil
}

// handshake performs a protocol handshake with the node.
func (c *Conn) handshake() error {
	// Write hello to client.
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &protoHandshake{
		Version: 5,
		Caps:    c.caps,
		ID:      pub0,
	}
	if err := c.Write(baseProto, handshakeMsg, ourHandshake); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}
	// Read hello from client.
	code, data, err := c.Read()
	if err != nil {
		return fmt.Errorf("erroring reading handshake: %v", err)
	}
	switch code {
	case handshakeMsg:
		msg := new(protoHandshake)
		if err := rlp.DecodeBytes(data, &msg); err != nil {
			return fmt.Errorf("error decoding handshake msg: %v", err)
		}
		// Set snappy if version is at least 5.
		if msg.Version >= 5 {
			c.SetSnappy(true)
		}
		c.negotiateEthProtocol(msg.Caps)
		if c.negotiatedProtoVersion == 0 {
			return fmt.Errorf("could not negotiate eth protocol (remote caps: %v, local eth version: %v)", msg.Caps, c.ourHighestProtoVersion)
		}
		// If we require snap, verify that it was negotiated.
		if c.ourHighestSnapProtoVersion != c.negotiatedSnapProtoVersion {
			return fmt.Errorf("could not negotiate snap protocol (remote caps: %v, local snap version: %v)", msg.Caps, c.ourHighestSnapProtoVersion)
		}
		return nil
	default:
		return fmt.Errorf("bad handshake: got msg code %d", code)
	}
}

// negotiateEthProtocol sets the Conn's eth protocol version to highest
// advertised capability from peer.
func (c *Conn) negotiateEthProtocol(caps []p2p.Cap) {
	var highestEthVersion uint
	var highestSnapVersion uint
	for _, capability := range caps {
		switch capability.Name {
		case "eth":
			if capability.Version > highestEthVersion && capability.Version <= c.ourHighestProtoVersion {
				highestEthVersion = capability.Version
			}
		case "snap":
			if capability.Version > highestSnapVersion && capability.Version <= c.ourHighestSnapProtoVersion {
				highestSnapVersion = capability.Version
			}
		}
	}
	c.negotiatedProtoVersion = highestEthVersion
	c.negotiatedSnapProtoVersion = highestSnapVersion
}

// statusExchange performs a `Status` message exchange with the given node.
func (c *Conn) statusExchange(chain *Chain, status *zond.StatusPacket) error {
loop:
	for {
		code, data, err := c.Read()
		if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}
		switch code {
		case zond.StatusMsg + protoOffset(zondProto):
			msg := new(zond.StatusPacket)
			if err := rlp.DecodeBytes(data, &msg); err != nil {
				return fmt.Errorf("error decoding status packet: %w", err)
			}
			if have, want := msg.Head, chain.blocks[chain.Len()-1].Hash(); have != want {
				return fmt.Errorf("wrong head block in status, want:  %#x (block %d) have %#x",
					want, chain.blocks[chain.Len()-1].NumberU64(), have)
			}
			if have, want := msg.TD.Cmp(chain.TD()), 0; have != want {
				return fmt.Errorf("wrong TD in status: have %v want %v", have, want)
			}
			if have, want := msg.ForkID, chain.ForkID(); !reflect.DeepEqual(have, want) {
				return fmt.Errorf("wrong fork ID in status: have %v, want %v", have, want)
			}
			if have, want := msg.ProtocolVersion, c.ourHighestProtoVersion; have != uint32(want) {
				return fmt.Errorf("wrong protocol version: have %v, want %v", have, want)
			}
			break loop
		case discMsg:
			var msg []p2p.DiscReason
			if rlp.DecodeBytes(data, &msg); len(msg) == 0 {
				return errors.New("invalid disconnect message")
			}
			return fmt.Errorf("disconnect received: %v", pretty.Sdump(msg))
		case pingMsg:
			// TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
			c.Write(baseProto, pongMsg, nil)
		default:
			return fmt.Errorf("bad status message: code %d", code)
		}
	}
	// make sure zond protocol version is set for negotiation
	if c.negotiatedProtoVersion == 0 {
		return errors.New("zond protocol version must be set in Conn")
	}
	if status == nil {
		// default status message
		status = &zond.StatusPacket{
			ProtocolVersion: uint32(c.negotiatedProtoVersion),
			NetworkID:       chain.config.ChainID.Uint64(),
			Head:            chain.blocks[chain.Len()-1].Hash(),
			Genesis:         chain.blocks[0].Hash(),
			ForkID:          chain.ForkID(),
		}
	}
	if err := c.Write(zondProto, zond.StatusMsg, status); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}
	return nil
}
*/
