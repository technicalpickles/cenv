package settings

// DeepMerge recursively merges overlay into base and returns a new map.
// Objects are merged recursively. All other types (scalars, arrays) in overlay
// replace the corresponding base value. Neither input is modified.
func DeepMerge(base, overlay map[string]any) map[string]any {
	result := make(map[string]any, len(base))

	for k, v := range base {
		result[k] = v
	}

	for k, overlayVal := range overlay {
		if overlayMap, overlayIsMap := overlayVal.(map[string]any); overlayIsMap {
			if baseMap, baseIsMap := result[k].(map[string]any); baseIsMap {
				result[k] = DeepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = overlayVal
	}

	return result
}
