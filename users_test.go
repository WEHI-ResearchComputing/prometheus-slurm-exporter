package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestParseUsersMetrics(t *testing.T) {
	// Read the input data from a file
	file, err := os.Open("test_data/users.txt")
	if err != nil {
		t.Fatalf("Can not open test data: %v", err)
	}
	data, err := ioutil.ReadAll(file)
	//t.Error(data)
	metrics := ParseUsersMetrics(data)
	for k, v := range metrics {
		//t.Error(k, v)
		t.Log(k, v)
	}
	//t.Logf("%+v", ParseNodesInfoMetrics(data))

}
