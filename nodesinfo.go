package main

import (
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type NodesInfoMetrics struct {
	freemem  float64
	allocmem float64
	totalmem float64
	cpus     float64
	cpuload  float64
}

// NodesInfoData Execute the sinfo command and return its output
func NodesInfoData() []byte {
	// sinfo -e -N -h -o%n,%e,%c,%O
	cmd := exec.Command("sinfo", "-h -e -N", "-o%n,%e/%m/%c/%O")
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
func ParseNodesInfoMetrics() map[string]*NodesInfoMetrics {
	nodes := make(map[string]*NodesInfoMetrics)

	lines := strings.Split(string(NodesInfoData()), "\n")

	for _, line := range lines {
		if strings.Contains(line, ",") {
			//node name
			node := strings.Split(line, ",")[0]
			_, key := nodes[node]
			if !key {
				nodes[node] = &NodesInfoMetrics{0, 0, 0, 0, 0}
			}
			info := strings.Split(line, ",")[1]
			freemem, _ := strconv.ParseFloat(strings.Split(info, "/")[0], 64)
			totalmem, _ := strconv.ParseFloat(strings.Split(info, "/")[1], 64)
			allocmem := totalmem - freemem
			cpus, _ := strconv.ParseFloat(strings.Split(info, "/")[2], 64)
			cpuload, _ := strconv.ParseFloat(strings.Split(info, "/")[3], 64)
			nodes[node].freemem = freemem
			nodes[node].totalmem = totalmem
			nodes[node].allocmem = allocmem
			nodes[node].cpus = cpus
			nodes[node].cpuload = cpuload
		}
	}

	return nodes
}

/*
 * Implement the Prometheus Collector interface and feed the
 * Slurm scheduler metrics into it.
 * https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector
 */

// NewNodesInfoCollector function
func NewNodesInfoCollector() *NodesInfoCollector {
	return &NodesInfoCollector{
		freemem:  prometheus.NewDesc("slurm_nodes_freemem", "Free node memory (MB)", nil, nil),
		allocmem: prometheus.NewDesc("slurm_nodes_allocmem", "Allocated node memory (MB)", nil, nil),
		totalmem: prometheus.NewDesc("slurm_nodes_totalmem", "Total node memory (MB)", nil, nil),
		cpus:     prometheus.NewDesc("slurm_nodes_cpus", "Number of node cpus", nil, nil),
		cpuload:  prometheus.NewDesc("slurm_nodes_cpuload", "Node cpu load", nil, nil),
	}
}

//NodesInfoCollector function
type NodesInfoCollector struct {
	freemem  *prometheus.Desc
	allocmem *prometheus.Desc
	totalmem *prometheus.Desc
	cpus     *prometheus.Desc
	cpuload  *prometheus.Desc
}

//Describe Send all metric descriptions
func (nic *NodesInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nic.freemem
	ch <- nic.allocmem
	ch <- nic.totalmem
	ch <- nic.cpus
	ch <- nic.cpuload

}

//Collect function
func (nic *NodesInfoCollector) Collect(ch chan<- prometheus.Metric) {
	pm := ParseNodesInfoMetrics()
	for p := range pm {
		if pm[p].allocmem > 0 {
			ch <- prometheus.MustNewConstMetric(nic.allocmem, prometheus.GaugeValue, pm[p].allocmem, p)
		}
		if pm[p].freemem > 0 {
			ch <- prometheus.MustNewConstMetric(nic.freemem, prometheus.GaugeValue, pm[p].freemem, p)
		}
		if pm[p].totalmem > 0 {
			ch <- prometheus.MustNewConstMetric(nic.totalmem, prometheus.GaugeValue, pm[p].totalmem, p)
		}
		if pm[p].cpus > 0 {
			ch <- prometheus.MustNewConstMetric(nic.cpus, prometheus.GaugeValue, pm[p].cpus, p)
		}
		if pm[p].cpuload > 0 {
			ch <- prometheus.MustNewConstMetric(nic.cpuload, prometheus.GaugeValue, pm[p].cpuload, p)
		}
	}
}
