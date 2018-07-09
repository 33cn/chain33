package privacy

func checkByteSliceValid(s []byte) bool {
	return s != nil && len(s) > 0
}
