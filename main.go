package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/nobonobo/joycon"
)

var sendTo *net.UDPAddr = nil
var sendStart time.Time = time.Time{}

func main() {
	devices, err := joycon.Search(joycon.JoyConR)
	if err != nil {
		panic(err)
	}
	jc, err := joycon.NewJoycon(devices[0].Path, false)
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println("JoyCon closing...")
		jc.Close()
	}()
	fmt.Println(devices[0], jc)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	cstate := jc.State()
	csensor := jc.Sensor()

	go listenServer()
	cd := ControllerData{
		ControllerInfo: ControllerInfo{
			SlotID:         0,
			SlotState:      2,
			DeviceModel:    2,
			ConnectionType: 1,
			MACAddress:     [6]byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xAB},
			BatteryStatus:  5,
		},
		IsConnected:     1,
		PacketNumber:    1,
		DigitalButtons1: 0x00,
		DigitalButtons2: 0x00,
		Stick_L_X:       128,
		Stick_L_Y:       128,
		Stick_R_X:       128,
		Stick_R_Y:       128,
	}
	writer := new(bytes.Buffer)
	started := time.Now()
	for {
		select {
		case <-c:
			close(c)
			return
		case state := <-cstate:
			highButtons := state.Buttons >> 8
			cd.Button_A = digitalToAnalog(bitCheck(state.Buttons, 3))
			cd.Button_B = digitalToAnalog(bitCheck(state.Buttons, 2))
			cd.Button_X = digitalToAnalog(bitCheck(state.Buttons, 1))
			cd.Button_Y = digitalToAnalog(bitCheck(state.Buttons, 0))
			cd.Button_L1 = digitalToAnalog(bitCheck(state.Buttons, 4))
			cd.Button_L2 = digitalToAnalog(bitCheck(state.Buttons, 5))
			cd.Button_R1 = digitalToAnalog(bitCheck(state.Buttons, 6))
			cd.Button_R2 = digitalToAnalog(bitCheck(state.Buttons, 7))
			cd.Button_PS = digitalToAnalog(bitCheck(highButtons, 4))
			cd.DigitalButtons1 = uint8((highButtons&0b10)<<2) | uint8((highButtons & 0b100))
			cd.Stick_R_X = uint8(clamp((state.RightAdj.X+1)/2) * 255)
			cd.Stick_R_Y = uint8(clamp((-state.RightAdj.Y+1)/2) * 255)
			fmt.Printf("%d %d\t", cd.Stick_R_X, cd.Stick_R_Y)
			fmt.Printf("%b\t%b\n", state.Buttons>>8, state.Buttons&0xFF)
		case sensor := <-csensor:
			addr := sendTo
			conn := server
			if addr != nil && conn != nil && time.Since(sendStart).Seconds() < 15 {
				t := time.Since(started)
				writer.Reset()
				cd.MotionDataTimestamp = uint64(t.Microseconds())
				// why 720???
				const ToDeg = 720.0 / math.Pi
				cd.AccelX = -sensor.Accel.Y
				cd.AccelY = sensor.Accel.Z
				cd.AccelZ = sensor.Accel.X
				cd.GyroRoll = sensor.Gyro.X * ToDeg
				cd.GyroPitch = sensor.Gyro.Y * ToDeg
				cd.GyroYaw = sensor.Gyro.Z * ToDeg
				panicIfError(binary.Write(writer, binary.LittleEndian, cd))
				cd.PacketNumber += 1
				sendPacket(conn, addr, 0x100002, writer.Bytes())
			}
			// fmt.Println(s)
		}
	}
}

func bitCheck(input uint32, bit uint8) bool {
	if (input & (1 << bit)) != 0 {
		return true
	} else {
		return false
	}
}

func digitalToAnalog(d bool) uint8 {
	if d {
		return 255
	} else {
		return 0
	}
}

func clamp(input float32) float32 {
	if input < 0 {
		return 0
	} else if input > 1 {
		return 1
	} else {
		return input
	}
}
