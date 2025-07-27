package types

// filter map[string]interface
type OutputFilters struct {
	IncludeKeys []string `hcl:"include_keys,optional"`
	ExcludeKeys []string `hcl:"exclude_keys,optional"`
}

func (kf *OutputFilters) Filter(kv map[string]interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	if len(kf.IncludeKeys) > 0 {
		for _, k := range kf.IncludeKeys {
			if v, ok := kv[k]; ok {
				res[k] = v
			}
		}
	} else {
		res = kv
	}
	for _, k := range kf.ExcludeKeys {
		delete(res, k)
	}
	return res
}
