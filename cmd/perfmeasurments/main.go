/*
 * Copyright (C) 2025 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Options struct {
	srcFolder               string
	tgtFolder               string
	timeBetweenMeasurements time.Duration
	timeToRun               time.Duration
}

type metricsStruct struct {
	BlockIO string `json:"BlockIO"`
	CPUPerc string `json:"CPUPerc"`
	Container string `json:"Container"`
	ID string `json:"ID"`
	MemPerc string `json:"MemPerc"`
	MemUsage string `json:"MemUsage"`
	Name string `json:"Name"`
	NetIO string `json:"NetIO"`
	PIDs string `json:"PIDs"`
	timeFromStart	float64
	FileIn int64 `json:"FileIn"`
	FileOut int64 `json:"FileOut"`
}

var opts Options

const (
	defaultSrcDir       = "perf_inputs/original_collector"
	defaultTgtDir       = "/tmp/perfmeasurements"
	defaultTick         = 5 * time.Second
	defaultTimeToRun    = 60 * time.Second
	definitionExt       = ".yaml"
	resultExt           = ".csv"
	resultsFolderPrefix = "perf_"
)

// rootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "perfmeasurements",
	Short: "Run performance measurements on specified config files",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func initFlags() {
	rootCmd.PersistentFlags().StringVar(&opts.srcFolder, "srcFolder", defaultSrcDir, "source folder")
	rootCmd.PersistentFlags().StringVar(&opts.tgtFolder, "tgtFolder", defaultTgtDir, "target folder")
	rootCmd.PersistentFlags().DurationVar(&opts.timeBetweenMeasurements, "timeBetweenMeasurements", defaultTick, "time between measurements")
	rootCmd.PersistentFlags().DurationVar(&opts.timeToRun, "timeToRun", defaultTimeToRun, "time to run each test")
}

func printFlags() {
	fmt.Printf("srcFolder = %s \n", opts.srcFolder)
	fmt.Printf("tgtFolder = %s \n", opts.tgtFolder)
	fmt.Printf("timeBetweenMeasurements = %v \n", opts.timeBetweenMeasurements)
	fmt.Printf("timeToRun = %v \n", opts.timeToRun)
}

func printFileNames(fileNames []string) {
	fmt.Printf("filennames of configuration files: \n")
	for _, f := range fileNames {
		fmt.Printf("%s \n", f)
	}
}

func main() {
	// Initialize flags (command line parameters)
	initFlags()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() {
	// Dump the configuration
	printFlags()
	fileNames := getYamlFileNames(opts.srcFolder, "")
	printFileNames(fileNames)
	tgtFolder := opts.tgtFolder + "/" + resultsFolderPrefix + time.Now().Format(time.RFC3339)
	err := createTargetFolder(tgtFolder)
	if err != nil {
		fmt.Printf("could not create target folder; err = %v, dirName = %s \n", err, opts.tgtFolder)
		os.Exit(1)
	}
	runMeasurements(opts.srcFolder, fileNames, tgtFolder)
}

func getYamlFileNames(rootPath string, prefix string) []string {
	var files []string

	newRootPath := filepath.Join(rootPath, prefix)
	dirEntries, err := os.ReadDir(newRootPath)
	if err != nil {
		fmt.Printf("could not read directory; err = %v, dirName = %s \n", err, rootPath)
		return nil
	}
	for _, f := range dirEntries {
		fMode := f.Type()
		fName := f.Name()
		if fMode.IsRegular() && filepath.Ext(fName) == definitionExt {
			if err != nil {
				fmt.Printf("could not obtain file path name; err = %v, fileName = %s \n", err, f.Name())
				return nil
			}
			fileName := filepath.Join(prefix, fName)
			files = append(files, fileName)
		}
		if fMode.IsDir() {
			fPath := filepath.Join(prefix, fName)
			subDirFiles := getYamlFileNames(rootPath, fPath)
			files = append(files, subDirFiles...)
		}
	}
	return files
}

func createTargetFolder(folderName string) error {
	err := os.MkdirAll(folderName, 0755)
	if err != nil {
		log.Debugf("os.MkdirAll err: %v ", err)
		return err
	}
	return nil
}

func CreateTargetFile(fileName string) (*os.File, error) {
	filePtr, err := os.Create(fileName)
	return filePtr, err
}

func runMeasurements(srcFolder string, filePaths []string, tgtFolder string) {
	for _, fPath := range filePaths {
		fmt.Printf("running measurements on %s \n", fPath)
		fullFilePath := filepath.Join(srcFolder, fPath)
		fmt.Printf("fullFilePath = %s \n", fullFilePath)
		err := restartCollector(fullFilePath)
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}

		startTime := time.Now()
		fmt.Printf("start time = %s \n", startTime.Format(time.RFC3339))
		ticker := time.NewTicker(opts.timeBetweenMeasurements)
		done := make(chan bool)

		// create results file
		fileName := filepath.Join(tgtFolder, fPath)
		// change the file extension
		fileName = fileName[:len(fileName)-len(filepath.Ext(fileName))] + resultExt
		fmt.Printf("output file name = %s \n", fileName)
		f, err := CreateTargetFile(fileName)
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		dw := bufio.NewWriter(f)
		// write the csv column headers
		l := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			"TimeFromStart","BlockIO","CPUPerc","Container","ID","MemPerc","MemUsage","Name","NetIO","PIDs","FileIn","FileOut",
		)
		fmt.Printf("%s", l)
		_, _ = dw.WriteString(l)
		dw.Flush()

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					metrics, err := collectMetrics()
					if err != nil {
						continue
					}
					currentTime := time.Now()
					timeFromStart := currentTime.Sub(startTime)
					metrics.timeFromStart = timeFromStart.Seconds()
					l := fmt.Sprintf("%f,%s,%s,%s,%s,%s,%s,%s,%s,%s,%d,%d\n",
						metrics.timeFromStart, metrics.BlockIO, metrics.CPUPerc, metrics.Container, metrics.ID,
						metrics.MemPerc, metrics.MemUsage, metrics.Name, metrics.NetIO, metrics.PIDs,
						metrics.FileIn, metrics.FileOut,
					)
					fmt.Printf("%s", l)
					_, _ = dw.WriteString(l)
					dw.Flush()
				}
			}
		}()

		go func() {
			time.Sleep(opts.timeToRun)
			ticker.Stop()
			done <- true
		}()

		<-done

		_ = f.Close()
	}
}

func restartCollector(fullFilePath string) (error){
	cmd := exec.Command("docker", "compose", "--env-file", ".env", "stop", "otel-collector")
	cmd.Dir, _ = os.Getwd()
	cmd.Run()
	cmd = exec.Command("docker", "compose", "--env-file", ".env", "rm", "--force", "otel-collector")
	cmd.Dir, _ = os.Getwd()
	cmd.Run()
	os.Setenv("OTEL_COLLECTOR_CONFIG", fullFilePath)
	cmd = exec.Command("docker", "compose", "--env-file", ".env", "create", "otel-collector")
	cmd.Dir, _ = os.Getwd()
	cmd.Run()
	cmd = exec.Command("docker", "compose", "--env-file", ".env", "start", "otel-collector")
	cmd.Dir, _ = os.Getwd()
	err := cmd.Run()
	return err
}

func collectMetrics() (metricsStruct, error) {
	// obtain otel-collector statistics
	cmd := exec.Command("docker", "stats", "otel-collector", "--no-stream", "--format", "json")
	cmd.Dir, _ = os.Getwd()
	output, err := cmd.CombinedOutput()

	var m metricsStruct
	json.Unmarshal(output, &m)

	// obtain size of data before and after sampling
	fileInInfo, err	:= os.Stat("otel-collector-in.json")
	if err == nil {
		m.FileIn = fileInInfo.Size()
	}
	fileOutInfo, err	:= os.Stat("otel-collector-out.json")
	if err == nil {
		m.FileOut = fileOutInfo.Size()
	}

	return m, err
}
