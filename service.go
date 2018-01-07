package main

import (
	"fmt"
	"time"
	"log"
	"os"
	"path/filepath"
	"io/ioutil"
	"strconv"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/registry"

	"github.com/rakyll/statik/fs"
	_ "github.com/emergy/zohm/statik"

	//"github.com/davecgh/go-spew/spew"
	"github.com/emergy/zohm/parser"
)

var elog debug.Log
var DebugMode bool

type zohmService struct{}

func (m *zohmService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\AlexEmergy\zohm`, registry.QUERY_VALUE)
	if err != nil {
		if Opts.Verbose == true {
			log.Printf("can't open vendor registry key: %s", err)
		}
	}

	keys, _ := k.ReadValueNames(-1)

	settings := make(map[string]string)

	for _, key := range keys {
		val, _, err := k.GetStringValue(key)
		if err != nil {
			log.Fatalf("can't read registry key '%s': %s", key, err)
		}

		settings[key] = val
	}

    updateTime, err := strconv.Atoi(settings["UpdateTime"])
    if err != nil {
        updateTime = 30
    }

    dir, err := ioutil.TempDir("", "OpenHardwareMonitor")
    if err != nil {
        log.Fatalf("Can't create tempory directory: %s\n", err)
    }
    defer os.RemoveAll(dir)

    unpackStatik(dir, []string{
        "OpenHardwareMonitorReport.exe",
        "OpenHardwareMonitorLib.dll",
    })

    cmd := filepath.Join(dir, "OpenHardwareMonitorReport.exe")

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	//slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-tick:
			data := parser.Exec(cmd)
			zabbixSend(data, settings, DebugMode)

			if DebugMode {
				//spew.Dump(data)
				os.Exit(0)
			}

			time.Sleep(time.Duration(updateTime) * time.Second)
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	DebugMode = isDebug
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run("zohm", &zohmService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}

func unpackStatik (dir string, fileNames []string) {
    embededFS, err := fs.New()
	if err != nil {
		log.Fatalf("can't create fs object: %s", err)
	}

    for _, fileName := range fileNames {
        file, err := embededFS.Open("/" + fileName)
		if err != nil {
			log.Fatalf("can't open file object '%s' from embededFS: %s", fileName, err)
		}

        data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("can't read file '%s': %s", fileName, err)
		}

        if err = ioutil.WriteFile(filepath.Join(dir, fileName), data, 0755); err != nil {
			log.Fatalf("can't write file '%s' to dir: %s", fileName, dir)
		}
    }
}


