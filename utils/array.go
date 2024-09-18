package utils

func InsertIfNotPresent(slice []string, item string) []string {
	for _, s := range slice {
		if item == s {
			return slice
		}
	}
	return append(slice, item)
}
