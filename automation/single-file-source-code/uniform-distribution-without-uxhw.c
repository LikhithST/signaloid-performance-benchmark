#include <stdio.h>
#include <stdlib.h>
#include <time.h>

int main(int argc, char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <iterations>\n", argv[0]);
        return 1;
    }

    double min = 0.05;
    double max = 0.07;
    int iterations = atoi(argv[1]);
    double principal = 100000.0;
    double sum_results = 0;

    srand(time(NULL));

    for (int i = 0; i < iterations; i++) {
        double daily_return = min + ((double)rand() / (double)RAND_MAX) * (max - min);
        double final_value = principal * (1 + daily_return);
        sum_results += final_value;
    }

    printf("%.2f\n", sum_results / iterations);
    return 0;
}