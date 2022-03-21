// x11link-client
// vim: set sw=4 sts=4:
// MIT License Copyright(c) 2022 Hiroshi Shimamoto
package main
import (
    "encoding/binary"
    "fmt"
    "os"
    "strings"

    "github.com/sirupsen/logrus"
    "github.com/hshimamoto/go-session"

    "github.com/hshimamoto/go-x11link"
)

func handle_X11(xlinks *x11link.X11LinkManager, buf []byte) {
    xid := binary.LittleEndian.Uint32(buf[0:4])
    xlink := xlinks.GetX11Link(xid)
    if xlink == nil {
	// New X client comes

	// get DISPLAY
	display := os.Getenv("DISPLAY")
	logrus.Infof("local DISPLAY=%s", display)
	xdisp := strings.Split(display, ":")
	if xdisp[0] == "" {
	    // unix
	    ndisp := strings.Split(xdisp[1], ".")
	    display = fmt.Sprintf("/tmp/.X11-unix/X%s", ndisp[0])
	}

	raw, err := session.Dial(display)
	if err != nil {
	    logrus.Infof("X id=%d Dial: %v", xid, err)
	    return
	}
	logrus.Infof("New X id=%d", xid)
	xlink = xlinks.NewX11LinkWithId(raw, xid)
	// start Transfer
	go func(x *x11link.X11Link, id uint32) {
	    x.Transfer()
	    logrus.Infof("Done X id=%d", id)
	}(xlink, xid)
    }
    var err error = nil
    if len(buf) > 4 {
	err = xlink.Receive(buf[4:])
    } else {
	err = fmt.Errorf("remote close")
    }
    if err != nil {
	// close
	xlink.Close()
	xlinks.DeleteLink(xlink)
    }
}

func main() {
    if os.Getenv("DEBUG") == "true" {
	x11link.Debug = true
    }

    if len(os.Args) == 1 {
	return
    }
    // connect to server
    conn, err := session.Dial(os.Args[1])
    if err != nil {
	logrus.Errorf("Dial: %v", err)
	return
    }
    defer conn.Close()

    cmdline := os.Args[2:]
    if len(cmdline) == 0 {
	cmdline = []string{"xterm"}
    }
    logrus.Infof("cmdline: %v", cmdline)

    // Send cmdline
    cmdline = append([]string{"EXEC"}, cmdline...)
    msg := strings.Join(cmdline, "\x00")
    x11link.SendMessage(conn, 1, []byte(msg))

    xlinks := x11link.NewX11LinkManager(conn)
    defer xlinks.CloseAll()

    err = xlinks.DispatchLoop(func(t int, buf []byte) {
	switch t {
	case 2: handle_X11(xlinks, buf)
	}
    })
    logrus.Infof("DispatchLoop: %v", err)
}
