package utils

func InsertIfNotPresent[T comparable](slice []T, item T) []T {
	for _, s := range slice {
		if item == s {
			return slice
		}
	}
	return append(slice, item)
}
