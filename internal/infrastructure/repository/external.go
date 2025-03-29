package repository

import "strings"

// Helper function to convert a map[string]string into an array of strings.
func convertToArray(m map[string]string) []string {
	result := make([]string, 0, len(m)) // Preallocate space for efficiency

	for key, value := range m {
		// Format the key-value pair as "key: value" and append to the result array
		result = append(result, key+": "+value)
	}

	return result
}

// Helper function to convert an array of strings into a map.
func convertToMap(arr []string) map[string]string {
	result := make(map[string]string)

	for _, item := range arr {
		// Split the string by the colon (":") delimiter
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])   // Trim any extra spaces around the key
			value := strings.TrimSpace(parts[1]) // Trim any extra spaces around the value
			result[key] = value
		}
	}

	return result
}
