package main

import (
    "github.com/davecgh/go-spew/spew"
	"log"
    "os"
    "golang.org/x/sys/windows/registry"
    "path/filepath"
    "golang.org/x/sys/windows/svc/eventlog"
    "golang.org/x/sys/windows/svc/mgr"
    "fmt"
    "strconv"
)

type InstallCommand struct {
    HostName     string `short:"H" long:"hostname"      description:"Hostname in Zabbix"`
    ZabbixServer string `short:"z" long:"zabbix-server" description:"Zabbix server <address[:port]>" default:"192.168.1.1:10051"`
    UpdateTime      int `short:"w" long:"update-time"   description:"Update time (sec)" default:"30"`
}

type UninstallCommand struct {
    KeepSettings   bool `short:"s" long:"keep-settings" description:"Do not delete the settings in the Windows registry"`
}

var installCommand InstallCommand
var uninstallCommand UninstallCommand

func init() {
    if installCommand.HostName == "" {
        var err error
        if installCommand.HostName, err = os.Hostname(); err != nil {
            installCommand.HostName = "localhost"
        }
    }

	optsParser.AddCommand("install",
		"Install Windows service",
		"", &installCommand)

    optsParser.AddCommand("uninstall",
        "Uninstall Windows service",
        "", &uninstallCommand)

    _ = spew.Sdump("")
}

func (x *InstallCommand) Execute(args []string) error {
    writeSettings(map[string]string{
        "ZabbixServer": x.ZabbixServer,
        "HostName": x.HostName,
        "UpdateTime": strconv.Itoa(x.UpdateTime),
    })

    if err := installService("zohm", "Zabbix Open Hardware Monitoring"); err != nil {
        log.Fatalf("can't install windows service: %s", err)
    }

	return nil
}

func (x *UninstallCommand) Execute(args []string) error {
    if err := stopService(); err != nil {
        if Opts.Verbose == true {
            log.Printf("can't stop zohm service: %s", err)
        }
    }

    if x.KeepSettings == false {
        if err := registry.DeleteKey(registry.LOCAL_MACHINE, `SOFTWARE\AlexEmergy\zohm`); err != nil {
            if Opts.Verbose == true {
                log.Printf("can't remove settings in registry: %s", err)
            }
        }

        k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\AlexEmergy`, registry.QUERY_VALUE)
        if err != nil {
            if Opts.Verbose == true {
                log.Printf("can't open vendor registry key: %s", err)
            }
        }

        names, _ := k.ReadValueNames(-1)

        if len(names) == 0 {
            if err := registry.DeleteKey(registry.LOCAL_MACHINE, `SOFTWARE\AlexEmergy`); err != nil {
                if Opts.Verbose == true {
                    log.Printf("can't remove vendor in registry: %s", err)
                }
            }
        }
    }

    if err := removeService("zohm"); err != nil {
        if Opts.Verbose == true {
            log.Fatalf("can't remove windows service: %s", err)
        }
    }

    return nil
}

func removeService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}

func installService(name, desc string) error {
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}
	return nil
}

func writeSettings(settings map[string]string) {
    k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\AlexEmergy\zohm`, registry.WRITE)
    if err != nil {
        log.Fatalf("error open registry key: %s\n", err)
    }
    defer k.Close()

    for key, val := range settings {
        if err := k.SetStringValue(key, val); err != nil {
            log.Fatalf("error write settings to registry: %s\n", err)
        }
    }
}

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}
