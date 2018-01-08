package parser

import (
    "fmt"
    "log"
    "os/exec"
    "strings"
    "regexp"
//    "time"
    "encoding/json"
//    . "github.com/blacked/go-zabbix"
//    "strconv"
    "github.com/davecgh/go-spew/spew"
)

type DataStruct struct {
	DiscoveryString string
	DiscoveryKey string
	ItemsList map[string]string
	ItemKey string
}

func Exec(cmd string) []DataStruct {
    _ = spew.Sdump("")
    _ = fmt.Sprintf("")

    var rv []DataStruct

    out, err := exec.Command(cmd).Output()
    if err != nil {
        log.Fatalf("Can't execute '%s': %s\n", cmd, err)
    }

    ohmReport := strings.Replace(string(out),"\r", "", -1)

    sensors := parseSensors(ohmReport)
    rv = append(rv, sensors)

    smart := parseSMART(ohmReport)
    rv = append(rv, smart)

    return rv
}

func parseSMART(data string) DataStruct {
    var smart string

    type discoveryStruct struct {
        ID string `json:"{#ID}"`
        Thres string `json:"{#THRES}"`
        Description string `json:"{#DESCRIPTION}"`
    }

    var discoveryList []discoveryStruct

    if m := regexp.MustCompile(`(?ms:^GenericHarddisk.*ID\s+Description.*?\n(.*?)\n\n)`).FindStringSubmatch(data); len(m) > 1 {
        smart = m[1]
    } else {
        log.Fatal("can't find SMART in in ohmReport")
    }

    itemsList := make(map[string]string)

    for _, line := range strings.Split(smart, "\n") {
        if g, m := lineParser(line, `^\s*(?P<ID>[[:xdigit:]]{2})\s+(?P<Description>.*?)\s{2,}(?P<RawValue>[[:xdigit:]]+)\s+(?P<Worst>\d+)\s+(?P<Value>\d+)\s+(?P<Thres>\d+)\s+(?P<Physical>.*?)\s*$`); m == true {
            discoveryList = append(discoveryList, discoveryStruct{
                ID: g["ID"],
                Thres: g["Thres"],
                Description: g["Description"],
            })

            itemsList[fmt.Sprintf("RawValue:%s", g["ID"])] = g["RawValue"]
            itemsList[fmt.Sprintf("Value:%s", g["ID"])] = g["Value"]
            itemsList[fmt.Sprintf("Worst:%s", g["ID"])] = g["Worst"]
        }
    }

    type discoveryDataStruct struct {
        Data []discoveryStruct `json:"data"`
    }

    discoveryString, err := json.Marshal(discoveryDataStruct{
        Data: discoveryList,
    })

    if err != nil {
        log.Fatal(fmt.Sprintf("Can't build discovery JSON: %s\n", err))
    }

    return DataStruct{
        DiscoveryString: string(discoveryString),
        DiscoveryKey: "zohm.smart.discovery",
        ItemsList: itemsList,
        ItemKey: "zohm.smart.item",
    }
}

func parseSensors(data string) DataStruct {
    var sensors string

    type discoveryStruct struct {
        DeviceName string `json:"{#DEVICENAME}"`
        DevicePath string `json:"{#DEVICEPATH}"`
        MetricName string `json:"{#METRICNAME}"`
        MetricPath string `json:"{#METRICPATH}"`
    }

	var discoveryList []discoveryStruct

	itemsList := make(map[string]string)

    if m := regexp.MustCompile(`(?ms:^Sensors.*?------$)`).FindAllString(data, -1); len(m) > 0 {
        sensors = m[0]
    } else {
        log.Fatal("can't find sensors report in ohmReport")
    }

    var device map[string]string

    for _, line := range strings.Split(sensors, "\n") {
        if g, m := lineParser(line, `^\+-\s+(?P<DeviceName>.*?)\s+\((?P<DevicePath>.*?)\)`); m == true {
            device = g
        }

        if g, m := lineParser(line, `\+-\s+(?P<MetricName>.*?)\s+:\s*(?P<Value>\S+)\s.*\((?P<MetricPath>.+)\)`); m == true {
			discovery := discoveryStruct{
				DeviceName: device["DeviceName"],
				DevicePath: device["DevicePath"],
				MetricName: g["MetricName"],
				MetricPath: g["MetricPath"],
			}
			
			discoveryList = append(discoveryList, discovery)
			path := device["DevicePath"] + g["MetricPath"]

			itemsList[path] = g["Value"]
        }
    }

	type discoveryDataStruct struct {
		Data []discoveryStruct `json:"data"`
	}

	discoveryString, err := json.Marshal(discoveryDataStruct{
		Data: discoveryList,
	})

    if err != nil {
        log.Fatal(fmt.Sprintf("Can't build discovery JSON: %s\n", err))
    }

	return DataStruct{
		DiscoveryString: string(discoveryString),
		DiscoveryKey: "zohm.sensors.discovery",
		ItemsList: itemsList,
		ItemKey: "zohm.sensors.item",
	}
}

func lineParser(s string, r string) (map[string]string, bool) {
    re := regexp.MustCompile(r)
    m := re.FindStringSubmatch(s)
    rv := make(map[string]string)

    if len(m) == 0 {
        return rv, false
    }

    for n, i := range m {
        if (n == 0) {
            continue
        }
        name := re.SubexpNames()[n]
        rv[name] = i
    }

    return rv, true
}

