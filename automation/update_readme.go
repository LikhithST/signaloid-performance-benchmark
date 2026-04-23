package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
)

func main() {
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
	re := regexp.MustCompile(`(?s)<!-- PLOTS_START -->.*?<!-- PLOTS_END -->`)
	if !re.MatchString(readmeContent) {
		log.Fatalf("Could not find <!-- PLOTS_START --> and <!-- PLOTS_END --> markers in %s", readmePath)
	}
	updatedReadme := re.ReplaceAllString(readmeContent, newPlots)

	if err := os.WriteFile(readmePath, []byte(updatedReadme), 0644); err != nil {
		log.Fatalf("Failed to write updated %s: %v", readmePath, err)
	}
	fmt.Println("Successfully updated README.md with the latest plots!")
}
