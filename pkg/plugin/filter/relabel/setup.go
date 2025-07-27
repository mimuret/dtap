package relabel

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	"go.uber.org/zap"
)

const PLUGIN_NAME = "relabel"

func init() {
	_ = registry.RegisterFilterPlugin(PLUGIN_NAME, Setup)
}

// Setup initializes the Relabel filter plugin from the given FilterBlock configuration.
// It parses the HCL body and compiles templates for each relabel configuration.
func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	fp := &Relabel{
		FilterBlock: *cfg,
	}

	// Decode the HCL body into the Relabel struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, fp)
	if diags.HasErrors() {
		// Return an error if there are issues decoding the HCL body.
		return nil, plugin.PluginError(fp, "failed to setup Relabel plugin: %w", errors.Join(diags.Errs()...))
	}
	// Compile templates for each replace relabel configuration.
	for i, label := range fp.ReplaceLabels {
		if label.TargetLabel == "" {
			// TargetLabel must not be empty.
			return nil, plugin.PluginError(fp, "invalid TargetLabel[%d]: must not be empty", i)
		}
		if label.Replacement == "" {
			label.Replacement = label.TargetLabel
		}
		tmpl, err := template.New("relabel").Parse(label.Template)
		if err != nil {
			// Return an error if the template cannot be parsed.
			return nil, plugin.PluginError(fp, "failed to parse template for TargetLabel[%d]: %w", i, err)
		}
		// Store the compiled template in the relabel configuration.
		label.compiledTemplate = tmpl
	}
	// Compile templates for each drop relabel configuration.
	for i, label := range fp.DropLabels {
		if label.TargetLabel == "" {
			// TargetLabel must not be empty.
			return nil, plugin.PluginError(fp, "invalid TargetLabel[%d]: must not be empty", i)
		}
	}

	return fp, nil
}

// Relabel is a filter plugin that applies relabeling configurations to Dnstap messages.
// It supports dropping and replacing labels based on the provided configurations.
// Example HCL configuration:
//
//	filter "relabel" "default" {
//	  drop_label "unwanted_label_name" {}
//	  replace_label "old_label_name" "new_label_name" {
//	    template = "new_value"
//	  }
//	  replace_label "replace_label_name" "replace_label_name" {
//	    template = "new_value"
//	  }
//	}
type Relabel struct {
	config.FilterBlock
	// DropLabels contains the configurations for dropping labels.
	DropLabels []*DropRelabelConfig `hcl:"drop_label,block"`
	// ReplaceLabels contains the configurations for replacing labels.
	ReplaceLabels []*ReplaceReLabelConfig `hcl:"replace_label,block"`
}

// DropRelabelConfig represents a single drop relabel configuration.
type DropRelabelConfig struct {
	// TargetLabel is the label name to be dropped.
	TargetLabel string `hcl:"target_label,label"`
}

// ReplaceReLabelConfig represents a single replace relabel configuration.
type ReplaceReLabelConfig struct {
	// TargetLabel is the label name to be replaced.
	TargetLabel string `hcl:"target_label,label"`
	// Replacement is the string to be used for replacement in the label name.
	Replacement string `hcl:"replacement,label"`
	// Template is the template string used to generate the label value.
	Template string `hcl:"template"`
	// compiledTemplate is the parsed and compiled template.
	compiledTemplate *template.Template
}

// Filter processes a Dnstap message and applies the relabel configurations.
// It executes the compiled templates and updates the message labels.
// The relabeling is applied in the following order:
// 1. Replace configurations are applied first.
// 2. Drop configurations are applied after replace configurations.
func (r *Relabel) Filter(ctx context.Context, msg *types.DnstapMessage) *types.DnstapMessage {
	msg = msg.DeepCopy()
	// Apply replace configurations.
	for _, label := range r.ReplaceLabels {
		r.replaceLabel(ctx, msg, label)
	}
	// Apply drop configurations.
	for _, label := range r.DropLabels {
		r.dropLabel(msg, label)
	}
	return msg
}

// replaceLabel applies a single replace relabel configuration to the message.
func (r *Relabel) replaceLabel(ctx context.Context, msg *types.DnstapMessage, label *ReplaceReLabelConfig) {
	var buf = bytes.NewBufferString("")
	val, err := types.CreateMsgValue(r, msg)
	if err != nil {
		return
	}
	if err := label.compiledTemplate.Execute(buf, val); err != nil {
		ctxzap.Error(ctx, "failed to execute template", zap.Error(err), zap.String("label", label.TargetLabel))
		return
	}
	delete(msg.Labels, label.TargetLabel)
	msg.Labels[label.Replacement] = buf.String()
}

// dropLabel applies a single drop relabel configuration to the message.
func (r *Relabel) dropLabel(msg *types.DnstapMessage, label *DropRelabelConfig) {
	delete(msg.Labels, label.TargetLabel)
}
