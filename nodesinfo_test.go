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
	//t.Error(data)
	metrics := ParseNodesInfoMetrics(data)
	for k, v := range metrics {
		//t.Error(k, v)
		t.Log(k, v)
	}
	//t.Logf("%+v", ParseNodesInfoMetrics(data))

}

func TestParseNodesDataMetrics(t *testing.T) {
	// Read the input data from a file
	file, err := os.Open("test_data/nodeinfo2.txt")
	if err != nil {
		t.Fatalf("Can not open test data: %v", err)
	}
	data, err := ioutil.ReadAll(file)
	//t.Error(data)
	metrics := ParseNodesDataMetrics(data)
	//.Error(metrics)
	for k, v := range metrics {
		t.Error(k, v)
		t.Log(k, v)
	}
	//t.Logf("%+v", ParseNodesInfoMetrics(data))

}

/*func TestNodesInfoGetMetrics(t *testing.T) {
	t.Logf("%+v", NodesInfoGetMetrics())
}*/
