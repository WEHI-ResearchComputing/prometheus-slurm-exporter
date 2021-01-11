package main

import (
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type NodesInfoMetrics struct {
	freemem  float64
	alloc    float64
	totalmem float64
	cpus     float64
	comp     float64
}

func NodesInfoGetMetrics() *NodesInfoMetrics {
	return ParseNodesInfoMetrics(NodesInfoData())
}

func ParseNodesInfoMetrics(input []byte) *NodesInfoMetrics {
	var nm NodesInfoMetrics
	lines := strings.Split(string(input), "\n")

	// Sort and remove all the duplicates from the 'sinfo' output
	sort.Strings(lines)
	lines_uniq := RemoveDuplicates(lines)

	for _, line := range lines_uniq {
		if strings.Contains(line, ",") {
			split := strings.Split(line, ",")
			count, _ := strconv.ParseFloat(strings.TrimSpace(split[0]), 64)
			state := split[1]
			alloc := regexp.MustCompile(`^alloc`)
			comp := regexp.MustCompile(`^comp`)

			switch {
			case alloc.MatchString(state) == true:
				nm.alloc += count
			case comp.MatchString(state) == true:
				nm.comp += count

			}
		}
	}
	return &nm
}

// NodesInfoData Execute the sinfo command and return its output
func NodesInfoData() []byte {
	cmd := exec.Command("sinfo", "-h", "-o %D,%T")
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

/*
 * Implement the Prometheus Collector interface and feed the
 * Slurm scheduler metrics into it.
 * https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector
 */

func NewNodesInfoCollector() *NodesCollector {
	return &NodesInfoCollector{
		alloc: prometheus.NewDesc("slurm_nodesi_alloc", "Allocated nodes", nil, nil),
		comp:  prometheus.NewDesc("slurm_nodesi_comp", "Completing nodes", nil, nil),
	}
}

type NodesInfoCollector struct {
	alloc *prometheus.Desc
	comp  *prometheus.Desc
}

// Send all metric descriptions
func (nc *NodesInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.alloc
	ch <- nc.comp
	ch <- nc.down
	ch <- nc.drain
	ch <- nc.err
	ch <- nc.fail
	ch <- nc.idle
	ch <- nc.maint
	ch <- nc.mix
	ch <- nc.resv
}
func (nc *NodesInfoCollector) Collect(ch chan<- prometheus.Metric) {
	nm := NodesInfoGetMetrics()
	ch <- prometheus.MustNewConstMetric(nc.alloc, prometheus.GaugeValue, nm.alloc)
	ch <- prometheus.MustNewConstMetric(nc.comp, prometheus.GaugeValue, nm.comp)
}
