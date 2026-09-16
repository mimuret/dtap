package types_test

import (
	"testing"

	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
)

var benchDM = testtool.CreateValidDnstapMessage()

// ConvertV1JSON: struct を直接 json.Marshal
func BenchmarkConvertV1JSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := benchDM.ConvertV1JSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ConvertV1JSONWithFilter (フィルタなし): map 経由で json.Marshal
func BenchmarkConvertV1JSONWithFilter_NoFilter(b *testing.B) {
	b.ReportAllocs()
	filter := types.OutputFilters{}
	for i := 0; i < b.N; i++ {
		_, err := benchDM.ConvertV1JSONWithFilter(filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ConvertV1JSONWithFilter (IncludeKeys あり)
func BenchmarkConvertV1JSONWithFilter_Include(b *testing.B) {
	b.ReportAllocs()
	filter := types.OutputFilters{IncludeKeys: []string{"qname", "qtype", "qclass", "rcode"}}
	for i := 0; i < b.N; i++ {
		_, err := benchDM.ConvertV1JSONWithFilter(filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ConvertV1JSONWithFilter (ExcludeKeys あり)
func BenchmarkConvertV1JSONWithFilter_Exclude(b *testing.B) {
	b.ReportAllocs()
	filter := types.OutputFilters{ExcludeKeys: []string{"query_address_hash", "response_address_hash", "ecs_net"}}
	for i := 0; i < b.N; i++ {
		_, err := benchDM.ConvertV1JSONWithFilter(filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}
