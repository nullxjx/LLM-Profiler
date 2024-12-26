package utils

import "math"

const (
	MinimumCount = 3
)

// IsClose 判断两个浮点数是否接近
func IsClose(a, b, tolerance float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	if a == 0 || b == 0 {
		return false
	}
	relativeError := math.Abs((a - b) / math.Max(math.Abs(a), math.Abs(b)))
	return relativeError <= tolerance
}

// MeanWithoutMinMax 计算平均值，排除最大值和最小值
// todo(@nullxjx) 有待优化
func MeanWithoutMinMax(numbers []float64) float64 {
	if len(numbers) < MinimumCount {
		return 0
	}

	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64
	sum := 0.0

	for _, num := range numbers {
		sum += num
		if num < minVal {
			minVal = num
		}
		if num > maxVal {
			maxVal = num
		}
	}

	sum -= minVal + maxVal
	mean := sum / float64(len(numbers)-2)
	return mean
}
