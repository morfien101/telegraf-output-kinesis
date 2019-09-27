package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	kinesis "github.com/morfien101/telegraf-output-kinesis/kinesishandler"
	"github.com/morfien101/telegraf-output-kinesis/metrics"
)

var (
	version = "v0.0.0"
)

// Login to AWS for Kinesis
//   - We should cache the credentials
// Lookup Kinesis Shard
//   - cache the result
// Convert json into a metric for Telegraf
// Drop any tags that are blacklisted? - processor plugin already exists for it
// Do what Kinesis output code does
//   - https://github.com/influxdata/telegraf/pull/5588

func main() {
	flagVersion := flag.Bool("v", false, "Shows the version of the plugin.")
	flagSampleConfig := flag.Bool("sample", false, "Prints a sample configuration to the StdOut.")
	flagDescription := flag.Bool("description", false, "Prints a sample configuration to the StdOut.")
	flagConfigFile := flag.String("f", "kinesis_output.toml", "The location of the plugins configuration file.")

	if *flagVersion {
		fmt.Println(version)
		return
	}

	if *flagSampleConfig {
		fmt.Println(kinesis.SampleConfig())
		return
	}

	if *flagDescription {
		fmt.Println(kinesis.Description())
		return
	}

	stdinBytes, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
	if err != nil {
		terminate("Failed to read passed in data: %s", err)
	}

	if len(stdinBytes) == 0 {
		// Noting to do
		return
	}

	metricList, err := metrics.Convert(stdinBytes)
	if err != nil {
		terminate("Failed to convert the metrics: %s", err)
	}

	ko, err := kinesis.NewKinesisOutput(*flagConfigFile)
	if err != nil {
		terminate("Failed to make Kinesis adaptor. Error: %s", err)
	}
	err = ko.Connect()
	if err != nil {
		terminate("Couldn't connect to Kinesis. Error: %s", err)
	}
	err = ko.Write(metricList)
	if err != nil {
		terminate("Failed to write to Kinesis. Error: %s", err)
	}
}

func terminate(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
