package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type MyHandle struct{}

var (
	server               http.Server
	listener             net.Listener
	hookableSignals      []os.Signal
	sigChan              chan os.Signal
	runningServerReg     sync.RWMutex
	runningServersForked bool
	isChild              bool
)

func init() {

	hookableSignals = []os.Signal{
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		//syscall.SIGINT,
		//syscall.SIGTERM,
		syscall.SIGTSTP,
	}
	sigChan = make(chan os.Signal)

	runningServerReg = sync.RWMutex{}

	runningServersForked = false

	flag.BoolVar(&isChild, "continue", false, "listen on open fd (after forking)")

	flag.Parse()

	log.Printf("isChild is %v\n", isChild)

}

func main() {

	server = http.Server{
		Addr:        ":8888",
		Handler:     &MyHandle{},
		ReadTimeout: 6 * time.Second,
	}

	go handleSignals()

	log.Printf("Actual pid is %d\n", syscall.Getpid())

	var err2 error

	listener, err2 = getListener(server.Addr)

	if err2 != nil {
		log.Println(err2)

	}

	log.Printf("isChild : %v ,listener: %v\n", isChild, listener)

	err := server.Serve(listener)

	if err != nil {
		log.Println(err)
	}

}

func handleSignals() {
	var sig os.Signal

	signal.Notify(
		sigChan,
		hookableSignals...,
	)

	pid := syscall.Getpid()

	for {
		sig = <-sigChan
		log.Println(pid, "Received SIG.", sig)
		switch sig {
		case syscall.SIGHUP:
			log.Println(pid, "Received SIGHUP. forking.")
			err := fork()
			if err != nil {
				log.Println("Fork err:", err)
			}
			//sig.Signal()
		case syscall.SIGUSR1:
			log.Println(pid, "Received SIGUSR1.")

		case syscall.SIGUSR2:
			log.Println(pid, "Received SIGUSR2.")
			//sig.Signal()
		case syscall.SIGINT:
			log.Println(pid, "Received SIGINT.")
			//srv.shutdown()
		case syscall.SIGTERM:
			log.Println(pid, "Received SIGTERM.")
			//shutdown()
		case syscall.SIGTSTP:
			log.Println(pid, "Received SIGTSTP.")
			shutdown()
		default:
			log.Printf("Received %v: nothing i care about...\n", sig)
		}

	}
}
func getListener(laddr string) (l net.Listener, err error) {
	if isChild {

		runningServerReg.RLock()
		defer runningServerReg.RUnlock()

		f := os.NewFile(3, "")

		l, err = net.FileListener(f)
		if err != nil {
			log.Printf("net.FileListener error:", err)
			return
		}

		log.Printf("laddr : %v ,listener: %v \n", laddr, l)

		syscall.Kill(syscall.Getppid(), syscall.SIGTSTP) //干掉父进程

	} else {
		l, err = net.Listen("tcp", laddr)
		if err != nil {
			log.Printf("net.Listen error: %v", err)
			return
		}
	}
	return
}

func fork() (err error) {
	runningServerReg.Lock()

	defer runningServerReg.Unlock()

	// only one server isntance should fork!
	if runningServersForked {
		return errors.New("Another process already forked. Ignoring this one.")
	}

	runningServersForked = true

	log.Println("Restart: forked Start....")

	tl := listener.(*net.TCPListener)
	fl, _ := tl.File()

	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-continue" {
				break
			}
			args = append(args, arg)
		}
	}
	args = append(args, "-continue")

	log.Println(path, args)
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{fl}

	err = cmd.Start()
	if err != nil {
		log.Printf("Restart: Failed to launch, error: %v", err)
	}

	return
}

func shutdown() {

	log.Printf("shutdown Listener :%v\n", listener)
	err := listener.Close()

	if err != nil {
		log.Println(syscall.Getpid(), "Listener.Close() error:", err)
	} else {
		log.Println(syscall.Getpid(), server.Addr, "Listener closed.")
	}

	os.Exit(1)
}

func (*MyHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "URL"+r.URL.String())
}

// func process(sig os.Signal) {
// 	n := signum(sig)
// 	if n < 0 {
// 		return
// 	}
// 	handlers.Lock()
// 	defer handlers.Unlock()
// 	for c, h := range handlers.m {
// 		if h.want(n) {
// 			// send but do not block for it
// 			select {
// 			case c <- sig:
// 			default:
// 			}
// 		}
// 	}
// }

// func init() {
// 	signal_enable(0) // first call - initialize
// 	go loop()
// }

// func loop() {
// 	for {
// 		process(syscall.Note(signal_recv()))
// 	}
// }
