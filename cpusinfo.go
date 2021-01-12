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

type CPUsInfoMetrics struct {
	alloc   float64
	idle    float64
	other   float64
	total   float64
	feature string
}

// CPUsInfoGetMetrics function
func CPUsInfoGetMetrics() *CPUsInfoMetrics {
	return ParseCPUsInfoMetrics(CPUsInfoData())
}

// ParseCPUsInfoMetrics function
func ParseCPUsInfoMetrics(input []byte) *CPUsInfoMetrics {
	var cm CPUsInfoMetrics
	if strings.Contains(string(input), "/") {
		splitted := strings.Split(strings.TrimSpace(string(input)), "/")
		cm.alloc, _ = strconv.ParseFloat(splitted[0], 64)
		cm.idle, _ = strconv.ParseFloat(splitted[1], 64)
		cm.other, _ = strconv.ParseFloat(splitted[2], 64)
		cm.total, _ = strconv.ParseFloat(splitted[3], 64)
		cm.feature = splitted[4]
	}
	return &cm
}

// Execute the sinfo command and return its output
func CPUsInfoData() []byte {
	cmd := exec.Command("sinfo", "-h", "-o %C/%f")
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

func NewCPUsInfoCollector() *CPUsInfoCollector {
	return &CPUsInfoCollector{
		alloc: prometheus.NewDesc("slurm_CPUsInfo_alloc", "Allocated CPUsInfo", nil, nil),
		idle:  prometheus.NewDesc("slurm_CPUsInfo_idle", "Idle CPUsInfo", nil, nil),
		other: prometheus.NewDesc("slurm_CPUsInfo_other", "Mix CPUsInfo", nil, nil),
		total: prometheus.NewDesc("slurm_CPUsInfo_total", "Total CPUsInfo", nil, nil),
	}
}

type CPUsInfoCollector struct {
	alloc *prometheus.Desc
	idle  *prometheus.Desc
	other *prometheus.Desc
	total *prometheus.Desc
}

// Send all metric descriptions
func (cc *CPUsInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.alloc
	ch <- cc.idle
	ch <- cc.other
	ch <- cc.total
}
func (cc *CPUsInfoCollector) Collect(ch chan<- prometheus.Metric) {
	cm := CPUsInfoGetMetrics()
	ch <- prometheus.MustNewConstMetric(cc.alloc, prometheus.GaugeValue, cm.alloc)
	ch <- prometheus.MustNewConstMetric(cc.idle, prometheus.GaugeValue, cm.idle)
	ch <- prometheus.MustNewConstMetric(cc.other, prometheus.GaugeValue, cm.other)
	ch <- prometheus.MustNewConstMetric(cc.total, prometheus.GaugeValue, cm.total)
}
