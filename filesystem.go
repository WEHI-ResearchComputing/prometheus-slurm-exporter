/* Copyright 2017 Victor Penso, Matteo Dessalvi

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
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/prometheus/client_golang/prometheus"
)

//FSMetrics to collect the partition name, size and used MB
type FSMetrics struct {
	size float64
	used float64
}

// FSGetMetrics Returns the scheduler metrics
func FSGetMetrics() map[string]*FSMetrics {
	return ParseFSMetrics(FSData())
}

//ParseFSMetrics to parse the output from the cmd executed on vc7-shared
func ParseFSMetrics(input []byte) map[string]*FSMetrics {
	fss := make(map[string]*FSMetrics)
	lines := strings.Split(string(input), "\n")
	for _, line := range lines {
		if strings.Contains(line, ",") {
			splitted := strings.Split(line, ",")
			name := splitted[0]
			_, key := fss[name]
			if !key {
				fss[name] = &FSMetrics{0, 0}
			}

			fss[name].size = GetMB(splitted[1])
			fss[name].used = GetMB(splitted[2])
			log.Println(fss[name])
		}
	}
	return fss
}

//GetMB to calculate the correct number of bytes in MB
func GetMB(str string) float64 {

	val := 0.0
	if strings.HasSuffix(str, "G") {
		val, _ = strconv.ParseFloat(str[:len(str)-1], 64)
		val *= 1024
	} else if strings.HasSuffix(str, "M") {
		val, _ = strconv.ParseFloat(str[:len(str)-1], 64)
	} else if strings.HasSuffix(str, "T") {
		val, _ = strconv.ParseFloat(str[:len(str)-1], 64)
		val *= (1024 * 1024)
	} else if strings.HasSuffix(str, "K") {
		val, _ = strconv.ParseFloat(str[:len(str)-1], 64)
		val /= 1024.0
	}
	return val
}

// FSData Execute the df -h  and return its output
func FSData() []byte {
	//cmd := exec.Command("df", "-h")
	hostname := "vc7-shared"

	hostKey, err := getHostKey(hostname)
	if err != nil {
		log.Fatal(err)
	}

	key, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: os.Getenv("USER"),
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", "vc7-shared:22", config)
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer client.Close()
	session, _ := client.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run("df -h | sed -e /Filesystem/d | awk 'BEGIN{OFS=\",\"}!/(tmpfs)|(dev)/{print $1,$2,$3}'")

	return stdoutBuf.Bytes()
}

/*
 * Implement the Prometheus Collector interface and feed the
 * Slurm queue metrics into it.
 * https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector
 */
//NewFSCollector
func NewFSCollector() *FSCollector {
	labels := []string{"name"}
	return &FSCollector{
		used: prometheus.NewDesc("slurm_fs_used", "used up space in files system partitions", labels, nil),
		size: prometheus.NewDesc("slurm_fs_total", "Total space in files system partitions", labels, nil),
	}
}

//FSCollector func
type FSCollector struct {
	used *prometheus.Desc
	size *prometheus.Desc
}

//Describe func
func (fsc *FSCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- fsc.used
	ch <- fsc.size
}

//Collect func
func (fsc *FSCollector) Collect(ch chan<- prometheus.Metric) {
	fsm := FSGetMetrics()
	for fs := range fsm {
		if fsm[fs].used >= 0 {
			ch <- prometheus.MustNewConstMetric(fsc.used, prometheus.GaugeValue,
				fsm[fs].used, fs)
		}
		if fsm[fs].size >= 0 {
			ch <- prometheus.MustNewConstMetric(fsc.size, prometheus.GaugeValue,
				fsm[fs].size, fs)
		}
	}

}
