package privacy

// 62387455827 -> 455827 + 7000000 + 80000000 + 300000000 + 2000000000 + 60000000000, where 455827 <= dust_threshold
//res:[455827, 7000000, 80000000, 300000000, 2000000000, 60000000000]
func decomposeAmount2digits(amount, dust_threshold int64) []int64 {
	res := make([]int64, 0)
	if 0 == amount {
		return res
	}

	is_dust_handled := false
	var dust int64 = 0
	var order int64 = 1
	var chunk int64 = 0

	for 0 != amount {
		chunk = (amount % 10) * order
		amount /= 10
		order *= 10
		if dust+chunk < dust_threshold {
			dust += chunk //累加小数，直到超过dust_threshold为止
		} else {
			if !is_dust_handled && 0 != dust {
				//正常情况下，先把dust保存下来
				res = append(res, dust)
				is_dust_handled = true
			}
			if 0 != chunk {
				//然后依次将大的整数额度进行保存
				res = append(res, chunk)
			}
		}
	}

	//如果需要被拆分的额度 < dust_threshold，则直接将其进行保存
	if !is_dust_handled && 0 != dust {
		res = append(res, dust)
	}

	return res
}
