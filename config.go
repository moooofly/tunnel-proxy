package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	logger "log"
	"os"
	"os/exec"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/moooofly/http-tunnel/services"
	"github.com/moooofly/http-tunnel/services/basic"
	"github.com/moooofly/http-tunnel/services/eavesdropper"

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
	basicArgs := basic.BasicArgs{}
	eavesdropperArgs := eavesdropper.EavesdropperArgs{}

	app = kingpin.New("proxy", "This is a HTTP tunnel proxy.")
	app.Author("moooofly").Version(APP_VERSION)
	isDebug = app.Flag("debug", "debug log output").Default("false").Bool()
	daemon := app.Flag("daemon", "run proxy in background").Default("false").Bool()
	forever := app.Flag("forever", "run proxy in forever,fail and retry").Default("false").Bool()
	logfile := app.Flag("log", "log file path").Default("").String()
	nolog := app.Flag("nolog", "turn off logging").Default("false").Bool()

	// ######### basic ##########
	basicCmd := app.Command("basic", "basic proxy")
	basicArgs.Local = basicCmd.Flag("local", "local ip:port to listen, multiple address use comma split, such as: 0.0.0.0:80,0.0.0.0:443").Short('p').Default(":8080").String()

	// ######### eavesdropper ##########
	eavesdropperCmd := app.Command("eavesdropper", "eavesdropper proxy")
	eavesdropperArgs.Local = eavesdropperCmd.Flag("local", "local ip:port to listen, multiple address use comma split, such as: 0.0.0.0:80,0.0.0.0:443").Short('p').Default(":8080").String()

	serviceName := kingpin.MustParse(app.Parse(os.Args[1:]))

	log := logger.New(os.Stderr, "", logger.Ldate|logger.Ltime)

	flags := logger.Ldate
	if *isDebug {
		flags |= logger.Lshortfile | logger.Lmicroseconds
		cpuProfilingFile, _ = os.Create("cpu.prof")
		memProfilingFile, _ = os.Create("memory.prof")
		blockProfilingFile, _ = os.Create("block.prof")
		goroutineProfilingFile, _ = os.Create("goroutine.prof")
		threadcreateProfilingFile, _ = os.Create("threadcreate.prof")
		pprof.StartCPUProfile(cpuProfilingFile)
	} else {
		flags |= logger.Ltime
	}
	log.SetFlags(flags)
	if *nolog {
		log.SetOutput(ioutil.Discard)
	} else if *logfile != "" {
		f, e := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if e != nil {
			log.Fatal(e)
		}
		log.SetOutput(f)
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
		log.Printf("%s%s [PID] %d running...\n", f, os.Args[0], cmd.Process.Pid)
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
					log.Printf("ERR:%s,restarting...\n", err)
					continue
				}
				cmdReader, err := cmd.StdoutPipe()
				if err != nil {
					log.Printf("ERR:%s,restarting...\n", err)
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
					log.Printf("ERR:%s,restarting...\n", err)
					continue
				}
				pid := cmd.Process.Pid
				log.Printf("worker %s [PID] %d running...\n", os.Args[0], pid)
				if err := cmd.Wait(); err != nil {
					log.Printf("ERR:%s,restarting...", err)
					continue
				}
				log.Printf("worker %s [PID] %d unexpected exited, restarting...\n", os.Args[0], pid)
			}
		}()
		return
	}
	if *logfile == "" {
		if *isDebug {
			log.Println("[profiling] cpu profiling save to file : cpu.prof")
			log.Println("[profiling] memory profiling save to file : memory.prof")
			log.Println("[profiling] block profiling save to file : block.prof")
			log.Println("[profiling] goroutine profiling save to file : goroutine.prof")
			log.Println("[profiling] threadcreate profiling save to file : threadcreate.prof")
		}
	}

	//regist services and run service
	switch serviceName {
	case "basic":
		services.Regist(serviceName, basic.NewBasic(), basicArgs, log)
	case "eavesdropper":
		services.Regist(serviceName, eavesdropper.NewEavesdropper(), eavesdropperArgs, log)
	}

	service, err = services.Run(serviceName, nil)
	if err != nil {
		log.Fatalf("run service [%s] fail, ERR:%s", serviceName, err)
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
