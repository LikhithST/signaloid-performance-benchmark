package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type TaskResult struct {
	Stats          map[string]interface{} `json:"Stats"`
	Output         string                 `json:"output"`
	Uxhw           bool                   `json:"uxhw"`
	IterationValue interface{}            `json:"iteration_value"`
}

func formatInt(n int64) string {
	in := strconv.FormatInt(n, 10)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits-- // First character is the - sign
	}
	numOfCommas := (numOfDigits - 1) / 3
	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}
	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

func formatFloatCommas(f float64, prec int) string {
	s := fmt.Sprintf(fmt.Sprintf("%%.%df", prec), f)
	parts := strings.Split(s, ".")
	intPart, _ := strconv.ParseInt(parts[0], 10, 64)
	formattedInt := formatInt(intPart)
	if len(parts) > 1 {
		return formattedInt + "." + parts[1]
	}
	return formattedInt
}

func parseOutput(out string) float64 {
	out = strings.TrimSpace(out)
	idx := strings.Index(out, "Ux")
	if idx != -1 {
		out = out[:idx]
	}
	val, _ := strconv.ParseFloat(out, 64)
	return val
}

type mcRow struct {
	Iter     float64
	Instr    float64
	ProcTime float64
	ExecTime float64
	Result   float64
}

func main() {
	// --- 1. FIND LATEST PLOTS DIRECTORY ---
	plotsDir := "plots"
	entries, err := os.ReadDir(plotsDir)
	if err != nil {
		log.Fatalf("Failed to read plots directory: %v", err)
	}

	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
		}
	}

	if len(subdirs) == 0 {
		fmt.Println("No plot directories found.")
		return
	}

	// Sort alphabetically; since the naming is YYYYMMDD_HHMMSS, the last one is the latest
	sort.Strings(subdirs)
	latestDir := subdirs[len(subdirs)-1]
	fmt.Printf("Latest plots directory found: %s\n", latestDir)

	// --- 2. FIND LATEST HISTORY DIRECTORY & READ JSON ---
	historyDir := "history"
	hEntries, err := os.ReadDir(historyDir)
	if err != nil {
		log.Fatalf("Failed to read history directory: %v", err)
	}
	var hSubdirs []string
	for _, entry := range hEntries {
		if entry.IsDir() {
			hSubdirs = append(hSubdirs, entry.Name())
		}
	}
	if len(hSubdirs) == 0 {
		log.Fatalf("No history directories found.")
	}
	sort.Strings(hSubdirs)
	latestHistory := hSubdirs[len(hSubdirs)-1]

	finalJSONPath := filepath.Join(historyDir, latestHistory, "final-outputs.json")
	content, err := os.ReadFile(finalJSONPath)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", finalJSONPath, err)
	}

	var results []TaskResult
	if err := json.Unmarshal(content, &results); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// --- 3. GENERATE DYNAMIC TABLE ---
	var (
		uxhwCount    int
		uxhwInstr    float64
		uxhwProcTime float64
		uxhwExecTime float64
		uxhwResult   float64
	)
	var mcRows []mcRow

	for _, r := range results {
		var instr, pTime, execTime float64
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
		outVal := parseOutput(r.Output)

		if r.Uxhw {
			uxhwCount++
			uxhwInstr += instr
			uxhwProcTime += pTime
			uxhwExecTime += execTime
			uxhwResult += outVal
		} else {
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
			mcRows = append(mcRows, mcRow{iter, instr, pTime, execTime, outVal})
		}
	}

	sort.Slice(mcRows, func(i, j int) bool { return mcRows[i].Iter < mcRows[j].Iter })

	tableStr := "<!-- TABLE_START -->\n| Method | Iterations | Dynamic Instructions | Processor Time (s) | Execution Time (ms) | Result |\n| :--- | :--- | :--- | :--- | :--- | :--- |\n"
	for _, r := range mcRows {
		tableStr += fmt.Sprintf("| Monte Carlo | %s | %s | %.4f | %s | %s |\n", formatInt(int64(r.Iter)), formatInt(int64(r.Instr)), r.ProcTime, formatInt(int64(r.ExecTime)), formatFloatCommas(r.Result, 2))
	}
	if uxhwCount > 0 {
		avgInstr := uxhwInstr / float64(uxhwCount)
		avgProc := uxhwProcTime / float64(uxhwCount)
		avgExec := uxhwExecTime / float64(uxhwCount)
		avgRes := uxhwResult / float64(uxhwCount)
		tableStr += fmt.Sprintf("| **UxHw (Avg)** | **N/A** | **~%s** | **~%.4f** | **~%s** | **%s** |\n", formatInt(int64(avgInstr)), avgProc, formatInt(int64(avgExec)), formatFloatCommas(avgRes, 2))
	}
	tableStr += "<!-- TABLE_END -->"

	// --- 4. UPDATE README CONTENT ---
	readmePath := "README.md"
	readmeBytes, err := os.ReadFile(readmePath)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", readmePath, err)
	}
	readmeContent := string(readmeBytes)

	// Define the markdown image block to inject
	newPlots := fmt.Sprintf(`<!-- PLOTS_START -->
![Dynamic Instructions](plots/%[1]s/instrChart.png)

![Processor Time](plots/%[1]s/timeChart.png)

![Execution Time](plots/%[1]s/execTimeChart.png)

![Result Distribution](plots/%[1]s/distChart.png)
<!-- PLOTS_END -->`, latestDir)

	// Use regex to replace everything between the markers
	rePlots := regexp.MustCompile(`(?s)<!-- PLOTS_START -->.*?<!-- PLOTS_END -->`)
	if !rePlots.MatchString(readmeContent) {
		log.Fatalf("Could not find <!-- PLOTS_START --> and <!-- PLOTS_END --> markers in %s", readmePath)
	}
	updatedReadme := rePlots.ReplaceAllString(readmeContent, newPlots)

	reTable := regexp.MustCompile(`(?s)<!-- TABLE_START -->.*?<!-- TABLE_END -->`)
	if !reTable.MatchString(updatedReadme) {
		log.Fatalf("Could not find <!-- TABLE_START --> and <!-- TABLE_END --> markers in %s", readmePath)
	}
	updatedReadme = reTable.ReplaceAllString(updatedReadme, tableStr)

	if err := os.WriteFile(readmePath, []byte(updatedReadme), 0644); err != nil {
		log.Fatalf("Failed to write updated %s: %v", readmePath, err)
	}
	fmt.Println("Successfully updated README.md with the latest plots and performance table!")
}
