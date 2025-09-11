package utils

// BuildDataMap construit une map[string]string à partir d'arguments variadiques.
// Usage : BuildDataMap("clé1", "val1", "clé2", "val2", ...)
func BuildDataMap(kv ...string) map[string]string {
	if len(kv)%2 != 0 {
		panic("BuildDataMap requires an even number of arguments (clé, valeur)")
	}

	data := make(map[string]string, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key := kv[i]
		value := kv[i+1]
		data[key] = value
	}
	return data
}
