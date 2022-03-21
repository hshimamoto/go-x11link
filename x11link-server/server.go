// x11link-server
// vim: set sw=4 sts=4:
// MIT License Copyright(c) 2022 Hiroshi Shimamoto
package main
import (
    "encoding/binary"
    "fmt"
    "net"
    "os"
    "os/exec"
    "strings"

    "github.com/sirupsen/logrus"
    "github.com/hshimamoto/go-session"

    "github.com/hshimamoto/go-x11link"
)

func LaunchXclient(cmdline []string, display string) {
    cmd := exec.Command(cmdline[0], cmdline[1:]...)
    cmd.Env = append(os.Environ(), "DISPLAY=" + display)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    go cmd.Run()
}

func handle_X11(xlinks *x11link.X11LinkManager, buf []byte) {
    xid := binary.LittleEndian.Uint32(buf[0:4])
    xlink := xlinks.GetX11Link(xid)
    if xlink == nil {
	return
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

func handler(conn net.Conn) {
    defer conn.Close()

    // check /tmp/.X11-unix
    files, err := os.ReadDir("/tmp/.X11-unix")
    if err != nil {
	logrus.Errorf("ReadDir: %v", err)
	return
    }
    disp := 20
    xsock := fmt.Sprintf("X%d", disp)
    for {
	ok := true
	for _, f := range files {
	    if f.Name() == xsock {
		ok = false
		break
	    }
	}
	if ok {
	    break
	}
	disp++
	xsock = fmt.Sprintf("X%d", disp)
    }
    logrus.Infof("Use %s", xsock)

    // X11Links
    xlinks := x11link.NewX11LinkManager(conn)
    defer xlinks.CloseAll()

    // setenv DISPLAY
    // create UNIX socket server
    xsock = fmt.Sprintf("/tmp/.X11-unix/X%d", disp)
    serv, err := session.NewServer(xsock, func (raw net.Conn) {
	defer raw.Close()   // no need?

	// new X client is connected
	xlink, xid := xlinks.NewX11Link(raw)
	logrus.Infof("New X client xid=%d", xid)
	xlink.Transfer()
	xlink.Close()
	logrus.Infof("Close X client xid=%d", xid)
	xlinks.DeleteLink(xlink)
    })
    if err != nil {
	logrus.Errorf("NewServer: %v", err)
	return
    }
    go func() {
	serv.Run()
	logrus.Infof("Remove %s", xsock)
	os.Remove(xsock)
    }()
    defer serv.Stop()

    env_display := fmt.Sprintf(":%d.0", disp)

    err = xlinks.DispatchLoop(func(t int, buf []byte) {
	switch t {
	case 1:
	    msg := strings.Split(string(buf), "\x00")
	    if msg[0] == "EXEC" {
		// start client
		LaunchXclient(msg[1:], env_display)
	    }
	case 2: handle_X11(xlinks, buf)
	}
    })
    logrus.Infof("DispatchLoop: %v", err)
}

func main() {
    if os.Getenv("DEBUG") == "true" {
	x11link.Debug = true
    }
    // start server
    serv, err := session.NewServer(":6020", handler)
    if err != nil {
	logrus.Errorf("NewServer: %v", err)
	return
    }
    serv.Run()
}
