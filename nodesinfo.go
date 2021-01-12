/* Copyright 2020 Julie Iskander

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>. */

package main

import (
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

//NodesInfoMetrics struct with all info for individual nodes
type NodesInfoMetrics struct {
	freemem  float64
	allocmem float64
	totalmem string
	cpus     string
	cpuload  float64
	state    string
	feature  string
	weight   string
}

//MetricKey struct for the bytes/node metric
type MetricKey struct {
	state   string
	feature string
}

// NodesInfoData Execute the sinfo command and return its output
func NodesInfoData() []byte {
	//sinfo -e -N -h -o%n,%e,%m,%c,%O,%T,%b,%w
	cmd := exec.Command("sinfo", "-h", "-e", "-N", "-o%n,%e,%m,%c,%O,%T,%b,%w")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	out, _ := ioutil.ReadAll(stdout)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return out
}

//NodesDataInfoData to execute a generic cmd that is sent as an argurment
func NodesDataInfoData(cmd *exec.Cmd) []byte {
	//sinfo -e -h -o%e,%T,%b
	//cmd := exec.Command("sinfo", "-h", "-e", "-o%e,%T,%b")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	out, _ := ioutil.ReadAll(stdout)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return out
}

//ParseNodesInfoMetrics function parse return from Data function
func ParseNodesInfoMetrics(input []byte) map[string]*NodesInfoMetrics {
	nodes := make(map[string]*NodesInfoMetrics)
	//nodebytes := map[Key]float64{}
	//nodecpus := map[Key]float64{}
	lines := strings.Split(string(input), "\n")

	for _, line := range lines {
		if strings.Contains(line, ",") {

			//node name
			node := strings.Split(line, ",")[0]
			_, key := nodes[node]
			if !key {
				nodes[node] = &NodesInfoMetrics{0, 0, "", "", 0, "", "", ""}
			}
			freemem, _ := strconv.ParseFloat(strings.Split(line, ",")[1], 64)
			totalmem := strings.Split(line, ",")[2]
			t, _ := strconv.ParseFloat(totalmem, 64)
			allocmem := t - freemem
			cpus := strings.Split(line, ",")[3]
			cpuload, _ := strconv.ParseFloat(strings.Split(line, ",")[4], 64)
			state := strings.Split(line, ",")[5]
			feature := strings.Split(line, ",")[6]
			weight := strings.Split(line, ",")[7]

			nodes[node].freemem = freemem
			nodes[node].totalmem = totalmem
			nodes[node].allocmem = allocmem
			nodes[node].cpus = cpus
			nodes[node].cpuload = cpuload
			nodes[node].state = state
			nodes[node].feature = feature
			nodes[node].weight = weight

		}
	}

	return nodes
}

/*ParseNodesDataMetrics function parse return from Data function
and returns accumulative total grouped by feature and state
*/
func ParseNodesDataMetrics(input []byte) map[MetricKey]float64 {
	//log.Println(input)
	data := map[MetricKey]float64{}

	lines := strings.Split(string(input), "\n")
	//log.Println(lines)
	//-OAllocMem:10:,FreeMem:10:,Memory:10:,StateLong:10:,Features:10,:free
	for _, line := range lines {
		if strings.Contains(line, ":") {

			feature := strings.TrimSpace(strings.Split(line, ":")[4])
			state := strings.TrimSpace(strings.Split(line, ":")[5])
			alloc, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(line, ":")[0]), 64)
			free, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(line, ":")[1]), 64)
			//total, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(line, ":")[2]), 64)
			s := strings.TrimSpace(strings.Split(line, ":")[3])
			_, ok := data[MetricKey{state, feature}]
			log.Println(data[MetricKey{state, feature}])
			if !ok {
				data[MetricKey{state, feature}] = 0
			}
			if s == "mixed" {
				data[MetricKey{"free", feature}] += free
				data[MetricKey{"alloc", feature}] += alloc
				/*}
				if state == "drained" || state == "free" {
					data[MetricKey{state, feature}] += free
				*/
			} else {
				data[MetricKey{state, feature}] += alloc
			}

		}
	}
	return data
}

//NodesInfoGetMetrics fun
func NodesInfoGetMetrics() map[string]*NodesInfoMetrics {
	return ParseNodesInfoMetrics(NodesInfoData())
}

/*
 * Implement the Prometheus Collector interface and feed the
 * Slurm scheduler metrics into it.
 * https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector
 */

// NewNodesInfoCollector function
func NewNodesInfoCollector() *NodesInfoCollector {
	labels := []string{"node", "state", "totalmem", "cpus", "feature", "weight"}
	labelsbyte := []string{"state", "feature"}
	return &NodesInfoCollector{
		freemem:  prometheus.NewDesc("slurm_node_freemem", "free node memory (MB)", labels, nil),
		allocmem: prometheus.NewDesc("slurm_node_allocmem", "allocated node memory (MB)", labels, nil),
		cpuload:  prometheus.NewDesc("slurm_node_cpuload", "node cpu load", labels, nil),
		bytes:    prometheus.NewDesc("slurm_nodes_bytes", "total size of allocated/requested memory", labelsbyte, nil),
	}
}

//NodesInfoCollector function
type NodesInfoCollector struct {
	freemem  *prometheus.Desc
	allocmem *prometheus.Desc
	cpuload  *prometheus.Desc
	bytes    *prometheus.Desc
}

//Describe Send all metric descriptions
func (nic *NodesInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nic.freemem
	ch <- nic.allocmem
	ch <- nic.cpuload
	ch <- nic.bytes
}

//Collect function
func (nic *NodesInfoCollector) Collect(ch chan<- prometheus.Metric) {
	pm := NodesInfoGetMetrics()
	for p := range pm {
		if pm[p].allocmem >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.allocmem, prometheus.GaugeValue,
				pm[p].allocmem, p, pm[p].state, pm[p].totalmem, pm[p].cpus, pm[p].feature, pm[p].weight)
		}
		if pm[p].freemem >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.freemem, prometheus.GaugeValue,
				pm[p].freemem, p, pm[p].state, pm[p].totalmem, pm[p].cpus, pm[p].feature, pm[p].weight)
		}
		if pm[p].cpuload >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.cpuload, prometheus.GaugeValue,
				pm[p].cpuload, p, pm[p].state, pm[p].totalmem, pm[p].cpus, pm[p].feature, pm[p].weight)
		}

	}
	//sinfo -e -o%e,%f,alloc --state allocated
	cmd := exec.Command("sinfo", "-h", "-e", "--state=allocated", "-OAllocMem:10:,FreeMem:10:,Memory:10:,StateLong:10:,Features:10,:alloc")
	data := ParseNodesDataMetrics(NodesDataInfoData(cmd))

	for d := range data {
		if data[d] >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.bytes, prometheus.GaugeValue,
				data[d], d.state, d.feature)
		}
	}
	cmd = exec.Command("sinfo", "-h", "-e", "--state=idle", "-OAllocMem:10:,FreeMem:10:,Memory:10:,StateLong:10:,Features:10,:free")
	data = ParseNodesDataMetrics(NodesDataInfoData(cmd))
	for d := range data {
		if data[d] >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.bytes, prometheus.GaugeValue,
				data[d], d.state, d.feature)
		}
	}
	cmd = exec.Command("sinfo", "-h", "-e", "--state=drained", "-OAllocMem:10:,FreeMem:10:,Memory:10:,StateLong:10:,Features:10,:drained")
	data = ParseNodesDataMetrics(NodesDataInfoData(cmd))
	for d := range data {
		if data[d] >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.bytes, prometheus.GaugeValue,
				data[d], d.state, d.feature)
		}
	}
	cmd = exec.Command("sinfo", "-h", "-e", "--state=maint", "-OAllocMem:10:,FreeMem:10:,Memory:10:,StateLong:10:,Features:10,:maint")
	data = ParseNodesDataMetrics(NodesDataInfoData(cmd))
	for d := range data {
		if data[d] >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.bytes, prometheus.GaugeValue,
				data[d], d.state, d.feature)
		}
	}

	cmd = exec.Command("sinfo", "-h", "-e", "--state=completing", "-OAllocMem:10:,FreeMem:10:,Memory:10:,StateLong:10:,Features:10,:completing")
	data = ParseNodesDataMetrics(NodesDataInfoData(cmd))
	for d := range data {
		if data[d] >= 0 {
			ch <- prometheus.MustNewConstMetric(nic.bytes, prometheus.GaugeValue,
				data[d], d.state, d.feature)
		}
	}
}
