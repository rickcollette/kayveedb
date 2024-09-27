package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// CommandType represents the type of command.
type CommandType byte

const (
	CommandAuth   CommandType = 0x01
	CommandInsert CommandType = 0x02
	CommandUpdate CommandType = 0x03
	CommandDelete CommandType = 0x04
	CommandRead   CommandType = 0x05
)

// StatusCode represents the status of the response.
type StatusCode uint32

const (
	StatusSuccess StatusCode = 0x00
	StatusError   StatusCode = 0x01
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
	const maxPayloadSize = 10 * 1024 * 1024 // 10 MB
	if dataSize > maxPayloadSize {
		return r, fmt.Errorf("DeserializeResponse: data size %d exceeds maximum allowed %d", dataSize, maxPayloadSize)
	}

	// Read Data
	dataBytes := make([]byte, dataSize)
	if _, err := io.ReadFull(reader, dataBytes); err != nil {
		return r, fmt.Errorf("DeserializeResponse: failed to read Data: %w", err)
	}
	r.Data = string(dataBytes)

	return r, nil
}
