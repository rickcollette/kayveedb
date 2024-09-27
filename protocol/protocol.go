package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/rickcollette/kayveedb/lib"
)

// CommandType represents the type of command.
type CommandType byte

const (
	CommandAuth        CommandType = 0x01
	CommandInsert      CommandType = 0x02
	CommandUpdate      CommandType = 0x03
	CommandDelete      CommandType = 0x04
	CommandRead        CommandType = 0x05
	CommandBeginTx     CommandType = 0x06
	CommandCommitTx    CommandType = 0x07
	CommandRollbackTx  CommandType = 0x08
	CommandSetCache    CommandType = 0x09
	CommandGetCache    CommandType = 0x0A
	CommandDeleteCache CommandType = 0x0B
	CommandFlushCache  CommandType = 0x0C
	CommandPublish     CommandType = 0x0D
	CommandSubscribe   CommandType = 0x0E
	CommandConnect     CommandType = 0x0F
	CommandDisconnect  CommandType = 0x10
	// New Command Types for Advanced Features
	CommandListPush    CommandType = 0x11
	CommandListRange   CommandType = 0x12
	CommandSetAdd      CommandType = 0x13
	CommandSetMembers  CommandType = 0x14
	CommandHashSet     CommandType = 0x15
	CommandHashGet     CommandType = 0x16
	CommandZSetAdd     CommandType = 0x17
	CommandZSetRange   CommandType = 0x18
)

type StatusCode uint32
type Subscriber struct {
	// Define fields for Subscriber
	ID       string
	Channel  string
	Messages chan string
}
const (
	StatusSuccess       StatusCode = 0x00
	StatusError         StatusCode = 0x01
	StatusTxBegin       StatusCode = 0x02
	StatusTxCommit      StatusCode = 0x03
	StatusTxRollback    StatusCode = 0x04
	StatusClientAdded   StatusCode = 0x05
	StatusClientRemoved StatusCode = 0x06
)

// Packet represents a protocol packet.
type Packet struct {
	CommandID   uint32
	CommandType CommandType
	Payload     []byte
}

// Response represents a response packet.
type Response struct {
	CommandID uint32
	Status    StatusCode
	Data      string
}

// Global BTree instance
var (
	bTreeInstance  *lib.BTree
	maxPayloadSize uint32 = 10 * 1024 * 1024 // Default 10 MB
	mu             sync.RWMutex         // Mutex to protect maxPayloadSize
)

// Initialize BTree
func InitBTree(t int, dbPath, dbName, logName string, hmacKey, encryptionKey, nonce []byte, cacheSize int) error {
	var err error
	bTreeInstance, err = lib.NewBTree(t, dbPath, dbName, logName, hmacKey, encryptionKey, nonce, cacheSize)
	if err != nil {
		return fmt.Errorf("InitBTree failed: %w", err)
	}
	return nil
}

// HandleClientConnect adds a client to the BTree's client list.
func HandleClientConnect(clientID uint32) error {
	if bTreeInstance == nil {
		return fmt.Errorf("BTree instance not initialized")
	}
	return bTreeInstance.AddClient(clientID)
}

// HandleClientDisconnect removes a client from the BTree's client list.
func HandleClientDisconnect(clientID uint32) error {
	if bTreeInstance == nil {
		return fmt.Errorf("BTree instance not initialized")
	}
	return bTreeInstance.RemoveClient(clientID)
}

// SetMaxPayloadSize sets a new maximum payload size.
func SetMaxPayloadSize(size uint32) {
	mu.Lock()
	defer mu.Unlock()
	maxPayloadSize = size
}

// GetMaxPayloadSize retrieves the current maximum payload size.
func GetMaxPayloadSize() uint32 {
	mu.RLock()
	defer mu.RUnlock()
	return maxPayloadSize
}

// SerializePacket serializes a Packet into bytes.
func SerializePacket(p Packet) ([]byte, error) {
	payloadSize := uint32(len(p.Payload))
	buf := new(bytes.Buffer)

	// Write CommandID
	if err := binary.Write(buf, binary.BigEndian, p.CommandID); err != nil {
		return nil, fmt.Errorf("SerializePacket: failed to write CommandID: %w", err)
	}

	// Write CommandType
	if err := binary.Write(buf, binary.BigEndian, p.CommandType); err != nil {
		return nil, fmt.Errorf("SerializePacket: failed to write CommandType: %w", err)
	}

	// Write PayloadSize
	if err := binary.Write(buf, binary.BigEndian, payloadSize); err != nil {
		return nil, fmt.Errorf("SerializePacket: failed to write PayloadSize: %w", err)
	}

	// Write Payload
	if _, err := buf.Write(p.Payload); err != nil {
		return nil, fmt.Errorf("SerializePacket: failed to write Payload: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeResponse deserializes bytes into a Response.
func DeserializeResponse(reader io.Reader) (Response, error) {
	var r Response

	// Read CommandID
	if err := binary.Read(reader, binary.BigEndian, &r.CommandID); err != nil {
		return r, fmt.Errorf("DeserializeResponse: failed to read CommandID: %w", err)
	}

	// Read Status
	var status uint32
	if err := binary.Read(reader, binary.BigEndian, &status); err != nil {
		return r, fmt.Errorf("DeserializeResponse: failed to read Status: %w", err)
	}
	r.Status = StatusCode(status)

	// Read DataSize
	var dataSize uint32
	if err := binary.Read(reader, binary.BigEndian, &dataSize); err != nil {
		return r, fmt.Errorf("DeserializeResponse: failed to read DataSize: %w", err)
	}

	// Validate data size to prevent potential buffer overflows or DoS attacks
	if dataSize > GetMaxPayloadSize() {
		return r, fmt.Errorf("DeserializeResponse: data size %d exceeds maximum allowed %d", dataSize, GetMaxPayloadSize())
	}

	// Read Data
	dataBytes := make([]byte, dataSize)
	if _, err := io.ReadFull(reader, dataBytes); err != nil {
		return r, fmt.Errorf("DeserializeResponse: failed to read Data: %w", err)
	}
	r.Data = string(dataBytes)

	return r, nil
}

// Command Types Mapping for Debugging/Logging
func (c CommandType) String() string {
	switch c {
	case CommandAuth:
		return "Auth"
	case CommandInsert:
		return "Insert"
	case CommandUpdate:
		return "Update"
	case CommandDelete:
		return "Delete"
	case CommandRead:
		return "Read"
	case CommandBeginTx:
		return "Begin Transaction"
	case CommandCommitTx:
		return "Commit Transaction"
	case CommandRollbackTx:
		return "Rollback Transaction"
	case CommandSetCache:
		return "Set Cache"
	case CommandGetCache:
		return "Get Cache"
	case CommandDeleteCache:
		return "Delete Cache"
	case CommandFlushCache:
		return "Flush Cache"
	case CommandPublish:
		return "Publish"
	case CommandSubscribe:
		return "Subscribe"
	case CommandConnect:
		return "Connect"
	case CommandDisconnect:
		return "Disconnect"
	case CommandListPush:
		return "List Push"
	case CommandListRange:
		return "List Range"
	case CommandSetAdd:
		return "Set Add"
	case CommandSetMembers:
		return "Set Members"
	case CommandHashSet:
		return "Hash Set"
	case CommandHashGet:
		return "Hash Get"
	case CommandZSetAdd:
		return "ZSet Add"
	case CommandZSetRange:
		return "ZSet Range"
	default:
		return "Unknown"
	}
}

// StatusCode Mapping for Debugging/Logging
func (s StatusCode) String() string {
	switch s {
	case StatusSuccess:
		return "Success"
	case StatusError:
		return "Error"
	case StatusTxBegin:
		return "Transaction Begin"
	case StatusTxCommit:
		return "Transaction Commit"
	case StatusTxRollback:
		return "Transaction Rollback"
	case StatusClientAdded:
		return "Client Added"
	case StatusClientRemoved:
		return "Client Removed"
	default:
		return "Unknown"
	}
}
