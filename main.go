package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
)

// const numberOfExperiments = 4
const parallelExecutions = 20
const repetitionsPerExperiments = 11
const scenarios = 5

var currentPath string

func init() {
	setupCurrentPath()
}

func setupCurrentPath() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current frame")
	}
	currentPath = path.Dir(filename)
}

func main() {
	for scenario := 1; scenario <= scenarios; scenario++ {
		h2FilePath := path.Join(currentPath, "files/h2", fmt.Sprintf("metrics-%d.csv", scenario))
		reader, file := getCsvReader(h2FilePath)
		defer file.Close()

		h2Averages := extractMetrics(reader)
		// fmt.Println(*h2Averages)

		h3FilePath := path.Join(currentPath, "files/h3", fmt.Sprintf("metrics-%d.csv", scenario))
		reader, file = getCsvReader(h3FilePath)
		defer file.Close()

		h3Averages := extractMetrics(reader)
		// fmt.Println(*h3Averages)

		saveAverages(scenario, h2Averages, h3Averages, true)
	}
}

func extractMetrics(reader *csv.Reader) *[]float64 {
	metrics := make([]float64, parallelExecutions*repetitionsPerExperiments)
	metricsMin := make([]float64, repetitionsPerExperiments)
	// metricsAverages := make([]float64, repetitionsPerExperiments)
	metricsAverages := []float64{}
	metricsMax := make([]float64, repetitionsPerExperiments)

	experimentCounter := 0

	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("Error while reading file", err)
		}

		recordSize := len(record)
		totalTime := record[recordSize-1 : recordSize][0]

		totalTimeAsInt := fieldToFloat64(totalTime, true)

		metrics[experimentCounter] = totalTimeAsInt

		if experimentCounter%parallelExecutions == 0 {
			position := experimentCounter / parallelExecutions
			metricsMin[position] = totalTimeAsInt
		}

		if experimentCounter%parallelExecutions == parallelExecutions-1 {
			// average := calculateAverage(metricsAverages)
			// deviation := calculateStandardDeviation(metricsAverages, average)

			// fmt.Printf("%v : %v -> %v\n\n", average, deviation, metricsAverages)

			startPosition := experimentCounter - (parallelExecutions - 1)
			endPosition := experimentCounter + 1
			metrics := metrics[startPosition:endPosition]

			average := calculateAverage(metrics)

			position := startPosition / parallelExecutions

			// metricsAverages[position] = average

			// myAverage := *metricsAverages
			metricsAverages = append(metricsAverages, average)

			metricsMax[position] = totalTimeAsInt
		}

		if experimentCounter == parallelExecutions*repetitionsPerExperiments-1 {
			// average := calculateAverage(metricsPerExperiment)
			// deviation := calculateStandardDeviation(metricsPerExperiment, average)

			// fmt.Printf("%v : %v -> %v\n\n", average, deviation, metricsPerExperiment)

			// minAverage := calculateAverage(metricsMin)
			// minDeviation := calculateStandardDeviation(metricsMin, minAverage)

			// fmt.Printf("%v : %v -> %v\n\n", minAverage, minDeviation, metricsMin)

			// maxAverage := calculateAverage(metricsMax)
			// maxDeviation := calculateStandardDeviation(metricsMax, maxAverage)

			// fmt.Printf("%v : %v -> %v\n\n", maxAverage, maxDeviation, metricsMax)

			// fmt.Printf("Met: %v\n\n", metricsPerExperiment)
			// fmt.Printf("Min: %v\n", metricsMin)
			// fmt.Printf("Avg: %v\n", metricsAverages)
			// fmt.Printf("Max: %v\n", metricsMax)
			// fmt.Println()

			experimentCounter = 0
		} else {
			experimentCounter++
		}
	}

	return &metricsAverages
}

func getCsvReader(csvPath string) (*csv.Reader, *os.File) {
	file, err := os.Open(csvPath)

	if err != nil {
		log.Fatal("Error while opening file: ", err)
	}

	reader := csv.NewReader(file)
	reader.Comma = ','

	if _, err := reader.Read(); err != nil {
		log.Fatal("Error while reading file", err)
	}

	return reader, file
}

func fieldToFloat64(value string, outputInMilliseconds bool) float64 {
	isTimeInSeconds := !strings.Contains(value, "ms") && strings.Contains(value, "s")
	isTimeInMinutes := !strings.Contains(value, "ms") && strings.Contains(value, "m")

	value = strings.Replace(value, "m", "", -1)
	value = strings.Replace(value, "s", "", -1)

	valueAsFloat, err := strconv.ParseFloat(value, 64)

	if err != nil {
		log.Fatal("Error while converting string to float", err)
	}

	if outputInMilliseconds && isTimeInSeconds {
		valueAsFloat = valueAsFloat * 1000
	} else if outputInMilliseconds && isTimeInMinutes {
		valueAsFloat = valueAsFloat * 60 / 1000
	}

	return valueAsFloat
}

func calculateAverage(values []float64) float64 {
	var sum float64

	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

func calculateStandardDeviation(values []float64, average float64) float64 {
	var sum float64
	countElements := len(values)

	for _, value := range values {
		sum += math.Pow(value-average, 2)
	}

	variance := sum / float64(countElements)

	return math.Sqrt(variance)
}

func saveAverages(scenario int, h2Averages *[]float64, h3Averages *[]float64, ignoreFirstMetric bool) {
	var data [][]string
	averagesCount := 1

	for i, h2Average := range *h2Averages {
		h3Average := (*h3Averages)[i]
		row := []string{fmt.Sprintf("%f", h2Average), fmt.Sprintf("%f", h3Average)}
		data = append(data, row)

		if averagesCount%repetitionsPerExperiments == 0 {
			fmt.Println(averagesCount)
			experiment := averagesCount / repetitionsPerExperiments

			csvFileName := fmt.Sprintf("c%d-p%d-averages.csv", scenario, experiment)
			csvFilePath := path.Join(currentPath, "files/averages")
			csvFile, err := createFile(csvFilePath, csvFileName)
			defer csvFile.Close()

			if err != nil {
				log.Fatalf("Falha ao criar arquivo '%s'\n%s", csvFilePath, err)
			}

			if ignoreFirstMetric {
				data = data[1:]
			}

			writeToCsv(&data, csvFile)
			data = nil
		}

		averagesCount++
	}
}

func createFile(pathFile string, nameFile string) (*os.File, error) {
	if _, err := os.Stat(pathFile); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(pathFile, os.ModePerm)
		if err != nil {
			log.Fatalf("Falha ao criar pasta '%s'\n%s", pathFile, err)
		}
	}

	file, err := os.Create(path.Join(pathFile, nameFile))
	return file, err
}

func writeToCsv(data *[][]string, csvFile *os.File) {
	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()
	csvWriter.Write([]string{"H2", "H3"})
	csvWriter.WriteAll(*data)
}
