package main

import (
    "github.com/emergy/zohm/parser"
    "github.com/davecgh/go-spew/spew"
    . "github.com/blacked/go-zabbix"
    "strings"
    "strconv"
	"log"
	"fmt"
)

func zabbixSend(dataHeap []parser.DataStruct, settings map[string]string, debug bool) {
    _ = spew.Sdump(dataHeap)
	var t *Metric
	_ = t

    if debug {
        spew.Dump(settings, dataHeap)
    }

	zabbixServer, zabbixPort := splitServerPort(settings["ZabbixServer"])
	z := NewSender(zabbixServer, zabbixPort)

    for _, d := range dataHeap {
	    //var discoveryMetrics []*Metric
        discoveryMetric := NewMetric(settings["HostName"], d.DiscoveryKey, d.DiscoveryString)
		//discoveryMetrics = append(discoveryMetrics, discoveryMetric)
        discoveryPacket := NewPacket([]*Metric{discoveryMetric})
        discoverySenderOutput, err := z.Send(discoveryPacket)
		if err != nil {
			log.Printf("Can't send packet to zabbix server: %s", err)
		}

        if debug {
			log.Printf("%s\n", discoverySenderOutput)
		}

		var metrics []*Metric

		for k, v := range d.ItemsList {
			metric := NewMetric(settings["HostName"], fmt.Sprintf("%s[%s]", d.ItemKey, k), v)
			metrics = append(metrics, metric)
		}

		packet := NewPacket(metrics)
		senderOutput, sndrErr := z.Send(packet)
		if sndrErr != nil {
			log.Printf("Can't send packet to zabbix server: %s", err)
		}

		if debug {
			log.Printf("%s\n", senderOutput)
		}
    }
}

func splitServerPort(s string) (string, int) {
	sp := strings.Split(s, ":")
    port, err := strconv.Atoi(sp[1])
    if err != nil {
        port = 10051
    }
	return sp[0], port
}
