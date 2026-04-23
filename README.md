# About Me
Software Engineer with 4 years of experience in the IT field at various companies, including Robert Bosch GmbH, specializing in cloud-native systems and system reliability. Experienced in developing microservices (Go, JavaScript, Python) and managing the full application lifecycle using Docker and Kubernetes across AWS and Azure, implementing CI/CD pipelines and enhancing system traceability through observability tools, such as OpenTelemetry and Prometheus.

# Table of Contents
1. [About Me](#about-me)
2. [Performance Benchmarking](#performance-benchmarking-monte-carlo-vs-signaloid-uxhw)
    - [Performance Comparison Table](#performance-comparison-table)
    - [Signaloid Execution Plots](#signaloid-execution-plots)
    - [Key Findings](#key-findings)
3. [Signaloid API Demonstration Scripts](#signaloid-api-demonstration-scripts)
4. [signaloid_pipe Workflow](#signaloid_pipe-workflow)
5. [Files Included](#files-included)
6. [Prerequisites](#prerequisites)
7. [Usage](#usage)
8. [Automated CI/CD Pipeline](#automated-cicd-pipeline)



# Performance Benchmarking: Monte Carlo vs. Signaloid UxHw

## Performance Comparison Table

<!-- TABLE_START -->
| Method | Iterations | Dynamic Instructions | Processor Time (s) | Execution Time (ms) | Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Monte Carlo | 10,000 | 577,131 | 0.0198 | 934 | 106,000.00 |
| Monte Carlo | 1,000,000 | 51,067,334 | 0.5649 | 1,394 | 106,000.00 |
| Monte Carlo | 10,000,000 | 510,069,396 | 6.3186 | 7,127 | 105,999.67 |
| **UxHw (Avg)** | **N/A** | **~2,510,000** | **~0.059** | **~1,432** | **106,128.00** |
<!-- TABLE_END -->

## Signaloid Execution Plots

The performance metrics gathered by the automation scripts are visualized below:

The generated plots include:
1. **`instrChart.png`**: A line graph comparing Dynamic Instructions against the number of mathematical iterations for both models.
2. **`processorTimeChart.png`**: A line graph comparing the Processor Execution Time (in seconds) between standard Monte Carlo and Signaloid UxHw.
3. **`execTimeChart.png`**: A line graph comparing the total Execution Time (in milliseconds) required to complete the tasks.
4. **`distChart.png`**: A clustered histogram showing the probability density distribution of the portfolio value calculation outputs.

<!-- PLOTS_START -->
![Dynamic Instructions](plots/20260423_212241/instrChart.png)

![Processor Time](plots/20260423_212241/timeChart.png)

![Execution Time](plots/20260423_212241/execTimeChart.png)

![Result Distribution](plots/20260423_212241/distChart.png)
<!-- PLOTS_END -->

## Key Findings

1. **Computational Cost:** The Monte Carlo method exhibits linear growth in `ProcessorTime` as iteration count ($N$) increases, adhering to $O(N)$ complexity.
2. **Deterministic Efficiency:** The Signaloid UxHw approach decouples computational complexity from iteration count, enabling constant-time $O(1)$ risk analysis.
3. **Precision:** Monte Carlo results represent statistical point estimates that converge slowly, whereas Signaloid provides the analytical distribution, offering higher fidelity for tail-risk modeling.

---

# Signaloid API Demonstration Scripts

**Repository:** [https://github.com/LikhithST/signaloid-demonstration](https://github.com/LikhithST/signaloid-demonstration)

This repository contains shell scripts demonstrating how to interact with the [Signaloid Cloud API](https://signaloid.io/) to build and execute C programs. The scripts showcase two different approaches to calculating a portfolio's future value given an uncertain daily return: one using Signaloid's Uncertainty API (`uxhw.h`), and another using a traditional Monte Carlo simulation approach.


---

## signaloid_pipe Workflow

Both scripts follow the same automated pipeline via the Signaloid API:
1. **Submit Build**: Uploads the embedded C code payload to be compiled.
2. **Poll Build Status**: Waits until the build is `Completed`.
3. **Submit Task**: Executes the compiled binary on a specified Signaloid Core (`CoreID`).
4. **Poll Task Status**: Waits until the execution is `Completed`.
5. **Retrieve Execution Statistics**: Fetches and displays statistics about the task execution, such as duration.
6. **Retrieve Output**: Fetches and displays the standard output (stdout) from the executed task.

**Example Script:**
```bash
#!/bin/bash

# Configuration
# Replace with your actual Signaloid API Key
# API_KEY="YOUR_SIGNALOID_API_KEY"
CORE_ID="cor_b21e4de9927158c1a5b603c2affb8a09" # using C0-S+
BASE_URL="https://api.signaloid.io"

# Helper function to extract JSON values using python3
parse_json() {
    python3 -c "import sys, json; print(json.load(sys.stdin)['$1'])"
}

echo "--- 1. Submitting Build ---"
BUILD_RESPONSE=$(curl -s -X POST "$BASE_URL/sourcecode/builds" \
    -H "Authorization: $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
        \"Code\": \"<Code_with_or_without_uxhw.h>\",
        \"Language\": \"C\",
        \"CoreID\": \"$CORE_ID\"
    }")

BUILD_ID=$(echo "$BUILD_RESPONSE" | parse_json "BuildID")
echo "Build ID: $BUILD_ID"

# Poll Build Status
echo "--- 2. Polling Build Status ---"
while true; do
    BUILD_STATUS_RESPONSE=$(curl -s -H "Authorization: $API_KEY" "$BASE_URL/builds/$BUILD_ID")
    STATUS=$(echo "$BUILD_STATUS_RESPONSE" | parse_json "Status")
    echo "Current Status: $STATUS"
    
    if [ "$STATUS" == "Completed" ]; then
        break
    elif [ "$STATUS" == "Cancelled" ] || [ "$STATUS" == "Stopped" ]; then
        echo "Build terminal state reached: $STATUS"
        exit 1
    fi
    sleep 2
done

# Execute Task
echo "--- 3. Submitting Task ---"
TASK_RESPONSE=$(curl -s -X POST "$BASE_URL/builds/$BUILD_ID/tasks" \
    -H "Authorization: $API_KEY")
TASK_ID=$(echo "$TASK_RESPONSE" | parse_json "TaskID")
echo "Task ID: $TASK_ID"

# Poll Task Status
echo "--- 4. Polling Task Status ---"
while true; do
    TASK_STATUS_RESPONSE=$(curl -s -H "Authorization: $API_KEY" "$BASE_URL/tasks/$TASK_ID")
    STATUS=$(echo "$TASK_STATUS_RESPONSE" | parse_json "Status")
    echo "Current Status: $STATUS"
    
    if [ "$STATUS" == "Completed" ]; then
        break
    elif [ "$STATUS" == "Cancelled" ] || [ "$STATUS" == "Stopped" ]; then
        echo "Task terminal state reached: $STATUS"
        exit 1
    fi
    sleep 2
done

# Fetch Execution Stats
echo "--- 5. Fetching Execution Stats ---"
# The TASK_STATUS_RESPONSE from the last poll already contains the completed task details
EXECUTION_STATS=$(echo "$TASK_STATUS_RESPONSE" | python3 -c "import sys, json; print(json.dumps(json.load(sys.stdin).get('Stats', {}), indent=2))")
echo "Execution Statistics:"
echo "$EXECUTION_STATS"

# Fetch Outputs
echo "--- 6. Retrieving Output ---"
OUTPUT_RESPONSE=$(curl -s -H "Authorization: $API_KEY" "$BASE_URL/tasks/$TASK_ID/outputs")
OUTPUT_URL=$(echo "$OUTPUT_RESPONSE" | parse_json "Stdout")

echo "Resulting Output:"
curl -s "$OUTPUT_URL"
echo ""
```

---

## Files Included

### 1. `run_signaloid_pipe_with_uxhw.sh`
This script submits a C program that leverages Signaloid's Uncertainty API (`uxhw.h`).
- Instead of looping through thousands of possibilities, it defines the daily return as an uncertain uniform distribution (`UxHwDoubleUniformDist(0.05, 0.07)`).
- Signaloid's hardware/microarchitecture propagates this uncertainty automatically through the calculation.
- The output is the entire probability distribution of the final portfolio value, calculated in a single pass without loops.

**C Code Snippet:**
```c
#include <stdio.h>
#include <uxhw.h>

int main() {
    double principal = 100000.0;

    // We define the market return as a known distribution of possibilities.
    // The hardware will propagate this uncertainty through the formula.
    double daily_return = UxHwDoubleUniformDist(0.05, 0.07);

    // One single calculation, zero loops.
    double final_value = principal * (1 + daily_return);

    // The output is the entire probability distribution of the result.
    printf("Portfolio outcome distribution: %lf\n", final_value);

    return 0;
}
```

### 2. `run_signaloid_pipe_without_uxhw.sh`
This script submits a traditional C program that relies on a standard Monte Carlo simulation.
- It uses the standard library (`rand()`) and loops 10,000 times to simulate different possible daily returns between 5% and 7%.
- The results are averaged to provide a projected average portfolio value.
- This serves as a baseline to compare against Signaloid's automated uncertainty-tracking capabilities.

**C Code Snippet:**
```c
#include <stdio.h>
#include <stdlib.h>
#include <time.h>

int main() {
    double min = 0.05;
    double max = 0.07;
    int iterations = 10000;
    double principal = 100000.0;
    double sum_results = 0;

    srand(time(NULL));

    for (int i = 0; i < iterations; i++) {
        double daily_return = min + ((double)rand() / (double)RAND_MAX) * (max - min);
        double final_value = principal * (1 + daily_return);
        sum_results += final_value;
    }

    printf("Projected Average Portfolio Value: %.2f\n", sum_results / iterations);
    return 0;
}
```

## Prerequisites

To run these scripts, you need the following installed on your system:
- `curl` (for making HTTP requests to the API)
- `python3` (used as a lightweight JSON parser in the scripts)
- A valid Signaloid API Key.

## Usage

1. Make sure the scripts are executable:
   ```bash
   chmod +x run_signaloid_pipe_with_uxhw.sh
   chmod +x run_signaloid_pipe_without_uxhw.sh
   ```

2. Set your Signaloid API Key as an environment variable. The scripts read the `$API_KEY` variable from your environment:
   ```bash
   export API_KEY="your_actual_api_key_here"
   ```

3. Run the scripts:
   ```bash
   ./run_signaloid_pipe_with_uxhw.sh
   ./run_signaloid_pipe_without_uxhw.sh
   ```

## Automated CI/CD Pipeline (GitHub Actions)

This repository also features a fully automated GitHub Actions workflow that executes both the Monte Carlo and UxHw C programs, fetches their execution statistics, generates performance plots, and commits the results back to the repository.

👉 **Read the GitHub Actions Pipeline Documentation here** for details on how to configure and trigger the automated pipeline.

---
