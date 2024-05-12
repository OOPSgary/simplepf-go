package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

func main() {
	var (
		services serviceList
	)
	flag.Var(&services, "L", ":ListenPort/IP:Port;IP:Port")
	nodelay := flag.Bool("nodelay", true, "nodelay")
	flag.Parse()
	if len(services) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if err := services.Run(*nodelay); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

/*
	if err != nil {
			log.Println("fail to parse flag: %v", err)
			flag.PrintDefaults()
		}
*/
type serviceList []*service

func (l *serviceList) String() string {
	s := ""
	for i := range *l {
		s = fmt.Sprintf("%s\nl:%d,targets:%s", s, (*l)[i].ListenPort, (*l)[i].TargetAddress)
	}
	return s
}
func (l *serviceList) Set(value string) error {
	lt := strings.SplitN(value, "/", 2)
	if len(lt) != 2 {
		return fmt.Errorf("fail to parse flag: %s", value)
	}
	listenPort, err := strconv.Atoi(lt[0][1:])
	if err != nil {
		return err
	}
	targetAddress := strings.Split(lt[1], ";")
	if len(targetAddress) == 0 {
		return fmt.Errorf("fail to parse flag: %s", value)
	}
	*l = append(*l, &service{
		ListenPort:    listenPort,
		TargetAddress: targetAddress,
	})
	return nil
}
func (l serviceList) Run(nodelay bool) error {
	wg := &sync.WaitGroup{}
	for i := range l {
		wg.Add(1)
		if err := l[i].Serve(nodelay, wg); err != nil {
			return err
		}
	}
	wg.Wait()
	return nil
}

type service struct {
	ListenPort    int
	TargetAddress []string
}

func (s *service) Serve(nodelay bool, wg *sync.WaitGroup) error {
	Listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("[::]"),
		Port: s.ListenPort,
	})
	if err != nil {
		return err
	}
	go func(Listener *net.TCPListener, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			conn, err := Listener.AcceptTCP()
			if err != nil {
				fmt.Println(err)
			}
			go s.serveTCP(conn, nodelay)
		}
	}(Listener, wg)
	fmt.Printf("%s listening\n", Listener.Addr())
	return nil
}
func (s *service) serveTCP(conn *net.TCPConn, nodelay bool) {
	defer conn.Close()
	addr, err := net.ResolveTCPAddr("tcp", s.TargetAddress[rand.Intn(len(s.TargetAddress))])
	if err != nil {
		fmt.Println(err)
		return
	}
	remote, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer remote.Close()
	err = conn.SetNoDelay(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = remote.SetNoDelay(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s<>%s<>%s\n", conn.RemoteAddr(), conn.LocalAddr(), remote.RemoteAddr())
	go func(conn, remote *net.TCPConn) {
		io.Copy(conn, remote)
		conn.Close()
		remote.Close()
	}(conn, remote)
	io.Copy(remote, conn)
}
