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
	state    string
}

// NodesInfoData Execute the sinfo command and return its output
func NodesInfoData() []byte {
	//sinfo -e -N -h -o%n,%e,%m,%c,%O,%T
	cmd := exec.Command("sinfo", "-h", "-e", "-N", "-o%n,%e,%m,%c,%O,%T")
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

	lines := strings.Split(string(input), "\n")

	for _, line := range lines {
		if strings.Contains(line, ",") {

			//node name
			node := strings.Split(line, ",")[0]
			_, key := nodes[node]
			if !key {
				nodes[node] = &NodesInfoMetrics{0, 0, 0, 0, 0}
			}
			freemem, _ := strconv.ParseFloat(strings.Split(line, ",")[1], 64)
			totalmem, _ := strconv.ParseFloat(strings.Split(line, ",")[2], 64)
			allocmem := totalmem - freemem
			cpus, _ := strconv.ParseFloat(strings.Split(line, ",")[3], 64)
			cpuload, _ := strconv.ParseFloat(strings.Split(line, ",")[4], 64)
			state := strings.Split(line, ",")[5]

			nodes[node].freemem = freemem

			nodes[node].totalmem = totalmem
			nodes[node].allocmem = allocmem
			nodes[node].cpus = cpus
			nodes[node].cpuload = cpuload
			nodes[node].state = state

		}
	}

	return nodes
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
	labels := []string{"node"}
	return &NodesInfoCollector{
		freemem:  prometheus.NewDesc("slurm_node_freemem", "Free node memory (MB)", labels, nil),
		allocmem: prometheus.NewDesc("slurm_node_allocmem", "Allocated node memory (MB)", labels, nil),
		totalmem: prometheus.NewDesc("slurm_node_totalmem", "Total node memory (MB)", labels, nil),
		cpus:     prometheus.NewDesc("slurm_node_cpus", "Number of node cpus", labels, nil),
		cpuload:  prometheus.NewDesc("slurm_node_cpuload", "Node cpu load", labels, nil),
		state:    prometheus.NewDesc("slurm_node_state", "Node state", labels, nil),
	}
}

//NodesInfoCollector function
type NodesInfoCollector struct {
	freemem  *prometheus.Desc
	allocmem *prometheus.Desc
	totalmem *prometheus.Desc
	cpus     *prometheus.Desc
	cpuload  *prometheus.Desc
	state    *prometheus.Desc
}

//Describe Send all metric descriptions
func (nic *NodesInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nic.freemem
	ch <- nic.allocmem
	ch <- nic.totalmem
	ch <- nic.cpus
	ch <- nic.cpuload
	ch <- nic.state
}

//Collect function
func (nic *NodesInfoCollector) Collect(ch chan<- prometheus.Metric) {
	pm := NodesInfoGetMetrics()
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
		if pm[p].state != "" {
			ch <- prometheus.MustNewConstMetric(nic.state, prometheus.GaugeValue, pm[p].state, p)
		}
	}
}
