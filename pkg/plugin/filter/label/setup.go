package label

import (
	"fmt"
	"strings"

	"github.com/goccy/go-json"

	"github.com/mimuret/dnsutils/getter"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterFilterPlugin("label", Setup)
}

func Setup(raw json.RawMessage) (types.FilterPlugin, error) {
	fp := &Label{}
	if err := json.Unmarshal(raw, fp); err != nil {
		return nil, err
	}
	for i, add := range fp.Add {
		if err := add.Setup(); err != nil {
			return nil, errors.Wrapf(err, "invalid Add[%d]", i)
		}

	}
	for i, del := range fp.Del {
		if err := del.Validate(); err != nil {
			return nil, errors.Wrapf(err, "invalid Add[%d]", i)
		}
	}
	return fp, nil
}

var _ types.FilterPlugin = &Label{}

type AddLabelType string

const (
	AddLabelTypeDNSTAP AddLabelType = "DNSTAP"
	AddLabelTypeDNS    AddLabelType = "DNS"
	AddLabelTypeSTATIC AddLabelType = "STATIC"
)

type AddLabel struct {
	// Label key name for adding or updating. It must not be empty.
	Name string
	// Type can be one of `DNSTAP`, `DNS`, or `STATIC`.
	Type AddLabelType
	// If `Type` is `DNSTAP`, the function name of DNSTAP of the getter function is set to value, and the value of DNSTAP message can be written in the label.
	// You can see getter names https://pkg.go.dev/github.com/mimuret/dnsutils/getter#DnstapGetterName.
	// If `Type` is `DNS`, the function name of the DNS message of the getter function is set to value, and the value of the DNS message can be written into the label.
	// You can see getter names https://pkg.go.dev/github.com/mimuret/dnsutils/getter#DnsMsgGetterName.
	// If the Type is `STATIC`, the string of the value is written directly into the label.
	Value  string
	getter types.DnstapMessageGetFunc
}

func (l *AddLabel) Setup() error {
	if l.Name == "" {
		return fmt.Errorf("missing parameter Name")
	}
	if l.Type == "" {
		l.Type = AddLabelTypeSTATIC
	}
	if l.Value == "" {
		return fmt.Errorf("missing parameter Value")
	}
	switch strings.ToUpper(string(l.Type)) {
	case "DNSTAP":
		getFunc := getter.NewDnstapStrFunc(getter.DnstapGetterName(l.Value))
		if getFunc == nil {
			return fmt.Errorf("unknown DNSTAP get func %s", l.Value)
		}
		l.getter = types.NewGetFuncFromDnstap(getFunc)
	case "DNS":
		getFunc := getter.NewDnsMsgStrFunc(getter.DnsMsgGetterName(l.Value))
		if getFunc == nil {
			return fmt.Errorf("unknown DNS get func %s", l.Value)
		}
		l.getter = types.NewGetFuncFromDnsMsg(getFunc)
	case "STATIC":
		l.getter = func(*types.DnstapMessage) string {
			return l.Value
		}
	default:
		return fmt.Errorf("unknown Type %s", l.Type)
	}
	return nil
}

// Delete Label
type DelLabel struct {
	// Label key name for deleting. It must not be empty.
	Name string
}

func (l *DelLabel) Validate() error {
	if l.Name == "" {
		return fmt.Errorf("missing parameter Name")
	}
	return nil
}

// The label plugin edits labels.
type Label struct {
	plugin.PluginCommon
	// Adding and updating labels
	Add []*AddLabel
	// Deleting labels
	Del []*DelLabel
}

func (f *Label) Filter(t *types.DnstapMessage) *types.DnstapMessage {
	dt := t.GetDnstap()
	if dt.Message == nil {
		return t
	}
	for _, add := range f.Add {
		t.Labels[add.Name] = add.getter(t)
	}
	for _, del := range f.Del {
		delete(t.Labels, del.Name)
	}
	return t
}
