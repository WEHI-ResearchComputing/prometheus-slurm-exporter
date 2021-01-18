/* Copyright 2020 Victor Penso

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
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func UsersData() []byte {
	cmd := exec.Command("squeue", "-a", "-r", "-h", "-o %A|%u|%T|%C|%m|%r")
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

/*UserJobMetrics struct to collect number of jobs in each state
as well as memory and cpus allocated for the job
*/
type UserJobMetrics struct {
	pending       float64
	pendingQOS    float64
	pendingOthers float64
	running       float64
	suspended     float64
	runningCpus   float64
	pendingCpus   float64
	suspendedCpus float64
	pendingMem    float64
	runningMem    float64
	suspendedMem  float64
	reason        string
}

func ParseUsersMetrics(input []byte) map[string]*UserJobMetrics {
	users := make(map[string]*UserJobMetrics)
	lines := strings.Split(string(input), "\n")

	for _, line := range lines {

		if strings.Contains(line, "|") {
			user := strings.Split(line, "|")[1]
			_, key := users[user]
			if !key {
				users[user] = &UserJobMetrics{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, ""}
			}
			state := strings.Split(line, "|")[2]
			state = strings.ToLower(state)
			cpus, _ := strconv.ParseFloat(strings.Split(line, "|")[3], 64)
			m := strings.Split(line, "|")[4]
			reason := strings.Split(line, "|")[5]
			mem := 0.0

			if strings.HasSuffix(m, "G") {
				m = m[:len(m)-1]
				mem, _ = strconv.ParseFloat(m, 64)
				mem *= 1024
			} else if strings.HasSuffix(m, "M") {
				m = m[:len(m)-1]
				mem, _ = strconv.ParseFloat(m, 64)
			}
			pending := regexp.MustCompile(`^pending`)
			running := regexp.MustCompile(`^running`)
			suspended := regexp.MustCompile(`^suspended`)
			switch {
			case pending.MatchString(state) == true:
				users[user].pending++
				users[user].pendingCpus += cpus
				users[user].pendingMem += mem
				users[user].reason = reason
				if strings.Contains(reason, "QOS") {
					users[user].pendingQOS++
				} else {
					users[user].pendingOthers++
				}

			case running.MatchString(state) == true:
				users[user].running++
				users[user].runningCpus += cpus
				users[user].runningMem += mem
			case suspended.MatchString(state) == true:
				users[user].suspended++
				users[user].suspendedCpus += cpus
				users[user].suspendedMem += mem
			}
		}
	}
	return users
}

//UsersCollector struct
type UsersCollector struct {
	pending       *prometheus.Desc
	pendingQOS    *prometheus.Desc
	pendingOthers *prometheus.Desc
	running       *prometheus.Desc
	suspended     *prometheus.Desc
	runningCpus   *prometheus.Desc
	pendingCpus   *prometheus.Desc
	suspendedCpus *prometheus.Desc
	pendingMem    *prometheus.Desc
	runningMem    *prometheus.Desc
	suspendedMem  *prometheus.Desc
}

func NewUsersCollector() *UsersCollector {
	labels := []string{"user"}
	return &UsersCollector{
		pending:       prometheus.NewDesc("slurm_user_jobs_pending", "Total Pending jobs for user", labels, nil),
		pendingQOS:    prometheus.NewDesc("slurm_user_jobs_pendingQOS", "Pending jobs for user due to QOS", labels, nil),
		pendingOthers: prometheus.NewDesc("slurm_user_jobs_pendingOthers", "Pending jobs for user due to other reasons", labels, nil),
		running:       prometheus.NewDesc("slurm_user_jobs_running", "Running jobs for user", labels, nil),
		suspended:     prometheus.NewDesc("slurm_user_jobs_suspended", "Suspended jobs for user", labels, nil),
		runningCpus:   prometheus.NewDesc("slurm_user_cpus_running", "Running cpus for user", labels, nil),
		pendingCpus:   prometheus.NewDesc("slurm_user_cpus_pending", "Pending cpus for user", labels, nil),
		suspendedCpus: prometheus.NewDesc("slurm_user_cpus_suspended", "Suspended cpus for user", labels, nil),
		runningMem:    prometheus.NewDesc("slurm_user_mem_running", "Running memory (MB) for user", labels, nil),
		pendingMem:    prometheus.NewDesc("slurm_user_mem_pending", "Pending memory (MB) for user", labels, nil),
		suspendedMem:  prometheus.NewDesc("slurm_user_mem_suspended", "Suspended memory (MB) for user", labels, nil),
	}
}

func (uc *UsersCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- uc.pending
	ch <- uc.pendingQOS
	ch <- uc.pendingOthers
	ch <- uc.running
	ch <- uc.suspended
	ch <- uc.pendingCpus
	ch <- uc.runningCpus
	ch <- uc.suspendedCpus
	ch <- uc.pendingMem
	ch <- uc.runningMem
	ch <- uc.suspendedMem
}

func (uc *UsersCollector) Collect(ch chan<- prometheus.Metric) {
	um := ParseUsersMetrics(UsersData())
	for u := range um {
		if um[u].pending > 0 {
			ch <- prometheus.MustNewConstMetric(uc.pending, prometheus.GaugeValue, um[u].pending, u)
		}
		if um[u].pendingQOS > 0 {
			ch <- prometheus.MustNewConstMetric(uc.pendingQOS, prometheus.GaugeValue, um[u].pendingQOS, u)
		}
		if um[u].pendingQOS > 0 {
			ch <- prometheus.MustNewConstMetric(uc.pendingOthers, prometheus.GaugeValue, um[u].pendingQOS, u)
		}
		if um[u].running > 0 {
			ch <- prometheus.MustNewConstMetric(uc.running, prometheus.GaugeValue, um[u].running, u)
		}
		if um[u].suspended > 0 {
			ch <- prometheus.MustNewConstMetric(uc.suspended, prometheus.GaugeValue, um[u].suspended, u)
		}
		if um[u].pendingCpus > 0 {
			ch <- prometheus.MustNewConstMetric(uc.pendingCpus, prometheus.GaugeValue, um[u].pendingCpus, u)
		}
		if um[u].runningCpus > 0 {
			ch <- prometheus.MustNewConstMetric(uc.runningCpus, prometheus.GaugeValue, um[u].runningCpus, u)
		}
		if um[u].suspendedCpus > 0 {
			ch <- prometheus.MustNewConstMetric(uc.suspendedCpus, prometheus.GaugeValue, um[u].suspendedCpus, u)
		}
		if um[u].pendingMem > 0 {
			ch <- prometheus.MustNewConstMetric(uc.pendingMem, prometheus.GaugeValue, um[u].pendingMem, u)
		}
		if um[u].runningMem > 0 {
			ch <- prometheus.MustNewConstMetric(uc.runningMem, prometheus.GaugeValue, um[u].runningMem, u)
		}
		if um[u].suspendedMem > 0 {
			ch <- prometheus.MustNewConstMetric(uc.suspendedMem, prometheus.GaugeValue, um[u].suspendedMem, u)
		}
	}
}
