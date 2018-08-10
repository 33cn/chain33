package common

func MinInt32(left, right int32) int32 {
	if left > right {
		return right
	}
	return left
}

func MaxInt32(left, right int32) int32 {
	if left > right {
		return left
	}
	return right
}
