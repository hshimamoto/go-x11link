// x11link
// vim: set sw=4 sts=4:
// MIT License Copyright(c) 2022 Hiroshi Shimamoto
package x11link

import (
    "encoding/binary"
    "fmt"
    "net"
    "sync"
)

type X11Link struct {
    xlink net.Conn
    raw net.Conn
    xid uint32
    m *sync.Mutex
}

func NewX11Link(xlink, raw net.Conn, xid uint32, m *sync.Mutex) *X11Link {
    x := &X11Link{
	xlink: xlink,
	raw: raw,
	xid: xid,
	m: m,
    }
    return x
}

func (x *X11Link)Transfer() {
    buf := make([]byte, BufferSize)
    binary.LittleEndian.PutUint32(buf[0:], x.xid)
    for {
	n, err := x.raw.Read(buf[4:])
	if n <= 0 {
	    x.m.Lock()
	    SendMessage(x.xlink, 2, buf[:4])
	    x.m.Unlock()
	    break
	}
	// serialize between xlinks
	x.m.Lock()
	err = SendMessage(x.xlink, 2, buf[:n+4])
	x.m.Unlock()
	if err != nil {
	    break
	}
    }
}

func (x *X11Link)Receive(buf []byte) error {
    l := len(buf)
    i := 0
    for i < l {
	n, err := x.raw.Write(buf[i:l])
	if n <= 0 {
	    if err != nil {
		return err
	    }
	    return fmt.Errorf("Write aborted")
	}
	i += n
    }
    return nil
}

func (x *X11Link)Close() {
    x.raw.Close()
}

type X11LinkManager struct {
    xlinks map[uint32]*X11Link
    m_xlinks sync.Mutex
    conn net.Conn
    xid uint32
    m sync.Mutex
}

func NewX11LinkManager(conn net.Conn) *X11LinkManager {
    x := &X11LinkManager{
	conn: conn,
    }
    x.xlinks = make(map[uint32]*X11Link)
    return x
}

func (x *X11LinkManager)NewX11Link(raw net.Conn) (*X11Link, uint32) {
    x.m.Lock()
    defer x.m.Unlock()
    xid := x.xid
    xlink := NewX11Link(x.conn, raw, xid, &x.m_xlinks)
    x.xlinks[xid] = xlink
    x.xid++
    return xlink, xid
}

func (x *X11LinkManager)NewX11LinkWithId(raw net.Conn, xid uint32) *X11Link {
    x.m.Lock()
    defer x.m.Unlock()
    xlink := NewX11Link(x.conn, raw, xid, &x.m_xlinks)
    x.xlinks[xid] = xlink
    return xlink
}

func (x *X11LinkManager)GetX11Link(xid uint32) *X11Link {
    x.m.Lock()
    defer x.m.Unlock()
    if x.xlinks != nil {
	xlink, ok := x.xlinks[xid]
	if ok {
	    return xlink
	}
    }
    return nil
}

func (x *X11LinkManager)DeleteLink(xlink *X11Link) {
    x.m.Lock()
    defer x.m.Unlock()
    if x.xlinks != nil {
	x.xlinks[xlink.xid] = nil
	delete(x.xlinks, xlink.xid)
    }
}

func (x *X11LinkManager)CloseAll() {
    x.m.Lock()
    defer x.m.Unlock()
    for xid, xlink := range x.xlinks {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf[0:], xid)
	x.m_xlinks.Lock()
	SendMessage(xlink.xlink, 2, buf)
	x.m_xlinks.Unlock()
	xlink.Close()
    }
    x.xlinks = nil
}

func (x *X11LinkManager)DispatchLoop(cb func(int, []byte)) error {
    buf := make([]byte, BufferSize)
    for {
	t, n, err := ReadMessage(x.conn, buf)
	if t == 0 {
	    if err != nil {
		return err
	    }
	    return fmt.Errorf("Close")
	}
	// callback
	cb(t, buf[:n])
    }
}
