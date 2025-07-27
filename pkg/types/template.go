package types

import "fmt"

type PluginValue struct {
	Type     string
	Name     string
	FullName string
}

type MsgValue struct {
	Plugin  PluginValue
	Labels  map[string]string
	Message *dnstapV1Flat
}

func CreateMsgValue(p Plugin, dm *DnstapMessage) (*MsgValue, error) {
	if p == nil || dm == nil {
		return nil, fmt.Errorf("plugin or dnstap message is nil")
	}
	v, err := dm.ConvertV1Flat()
	if err != nil {
		return nil, fmt.Errorf("failed to convert dnstap message to map: %w", err)
	}
	return &MsgValue{
		Plugin: PluginValue{
			Type:     p.GetType(),
			Name:     p.GetName(),
			FullName: p.GetFullName(),
		},
		Labels:  dm.Labels,
		Message: v,
	}, nil
}
