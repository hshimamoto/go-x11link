// x11link
// vim: set sw=4 sts=4:
// MIT License Copyright(c) 2022 Hiroshi Shimamoto
package x11link

import (
    "encoding/binary"
    "fmt"
    "net"

    "github.com/sirupsen/logrus"
)

var Debug bool = false
const BufferSize = 65536

// message
// "S|len|......." shell data
// "X|len|xid|..." X11 data
func ReadMessage(conn net.Conn, buf []byte) (int, int, error) {
    head := make([]byte, 5)
    i := 0
    for i < 5 {
	n, err := conn.Read(head[i:])
	if n <= 0 {
	    return 0, 0, err
	}
	i += n
    }
    t := 0
    switch head[0] {
    case 'S': t = 1
    case 'X': t = 2
    default:
	return 0, 0, fmt.Errorf("unknown type 0x%02x", head[0])
    }
    l := int(binary.LittleEndian.Uint32(head[1:5]))
    if Debug {
	logrus.Infof("ReadMessage: type=%d len=%d", t, l)
    }
    // now read it
    if l > BufferSize {
	return 0, 0, fmt.Errorf("bad length")
    }
    // read body
    i = 0
    for i < l {
	n, err := conn.Read(buf[i:l])
	if n <= 0 {
	    return 0, 0, err
	}
	i += n
    }
    return t, l, nil
}

func SendMessage(conn net.Conn, t int, buf []byte) error {
    head := make([]byte, 5)
    switch t {
    case 1: head[0] = 'S'
    case 2: head[0] = 'X'
    default:
	return fmt.Errorf("unknown type")
    }
    l := len(buf)
    if l > BufferSize {
	return fmt.Errorf("bad length")
    }
    binary.LittleEndian.PutUint32(head[1:], uint32(l))
    i := 0
    for i < 5 {
	n, err := conn.Write(head[i:])
	if n <= 0 {
	    return err
	}
	i += n
    }
    i = 0
    for i < l {
	n, err := conn.Write(buf[i:])
	if n <= 0 {
	    return err
	}
	i += n
    }
    if Debug {
	logrus.Infof("SendMessage: type=%d len=%d", t, l)
    }
    return nil
}
