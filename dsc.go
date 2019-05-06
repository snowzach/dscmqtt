package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	config "github.com/spf13/viper"
	"github.com/tarm/serial"
)

const (
	DSC_TYPE_UNKNOWN = "unknown"
	DSC_TYPE_ZONE    = "zone"

	DSC_STATE_OPEN   = "open"
	DSC_STATE_CLOSED = "closed"

	DSC_CMD_NOOP        = "000"
	DSC_CMD_FULL_STATUS = "001"
	DSC_CMD_TIME_UPDATE = "010"
)

type DSCPanel struct {
	port      io.ReadWriteCloser
	ack       chan error
	msgBuf    chan *DSCMessage
	sendMutex sync.Mutex
}

type DSCMessage struct {
	Type  string
	State string
	Id    string
	Err   error
}

func NewDSCPanel() (*DSCPanel, error) {

	serialPort, err := serial.OpenPort(&serial.Config{
		Name: config.GetString("dsc.port"),
		Baud: config.GetInt("dsc.baud"),
	})
	if err != nil {
		return nil, fmt.Errorf("Could not open serial port: %v", err)
	}
	p := &DSCPanel{
		port:   serialPort,
		ack:    make(chan error),
		msgBuf: make(chan *DSCMessage),
	}

	// Read commands as they come in
	go func() {
		var buf *bufio.Reader = bufio.NewReader(serialPort)
		for {
			var line bytes.Buffer
			for {
				b, err := buf.ReadByte()
				if err != nil {
					p.msgBuf <- &DSCMessage{
						Err: err,
					}
					return
				}
				if b == '\r' {
					continue // Ignore it
				} else if b == '\n' {
					if line.Len() == 0 {
						continue // Nothing there
					}
					break // done
				}
				line.WriteByte(b)
			}

			// Validate the checksum
			s := line.String()
			if len(s) >= 5 {
				if checksum(s[:len(s)-2]) != s[len(s)-2:] {
					p.msgBuf <- &DSCMessage{
						Err: fmt.Errorf("Bad Checksum: %s", s),
					}
					continue
				}
			} else {
				p.msgBuf <- &DSCMessage{
					Err: fmt.Errorf("Bad Response: %s", s),
				}
				continue
			}

			// Remove the checksum
			s = s[:len(s)-2]

			// Check Status
			switch s[:3] {
			// Command Ack
			case "500":
				p.ack <- nil

			// Bad Checksum
			case "501":
				p.ack <- fmt.Errorf("Bad Checksum")

			// Zone Open
			case "609":
				if len(s) >= 6 {
					p.msgBuf <- &DSCMessage{
						Type:  DSC_TYPE_ZONE,
						State: DSC_STATE_OPEN,
						Id:    strings.TrimLeft(s[3:6], "0"),
					}
					continue
				}
				p.msgBuf <- &DSCMessage{
					Err: fmt.Errorf("Invalid Zone: %s", s),
				}

			// Zone Closed
			case "610":
				if len(s) >= 6 {
					p.msgBuf <- &DSCMessage{
						Type:  DSC_TYPE_ZONE,
						State: DSC_STATE_CLOSED,
						Id:    strings.TrimLeft(s[3:6], "0"),
					}
					continue
				}
				p.msgBuf <- &DSCMessage{
					Err: fmt.Errorf("Invalid Zone: %s", s),
				}

			default:
				p.msgBuf <- &DSCMessage{
					Type: DSC_TYPE_UNKNOWN,
					Id:   s[:3],
				}
			}
		}
	}()

	// Validate the connection
	err = p.SendCmd(DSC_CMD_NOOP)
	if err != nil {
		return nil, fmt.Errorf("Could not send check command: %v", err)
	}

	return p, nil

}

// Request a full status update
func (p *DSCPanel) FullUpdate() error {
	return p.SendCmd(DSC_CMD_FULL_STATUS)
}

// Update the time
func (p *DSCPanel) TimeUpdate() error {
	return p.SendCmd(DSC_CMD_TIME_UPDATE+time.Now().Format("1504010206"))
}

// Send a command
func (p *DSCPanel) SendCmd(cmd string) error {

	p.sendMutex.Lock()
        defer p.sendMutex.Unlock()

	// Send it to the port
	_, err := fmt.Fprintf(p.port, "%s%s\r\n", cmd, checksum(cmd))

	select {
	case err := <-p.ack:
		if err != nil {
			return err
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Timout on SendCmd")
	}

	return err

}

func (p *DSCPanel) GetMessage(block bool) *DSCMessage {

	if block {
		m := <-p.msgBuf
		return m
	} else {
		select {
		case m := <-p.msgBuf:
			return m
		default:
		}
	}

	return nil

}

func (p *DSCPanel) GetMessageChan() <-chan *DSCMessage {
	return p.msgBuf
}

func checksum(cmd string) string {
	// Calc checksum
	var checksum uint16
	for _, c := range []byte(cmd) {
		checksum += uint16(c)
	}
	return fmt.Sprintf("%02X", checksum&0xFF)
}
