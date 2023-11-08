package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-------+-+-------------+-------------------------------+
// |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
// |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
// |N|V|V|V|       |S|             |   (if payload len==126/127)   |
// | |1|2|3|       |K|             |                               |
// +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
// |     Extended payload length continued, if payload len == 127  |
// + - - - - - - - - - - - - - - - +-------------------------------+
// |                               |Masking-key, if MASK set to 1  |
// +-------------------------------+-------------------------------+
// | Masking-key (continued)       |          Payload Data         |
// +-------------------------------- - - - - - - - - - - - - - - - +
// :                     Payload Data continued ...                :
// + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
// |                     Payload Data continued ...                |
// +---------------------------------------------------------------+

var mask []byte

func ReadMessage(rwbuf *bufio.ReadWriter) ([]byte, error) {
	var message []byte
	// var header []byte

	// b, _ := rwbuf.ReadByte()
	// header = append(header, b)

	header := make([]byte, 2, 12)

	_, err := rwbuf.Read(header)
	if err != nil {
		return nil, err
	}

	fin := header[0] >> 7
	opCode := header[0] & 0xf

	maskBit := header[1] >> 7

	extra := 0
	if maskBit == 1 {
		extra += 4
	}

	size := uint64(header[1] & 0x7f)
	if size == 126 {
		extra += 2
	} else if size == 127 {
		extra += 8
	}

	if extra > 0 {
		header = header[:extra]
		_, err = rwbuf.Read(header)
		if err != nil {
			return nil, err
		}

		if size == 126 {
			size = uint64(binary.BigEndian.Uint16(header[:2]))
			header = header[2:]
		} else if size == 127 {
			size = uint64(binary.BigEndian.Uint64(header[:8]))
			header = header[8:]
		}
	}

	if maskBit == 1 {
		mask = header
	}

	payload := make([]byte, int(size))
	_, err = io.ReadFull(rwbuf, payload)
	if err != nil {
		return nil, err
	}

	if maskBit == 1 {
		for i := 0; i < len(payload); i++ {
			payload[i] ^= mask[i%4]
		}
	}

	message = append(message, payload...)

	if opCode == 8 {
		return nil, err
	} else if fin == 1 {
		fmt.Println(string(message))
		// message = message[:0]
	}
	return message, nil
}

func WriteMessage(writer io.Writer, opcode byte, message []byte) (int, error) {
	header := make([]byte, 10) // WebSocket frame header

	finBit := byte(0x80)
	rsvBits := byte(0)
	maskBit := byte(0)
	payloadLen := len(message)

	header[0] = finBit | rsvBits | opcode

	// Determine the payload length and adjust the header
	if payloadLen <= 125 {
		header[1] = maskBit | byte(payloadLen)
	} else if payloadLen < 65536 {
		header[1] = maskBit | 126
		binary.BigEndian.PutUint16(header[2:4], uint16(payloadLen))
	} else {
		header[1] = maskBit | 127
		binary.BigEndian.PutUint64(header[2:10], uint64(payloadLen))
	}

	writer.Write(header) // Write the header to the writer
	maskingKey := mask
	if maskingKey != nil {
		writer.Write(maskingKey) // Write the masking key

		// Mask the data and write it
		maskedData := make([]byte, payloadLen)
		for i := 0; i < payloadLen; i++ {
			maskedData[i] = message[i] ^ maskingKey[i%4]
		}
		fmt.Println(string(maskedData))

		writer.Write(maskedData)
	} else {
		fmt.Println(string(message))

		writer.Write(message) // Write the data as is
	}

	return payloadLen, nil
}
