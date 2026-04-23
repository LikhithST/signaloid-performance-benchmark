package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TaskResult maps the structure inside final-outputs.json
type TaskResult struct {
	BuildID        string                 `json:"buildID"`
	TaskID         string                 `json:"taskId"`
	Stats          map[string]interface{} `json:"Stats"`
	Output         string                 `json:"output"`
	Uxhw           bool                   `json:"uxhw"`
	IterationValue interface{}            `json:"iteration_value"`
}

// PointLinear represents an X/Y coordinate for linear Chart.js line charts
type PointLinear struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PointCategory represents an X/Y coordinate with a categorical string X-axis
type PointCategory struct {
	X string  `json:"x"`
	Y float64 `json:"y"`
}

// parseOutput cleans up the string "106000.00Ux..." and extracts the float value
func parseOutput(out string) float64 {
	out = strings.TrimSpace(out)
	idx := strings.Index(out, "Ux")
	if idx != -1 {
		out = out[:idx]
	}
	val, _ := strconv.ParseFloat(out, 64)
	return val
}

// saveChart uses the QuickChart.io API to render the chart.js config to a PNG
func saveChart(configStr, filename string) {
	payload := map[string]interface{}{
		"version":         "3",
		"width":           800,
		"height":          600,
		"backgroundColor": "white",
	}
	var chartObj map[string]interface{}
	json.Unmarshal([]byte(configStr), &chartObj)
	payload["chart"] = chartObj

	body, _ := json.Marshal(payload)
	resp, err := http.Post("https://quickchart.io/chart", "application/json", bytes.NewBuffer(body))
	if err != nil || resp.StatusCode != 200 {
		log.Printf("Failed to request chart %s: %v", filename, err)
		return
	}
	defer resp.Body.Close()

	out, _ := os.Create(filename)
	defer out.Close()
	io.Copy(out, resp.Body)
	fmt.Printf("Saved plot: %s\n", filename)
}

func main() {
	content, err := os.ReadFile("final-outputs.json")
	if err != nil {
		log.Fatalf("Failed to read final-outputs.json: %v", err)
	}

	var results []TaskResult
	if err := json.Unmarshal(content, &results); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Data arrays for Line Plots (Stats vs Iterations)
	var falseInstr []PointLinear
	var trueInstr []PointCategory
	var falseTime []PointLinear
	var trueTime []PointCategory
	var falseExecTime []PointLinear
	var trueExecTime []PointCategory

	// Data arrays for Histogram (Distribution of Result Value)
	var falseResults []float64
	var trueResults []float64

	uxhwTrial := 1
	for _, r := range results {
		// 1. Extract Iteration Value
		var iter float64 = 1
		if r.IterationValue != nil {
			switch v := r.IterationValue.(type) {
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					iter = parsed
				}
			case float64:
				iter = v
			}
		}

		// 2. Extract Stats
		var instr float64
		var pTime float64
		var execTime float64
		if r.Stats != nil {
			if i, ok := r.Stats["DynamicInstructions"].(float64); ok {
				instr = i
			}
			if t, ok := r.Stats["ProcessorTime"].(float64); ok {
				pTime = t
			}
			if e, ok := r.Stats["ExecutionTimeInMilliseconds"].(float64); ok {
				execTime = e
			}
		}

		// 3. Extract Result String into Float
		outVal := parseOutput(r.Output)

		// Sort into UxHw true or false buckets
		if r.Uxhw {
			trialLabel := fmt.Sprintf("trial-%d", uxhwTrial)
			trueInstr = append(trueInstr, PointCategory{X: trialLabel, Y: instr})
			trueTime = append(trueTime, PointCategory{X: trialLabel, Y: pTime})
			trueExecTime = append(trueExecTime, PointCategory{X: trialLabel, Y: execTime})
			trueResults = append(trueResults, outVal)
			uxhwTrial++
		} else {
			falseInstr = append(falseInstr, PointLinear{X: iter, Y: instr})
			falseTime = append(falseTime, PointLinear{X: iter, Y: pTime})
			falseExecTime = append(falseExecTime, PointLinear{X: iter, Y: execTime})
			falseResults = append(falseResults, outVal)
		}
	}

	// Sort line points by X (iteration value) to ensure lines draw cleanly left-to-right
	sortLinearPoints := func(pts []PointLinear) {
		sort.Slice(pts, func(i, j int) bool { return pts[i].X < pts[j].X })
	}
	sortLinearPoints(falseInstr)
	sortLinearPoints(falseTime)
	sortLinearPoints(falseExecTime)

	// ----------------------------------------------------
	// Prepare Histogram Buckets for the Outputs
	// ----------------------------------------------------
	minVal, maxVal := math.MaxFloat64, -math.MaxFloat64
	for _, v := range append(falseResults, trueResults...) {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	// Add slight padding if min == max (e.g. all answers are exactly identical)
	if minVal == maxVal {
		minVal -= 1
		maxVal += 1
	} else if minVal == math.MaxFloat64 {
		minVal, maxVal = 0, 100 // fallback
	}

	numBuckets := 10
	bucketSize := (maxVal - minVal) / float64(numBuckets)
	labels := make([]string, numBuckets)
	falseDist := make([]int, numBuckets)
	trueDist := make([]int, numBuckets)

	for i := 0; i < numBuckets; i++ {
		labels[i] = fmt.Sprintf("%.2f - %.2f", minVal+float64(i)*bucketSize, minVal+float64(i+1)*bucketSize)
	}

	getBucket := func(val float64) int {
		b := int((val - minVal) / bucketSize)
		if b >= numBuckets {
			b = numBuckets - 1
		}
		if b < 0 {
			b = 0
		}
		return b
	}

	for _, v := range falseResults {
		falseDist[getBucket(v)]++
	}
	for _, v := range trueResults {
		trueDist[getBucket(v)]++
	}

	// ----------------------------------------------------
	// Serialize to JSON arrays for Javascript injection
	// ----------------------------------------------------
	falseInstrJSON, _ := json.Marshal(falseInstr)
	trueInstrJSON, _ := json.Marshal(trueInstr)
	falseTimeJSON, _ := json.Marshal(falseTime)
	trueTimeJSON, _ := json.Marshal(trueTime)
	falseExecTimeJSON, _ := json.Marshal(falseExecTime)
	trueExecTimeJSON, _ := json.Marshal(trueExecTime)
	labelsJSON, _ := json.Marshal(labels)
	falseDistJSON, _ := json.Marshal(falseDist)
	trueDistJSON, _ := json.Marshal(trueDist)

	dateTime := time.Now().Format("20060102_150405")
	targetDir := filepath.Join("plots", dateTime)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Fatalf("Failed to create target directory: %v", err)
	}
	fmt.Printf("Saving plots to directory: %s\n", targetDir)

	instrConfig := fmt.Sprintf(`{
		"type": "line",
		"data": { "datasets": [
			{ "label": "Dynamic Instructions (Monte Carlo)", "data": %s, "borderColor": "red", "backgroundColor": "red", "fill": false, "tension": 0.1, "xAxisID": "x", "yAxisID": "y" },
			{ "label": "Dynamic Instructions (Signaloid API)", "data": %s, "borderColor": "blue", "backgroundColor": "blue", "fill": false, "tension": 0.1, "xAxisID": "x2", "yAxisID": "y2" }
		]},
		"options": { "plugins": { "title": { "display": true, "text": "Dynamic Instructions vs Iterations" } }, "scales": { "x": { "type": "linear", "position": "bottom", "title": { "display": true, "text": "Iteration Value" } }, "x2": { "type": "category", "position": "top", "title": { "display": true, "text": "UxHw Trials" }, "grid": { "drawOnChartArea": false } }, "y": { "type": "linear", "stack": "y-axes", "stackWeight": 1, "position": "left", "title": { "display": true, "text": "Monte Carlo" } }, "y2": { "type": "linear", "stack": "y-axes", "stackWeight": 1, "position": "left", "title": { "display": true, "text": "Signaloid API" } } } }
	}`, string(falseInstrJSON), string(trueInstrJSON))

	timeConfig := fmt.Sprintf(`{
		"type": "line",
		"data": { "datasets": [
			{ "label": "Processor Time (s) (Monte Carlo)", "data": %s, "borderColor": "orange", "backgroundColor": "orange", "fill": false, "tension": 0.1, "xAxisID": "x", "yAxisID": "y" },
			{ "label": "Processor Time (s) (Signaloid API)", "data": %s, "borderColor": "green", "backgroundColor": "green", "fill": false, "tension": 0.1, "xAxisID": "x2", "yAxisID": "y2" }
		]},
		"options": { "plugins": { "title": { "display": true, "text": "Processor Time vs Iterations" } }, "scales": { "x": { "type": "linear", "position": "bottom", "title": { "display": true, "text": "Iteration Value" } }, "x2": { "type": "category", "position": "top", "title": { "display": true, "text": "UxHw Trials" }, "grid": { "drawOnChartArea": false } }, "y": { "type": "linear", "stack": "y-axes", "stackWeight": 1, "position": "left", "title": { "display": true, "text": "Monte Carlo (s)" } }, "y2": { "type": "linear", "stack": "y-axes", "stackWeight": 1, "position": "left", "title": { "display": true, "text": "Signaloid API (s)" } } } }
	}`, string(falseTimeJSON), string(trueTimeJSON))

	execTimeConfig := fmt.Sprintf(`{
		"type": "line",
		"data": { "datasets": [
			{ "label": "Execution Time (ms) (Monte Carlo)", "data": %s, "borderColor": "purple", "backgroundColor": "purple", "fill": false, "tension": 0.1, "xAxisID": "x", "yAxisID": "y" },
			{ "label": "Execution Time (ms) (Signaloid API)", "data": %s, "borderColor": "teal", "backgroundColor": "teal", "fill": false, "tension": 0.1, "xAxisID": "x2", "yAxisID": "y2" }
		]},
		"options": { "plugins": { "title": { "display": true, "text": "Execution Time vs Iterations" } }, "scales": { "x": { "type": "linear", "position": "bottom", "title": { "display": true, "text": "Iteration Value" } }, "x2": { "type": "category", "position": "top", "title": { "display": true, "text": "UxHw Trials" }, "grid": { "drawOnChartArea": false } }, "y": { "type": "linear", "stack": "y-axes", "stackWeight": 1, "position": "left", "title": { "display": true, "text": "Monte Carlo (ms)" } }, "y2": { "type": "linear", "stack": "y-axes", "stackWeight": 1, "position": "left", "title": { "display": true, "text": "Signaloid API (ms)" } } } }
	}`, string(falseExecTimeJSON), string(trueExecTimeJSON))

	distConfig := fmt.Sprintf(`{
		"type": "bar",
		"data": {
			"labels": %s,
			"datasets": [
				{ "label": "Monte Carlo (UxHw: false)", "data": %s, "backgroundColor": "rgba(255, 99, 132, 0.6)" },
				{ "label": "Signaloid API (UxHw: true)", "data": %s, "backgroundColor": "rgba(54, 162, 235, 0.6)" }
			]
		},
		"options": { "plugins": { "title": { "display": true, "text": "Result Distribution (Histogram)" } }, "scales": { "x": { "title": { "display": true, "text": "Result Value Buckets" } }, "y": { "title": { "display": true, "text": "Frequency" } } } }
	}`, string(labelsJSON), string(falseDistJSON), string(trueDistJSON))

	saveChart(instrConfig, filepath.Join(targetDir, "instrChart.png"))
	saveChart(timeConfig, filepath.Join(targetDir, "timeChart.png"))
	saveChart(execTimeConfig, filepath.Join(targetDir, "execTimeChart.png"))
	saveChart(distConfig, filepath.Join(targetDir, "distChart.png"))

	fmt.Println("All plots successfully generated!")
}
