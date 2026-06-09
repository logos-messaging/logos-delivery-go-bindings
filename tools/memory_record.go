package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/logos-messaging/logos-delivery-go-bindings/utils"
)

var (
	testName  string
	iteration int
	phase     string
)

func main() {
	flag.StringVar(&testName, "testName", "FullTestSuite", "Name of the test ")
	flag.IntVar(&iteration, "iteration", 0, "Iteration number")
	flag.StringVar(&phase, "phase", "", "'start' or 'end')")
	flag.Parse()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	heapKB := memStats.HeapAlloc / 1024

	rssKB, err := utils.GetRSSKB()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get RSS:", err)
		rssKB = 0
	}

	if err := utils.RecordMemoryMetricsCSV(testName, iteration, phase, heapKB, rssKB); err != nil {
		fmt.Fprintln(os.Stderr, "Error recording metrics:", err)
		os.Exit(1)
	}
}
