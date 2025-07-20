package oteltrace

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func GetResource(kv map[string]string) *resource.Resource {
	var attrs = make([]attribute.KeyValue, 0, len(kv))
	for k, v := range kv {
		attrs = append(attrs, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.StringValue(v)})
	}
	return resource.NewSchemaless(attrs...)
}
