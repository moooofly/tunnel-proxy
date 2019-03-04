package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/moooofly/tunnel-proxy/services"
	"github.com/moooofly/tunnel-proxy/services/basic"
	"github.com/moooofly/tunnel-proxy/services/eavesdropper"
	"github.com/sirupsen/logrus"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app     *kingpin.Application
	service *services.ServiceItem
	cmd     *exec.Cmd
	cpuProfilingFile, memProfilingFile, blockProfilingFile,
	goroutineProfilingFile, threadcreateProfilingFile *os.File
	isDebug *bool
)

func initConfig() (err error) {

	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "_timestamp",
			logrus.FieldKeyLevel: "_level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	basicArgs := basic.BasicArgs{}
	eavesdropperArgs := eavesdropper.EavesdropperArgs{}

	app = kingpin.New("proxy", "This is a HTTP Tunnel Proxy.")
	app.Author("moooofly").Version(APP_VERSION)
	isDebug = app.Flag("debug", "debug log output").Default("false").Bool()
	daemon := app.Flag("daemon", "run proxy in background").Default("false").Bool()
	forever := app.Flag("forever", "run proxy in forever, fail and retry").Default("false").Bool()
	logfile := app.Flag("log", "log file path").Default("").String()
	nolog := app.Flag("nolog", "turn off logging").Default("false").Bool()

	// ######### basic ##########
	basicCmd := app.Command("basic", "basic proxy")
	basicArgs.Local = basicCmd.Flag("local", "Address to listen, multiple addresses separating by comma, e.g. \"0.0.0.0:80,0.0.0.0:443\"").Short('l').Default(":8080").String()
	basicArgs.White = basicCmd.Flag("white", "white-list file, please set one domain each line").Default("whitelist.cfg").Short('w').String()
	basicArgs.AuthFile = basicCmd.Flag("auth-file", "HTTP basic auth file, please set one \"username:password\" each line").Short('A').String()
	basicArgs.Auth = basicCmd.Flag("auth-args", "HTTP basic auth arguments, can set mutiple times, e.g. \"-a user1:pass1 -a user2:pass2\"").Short('a').Strings()

	// ######### eavesdropper ##########
	eavesdropperCmd := app.Command("eavesdropper", "eavesdropper proxy")
	eavesdropperArgs.Local = eavesdropperCmd.Flag("local", "local ip:port to listen, multiple address use comma split, such as: 0.0.0.0:80,0.0.0.0:443").Short('p').Default(":8080").String()
	eavesdropperArgs.White = eavesdropperCmd.Flag("white", "white-list file, please set one domain each line").Default("whitelist.cfg").Short('w').String()
	eavesdropperArgs.AuthFile = eavesdropperCmd.Flag("auth-file", "HTTP basic auth file, please set one \"username:password\" each line").Short('A').String()
	eavesdropperArgs.Auth = eavesdropperCmd.Flag("auth-args", "HTTP basic auth arguments, can set mutiple times, e.g. \"-a user1:pass1 -a user2:pass2\"").Short('a').Strings()

	serviceName := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *isDebug {
		cpuProfilingFile, _ = os.Create("cpu.prof")
		memProfilingFile, _ = os.Create("memory.prof")
		blockProfilingFile, _ = os.Create("block.prof")
		goroutineProfilingFile, _ = os.Create("goroutine.prof")
		threadcreateProfilingFile, _ = os.Create("threadcreate.prof")
		pprof.StartCPUProfile(cpuProfilingFile)
	}

	if *nolog {
		logrus.SetOutput(ioutil.Discard)
	} else if *logfile != "" {
		f, e := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if e != nil {
			logrus.Fatal(e)
		}
		logrus.SetOutput(f)
	}
	if *daemon {
		args := []string{}
		for _, arg := range os.Args[1:] {
			if arg != "--daemon" {
				args = append(args, arg)
			}
		}
		cmd = exec.Command(os.Args[0], args...)
		cmd.Start()
		f := ""
		if *forever {
			f = "forever "
		}
		logrus.Printf("%s%s [PID] %d running...\n", f, os.Args[0], cmd.Process.Pid)
		os.Exit(0)
	}
	if *forever {
		args := []string{}
		for _, arg := range os.Args[1:] {
			if arg != "--forever" {
				args = append(args, arg)
			}
		}
		go func() {
			defer func() {
				if e := recover(); e != nil {
					fmt.Printf("crashed, err: %s\nstack:%s", e, string(debug.Stack()))
				}
			}()
			for {
				if cmd != nil {
					cmd.Process.Kill()
					time.Sleep(time.Second * 5)
				}
				cmd = exec.Command(os.Args[0], args...)
				cmdReaderStderr, err := cmd.StderrPipe()
				if err != nil {
					logrus.Printf("ERR: %s, restarting...\n", err)
					continue
				}
				cmdReader, err := cmd.StdoutPipe()
				if err != nil {
					logrus.Printf("ERR: %s, restarting...\n", err)
					continue
				}
				scanner := bufio.NewScanner(cmdReader)
				scannerStdErr := bufio.NewScanner(cmdReaderStderr)
				go func() {
					defer func() {
						if e := recover(); e != nil {
							fmt.Printf("crashed, err: %s\nstack:%s", e, string(debug.Stack()))
						}
					}()
					for scanner.Scan() {
						fmt.Println(scanner.Text())
					}
				}()
				go func() {
					defer func() {
						if e := recover(); e != nil {
							fmt.Printf("crashed, err: %s\nstack:%s", e, string(debug.Stack()))
						}
					}()
					for scannerStdErr.Scan() {
						fmt.Println(scannerStdErr.Text())
					}
				}()
				if err := cmd.Start(); err != nil {
					logrus.Printf("ERR: %s, restarting...\n", err)
					continue
				}
				pid := cmd.Process.Pid
				logrus.Printf("worker %s [PID] %d running...\n", os.Args[0], pid)
				if err := cmd.Wait(); err != nil {
					logrus.Printf("ERR: %s, restarting...", err)
					continue
				}
				logrus.Printf("worker %s [PID] %d unexpected exited, restarting...\n", os.Args[0], pid)
			}
		}()
		return
	}
	if *logfile == "" {
		if *isDebug {
			logrus.Println("[profiling] cpu profiling save to file : cpu.prof")
			logrus.Println("[profiling] memory profiling save to file : memory.prof")
			logrus.Println("[profiling] block profiling save to file : block.prof")
			logrus.Println("[profiling] goroutine profiling save to file : goroutine.prof")
			logrus.Println("[profiling] threadcreate profiling save to file : threadcreate.prof")
		}
	}

	switch serviceName {
	case "basic":
		services.Regist(serviceName, basic.NewBasic(), basicArgs, logrus.StandardLogger())
	case "eavesdropper":
		services.Regist(serviceName, eavesdropper.NewEavesdropper(), eavesdropperArgs, logrus.StandardLogger())
	}

	service, err = services.Run(serviceName, nil)
	if err != nil {
		logrus.Fatalf("run service [%s] fail, ERR:%s", serviceName, err)
	}
	return
}

func saveProfiling() {
	goroutine := pprof.Lookup("goroutine")
	goroutine.WriteTo(goroutineProfilingFile, 1)
	heap := pprof.Lookup("heap")
	heap.WriteTo(memProfilingFile, 1)
	block := pprof.Lookup("block")
	block.WriteTo(blockProfilingFile, 1)
	threadcreate := pprof.Lookup("threadcreate")
	threadcreate.WriteTo(threadcreateProfilingFile, 1)
	pprof.StopCPUProfile()
}
