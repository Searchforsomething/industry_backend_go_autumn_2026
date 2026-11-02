package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}
	differenceStats := Stats{Min: nums[1] - nums[0], Max: nums[1] - nums[0]}

	for i := range len(nums) - 1 {
		difference := nums[i+1] - nums[i]
		differenceStats.Count++
		differenceStats.Sum += difference
		if difference > differenceStats.Max {
			differenceStats.Max = difference
		}
		if difference < differenceStats.Min {
			differenceStats.Min = difference
		}
	}
	return differenceStats
}
