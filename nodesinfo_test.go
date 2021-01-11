package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestParseNodesInfoMetrics(t *testing.T) {
	// Read the input data from a file
	file, err := os.Open("test_data/nodeinfo.txt")
	if err != nil {
		t.Fatalf("Can not open test data: %v", err)
	}
	data, err := ioutil.ReadAll(file)
	t.Logf("%+v", ParseQueueMetrics(data))
}
