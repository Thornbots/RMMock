package stream

import (
	"encoding/binary"
	"net"

	"github.com/sirupsen/logrus"
)

func GetUDPConn() *net.UDPConn {
	serverAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:3334")
	if err != nil {
	logrus.Fatalf("failed to resolve server address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
	logrus.Fatalf("failed to connect to UDP server: %v", err)
	}
	return conn
}

func PacketFactory(frameID, sliceID uint16, frameSize uint32, sliceData []byte) []byte {
	packet := make([]byte, 8+len(sliceData))

	binary.BigEndian.PutUint16(packet[0:2], frameID)
	binary.BigEndian.PutUint16(packet[2:4], sliceID)
	binary.BigEndian.PutUint32(packet[4:8], frameSize)
	copy(packet[8:], sliceData)

	return packet
}

func SendPacket(conn *net.UDPConn, encodedData []byte, frameID uint16) {
// calculate how many slices are needed
	packetSize := 1400 - 8// max recommended UDP packet size, leaving room for headers
	frameSize := len(encodedData)
	totalSlices := frameSize / packetSize
	if len(encodedData)%packetSize != 0 {
		totalSlices++
	}

// send each slice
	for sliceID := uint16(0); sliceID < uint16(totalSlices); sliceID++ {
		start := int(sliceID) * packetSize
		end := start + packetSize
		if end > len(encodedData) {
			end = len(encodedData)
		}

	// send the data packet to the server over UDP
		packet := PacketFactory(frameID, sliceID, uint32(frameSize), encodedData[start:end])
		_, err := conn.Write(packet)
		if err != nil {
		logrus.Errorf("failed to send slice: %v", err)
			continue
		}
	logrus.Debugf("sent slice %d", sliceID)
	}
}
