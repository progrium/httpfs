package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	Version string

	showVersion = flag.Bool("v", false, "show version")
	mountpath   = flag.String("mount", "", "path to mount filesystem")
	mount       Mounter
	cmd         *exec.Cmd
)

type Mounter interface {
	Prepare() error
	Mount() error
	Unmount() error
}

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println(Version)
		return
	}

	if len(flag.Args()) < 1 || mountpath == nil {
		println("Usage: httpfs -mount=MOUNTPATH EXEC-PATH [EXEC-ARGS...]")
		os.Exit(1)
	}

	port, err := findPort()
	if err != nil {
		log.Fatal("unable to find unused port: ", err)
	}
	os.Setenv("PORT", strconv.Itoa(port))

	cmd = exec.Command(flag.Arg(0), flag.Args()[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal("unable to start subprocess: ", err)
	}

	if mountpath != nil {
		fs := &httpFS{
			baseURL: fmt.Sprintf("http://localhost:%d", port),
		}

		// poll until server is up or 5 sec timeout
		_, err := fs.Stat(".")
		for err != nil {
			if time.Since(startTime).Seconds() >= 5 {
				break
			}
			<-time.After(500 * time.Millisecond)
			_, err = fs.Stat(".")
		}
		if err != nil {
			tryShutdown()
			log.Fatal("unable to stat httpfs: ", err)
		}

		mount = &fuseMount{
			fs:   fs,
			path: *mountpath,
		}
		if err := mount.Prepare(); err != nil {
			tryShutdown()
			log.Fatal("mount prep failed: ", err)
		}
		if err := mount.Mount(); err != nil {
			tryUnmount()
			tryShutdown()
			log.Fatal("unable to mount: ", err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)
	go func() {
		for sig := range sigChan {
			if sig == os.Interrupt {
				tryUnmount()
				tryShutdown()
				return
			}
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		tryUnmount()
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatal("error running subprocess:", err)
	}

	tryUnmount()
}

func tryUnmount() {
	if mount == nil {
		return
	}
	if err := mount.Unmount(); err != nil {
		log.Println("unable to unmount:", err)
	}
}

func tryShutdown() {
	if cmd == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		panic(err)
	}
	syscall.Kill(-pgid, syscall.SIGTERM)
}

func findPort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
