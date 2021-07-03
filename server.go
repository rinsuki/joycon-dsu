package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

var server *net.UDPConn

func listenServer() {
	server_, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 26760,
	})
	server = server_
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 1024)
	writer := new(bytes.Buffer)
	for {
		// read request
		n, addr, err := server.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}
		if n < 16 { // if too small, reject
			continue
		}
		// read headers
		b := bytes.NewReader(buf[0:n])
		var header DSUHeader
		panicIfError(binary.Read(b, binary.LittleEndian, &header))
		if header.MagicBytes != 0x43555344 { // magic bytes are not "DSUC", just ignore
			continue
		}

		fmt.Println("received", header)
		// handle requests
		switch header.EventType {
		case 0x100001:
			binary.Write(writer, binary.LittleEndian, header.EventType)
			// information about connected controllers
			var portsCount uint32
			panicIfError(binary.Read(b, binary.LittleEndian, &portsCount))
			if portsCount > 4 {
				log.Fatalln("portsCount > 4")
			}
			ports := make([]byte, portsCount)
			panicIfError(binary.Read(b, binary.LittleEndian, ports))

			for _, i := range ports {
				writer.Reset()
				var info ControllerInfo
				if i == 0 {
					info = ControllerInfo{
						SlotID:         i,
						SlotState:      2,
						DeviceModel:    2,
						ConnectionType: 1,
						MACAddress:     [6]byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xAB},
						BatteryStatus:  5,
					}
				} else {
					info = ControllerInfo{
						SlotID: i,
					}
				}
				panicIfError(binary.Write(writer, binary.LittleEndian, info))
				writer.Write([]byte{0})
				sendPacket(server, addr, 0x100001, writer.Bytes())
			}
		case 0x100002:
			var req ControllerDataRequest
			panicIfError(binary.Read(b, binary.LittleEndian, &req))
			fmt.Println(req)
			sendTo = addr
			sendStart = time.Now()
		default:
			fmt.Println("Unknown Event Type", header.EventType)
		}
		_ = addr
	}
}
