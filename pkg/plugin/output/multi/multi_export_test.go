package multi

import "github.com/mimuret/dtap/v2/pkg/types"

func (m *MultiRunner) Plugins() []types.OutputPlugin {
	return m.plugns
}
