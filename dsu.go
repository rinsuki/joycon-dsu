package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"math/rand"
	"net"
	"time"
)

type DSUHeader struct {
	MagicBytes      uint32
	ProtocolVersion uint16
	PacketLength    uint16
	CRC32           uint32
	ClientID        uint32
	EventType       uint32
}

type ControllerInfo struct {
	SlotID         uint8
	SlotState      uint8
	DeviceModel    uint8
	ConnectionType uint8
	MACAddress     [6]byte
	BatteryStatus  uint8
}

type ControllerDataRequest struct {
	ControllerRequestType uint8
	Slot                  uint8
	MACAddress            [6]byte
}

type ControllerData struct {
	ControllerInfo      ControllerInfo
	IsConnected         uint8
	PacketNumber        uint32
	DigitalButtons1     uint8
	DigitalButtons2     uint8
	Button_PS           uint8
	Button_Touch        uint8
	Stick_L_X           uint8
	Stick_L_Y           uint8
	Stick_R_X           uint8
	Stick_R_Y           uint8
	Button_Left         uint8
	Button_Down         uint8
	Button_Right        uint8
	Button_Up           uint8
	Button_Y            uint8
	Button_B            uint8
	Button_A            uint8
	Button_X            uint8
	Button_R1           uint8
	Button_L1           uint8
	Button_R2           uint8
	Button_L2           uint8
	Touch1              ControllerTouchData
	Touch2              ControllerTouchData
	MotionDataTimestamp uint64
	AccelX              float32
	AccelY              float32
	AccelZ              float32
	GyroPitch           float32
	GyroYaw             float32
	GyroRoll            float32
}

type ControllerTouchData struct {
	Active uint8
	ID     uint8
	X      uint16
	Y      uint16
}

var serverID uint32 = rand.New(rand.NewSource(time.Now().Unix())).Uint32()

func sendPacket(conn *net.UDPConn, addr *net.UDPAddr, eventType uint32, content []byte) error {
	// fmt.Println("SEND", eventType, content)
	header := DSUHeader{
		MagicBytes:      0x53555344,
		ProtocolVersion: 1001,
		PacketLength:    uint16(len(content) + 4),
		CRC32:           0, // calc later
		ClientID:        serverID,
		EventType:       eventType,
	}

	writer := new(bytes.Buffer)
	if err := binary.Write(writer, binary.LittleEndian, header); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, content); err != nil {
		return err
	}

	payload := writer.Bytes()
	header.CRC32 = crc32.ChecksumIEEE(payload)

	writer.Reset()
	// TODO: simply overrides only CRC32 field
	if err := binary.Write(writer, binary.LittleEndian, header); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, content); err != nil {
		return err
	}

	// fmt.Println(addr, writer.Bytes(), len(content))
	_, err := conn.WriteToUDP(writer.Bytes(), addr)
	return err
}
