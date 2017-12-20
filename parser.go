package main

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "strings"
    "regexp"
    "time"
    "encoding/json"
    . "github.com/blacked/go-zabbix"
    "strconv"
)

type discoveryType struct {
    Name string `json:"{#NAME}"`
}

type senderOutput struct {
    Data []discoveryType `json:"data"`
}

func ohmParser(cmd string, settings map[string]string, debug bool) {
    updateTime, err := strconv.Atoi(settings["UpdateTime"])
    if err != nil {
        updateTime = 30
    }

    printOnly := debug
    hostname := settings["HostName"]

    var zabbixServer string
    var zabbixPort int

    zbxServer := strings.Split(settings["ZabbixServer"], ":")

    zabbixServer = zbxServer[0]

    if zbxServer[1] == "" {
        zabbixPort = 10051
    } else {
        var err error
        zabbixPort, err = strconv.Atoi(zbxServer[1])
        if err != nil {
            zabbixPort = 10051
        }
    }

    for {
        out, err := exec.Command(cmd).Output()
        if err != nil {
            log.Fatalf("Can't execute '%s': %s\n", cmd, err)
        }

        sensors, _ := cutFromString(string(out), "Sensors", 3, "-------", 0)

        var heap []string

        for {
            var i string
            i, sensors = cutFromString(sensors, "^\\+-", 0, "^\\|?\\s*$", 0)
            heap = append(heap, i)
            if sensors == "" {
                break
            }
        }

        var discovery []discoveryType
        var metrics []*Metric

        for _, i := range heap {
            var title string

            for n, line := range strings.Split(i, "\n") {
                line = regexp.MustCompile("^.*\\+- ").ReplaceAllString(line, "")
                line = regexp.MustCompile("\\s*\\r").ReplaceAllString(line, "")

                if n == 0 {
                    title = line
                } else {
                    fields := regexp.MustCompile("\\s*:\\s*").Split(line, 2)
                    if len(fields) <= 1 {
                        continue
                    }
                    val := strings.Fields(fields[1])
                    if len(val) <= 1 {
                        continue
                    }

                    if (printOnly == true) {
                        fmt.Printf("%s %s %s: %s\n", title, fields[0], val[len(val) - 1], val[0])
                    } else {
                        key := fmt.Sprintf("%s %s %s", title, fields[0], val[len(val) - 1])

                        discovery = append(discovery, discoveryType{
                            Name: key,
                        });

                        metrics = append(metrics,
                            NewMetric(hostname, fmt.Sprintf(`ohm.sensors.item[%s]`, key), val[0]))
                    }
                }
            }
        }

        if printOnly == true {
            fmt.Println(discovery)
            os.Exit(0)
        } else {
            metrics = append(metrics, NewMetric(hostname, "ohm.sensors.discovery", buildDiscovery(discovery)))
            packet := NewPacket(metrics)
            z := NewSender(zabbixServer, zabbixPort)
            if b, e := z.Send(packet); e != nil {
                log.Printf("%s", e)
            } else {
                log.Printf("%s", b)
            }
        }

        time.Sleep(time.Duration(updateTime) * time.Second)
    }
}

func buildDiscovery(data []discoveryType) string {
    discovery, err := json.Marshal(senderOutput{
        Data: data,
    })

    if err != nil {
        log.Fatal(fmt.Sprintf("Can't build discovery JSON: %s\n", err))
    }

    //return strings.Replace(string(discovery), "\"", "\\\"", -1)
    return string(discovery)
}

func cutFromString(data string, a string, aOffset int, b string, bOffset int) (string, string) {
    firstLine := -1
    lastLine := -1

    stringsList := strings.Split(data, "\n")

    for i, line := range stringsList {
        reA := regexp.MustCompile(a)
        if reA.MatchString(line) == true {
            firstLine = i
        }

        if firstLine >= 0 {
            reB := regexp.MustCompile(b)
            if reB.MatchString(line) == true {
                lastLine = i
                break
            }
        }
    }

    if lastLine + bOffset > len(stringsList) - 3 {
        return "", ""
    }

    m := strings.Join(stringsList[firstLine + aOffset:lastLine + bOffset], "\n");
    d := strings.Join(stringsList[lastLine + bOffset + 1:], "\n")

    return m, d
}

