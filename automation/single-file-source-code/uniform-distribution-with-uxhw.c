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
    printf("%lf\n", final_value);

    return 0;
}