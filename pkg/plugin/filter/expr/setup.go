package expr

import (
	"context"
	"fmt"

	goexpr "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	"go.uber.org/zap"
)

const PLUGIN_NAME = "expr"

func init() {
	_ = registry.RegisterFilterPlugin(PLUGIN_NAME, Setup)
}

// Example HCL configuration:
// filter "expr" "hoge" {
//    expression =  <<EOF
//    qtype == "A"
//    EOF
// }

// Expr is a filter plugin that evaluates DNSTAP messages using go-expr expressions.
type Expr struct {
	config.FilterBlock
	// Expression is the expr expression to evaluate.
	// Refer to the output of types.ConvertV1MapString for the full list of available variables.
	Expression string `hcl:"expression"`
	// ErrorLogEnabled indicates whether to log errors during expression evaluation.
	ErrorLogEnabled bool `hcl:"error_log_enabled,optional"`

	program *vm.Program // Compiled expression program
}

// Setup initializes the Expr plugin from the given FilterBlock configuration.
func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	fp := &Expr{
		FilterBlock: *cfg,
	}

	// Decode the HCL body into the ExprFilter struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, fp)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL body: %w", diags)
	}

	// Compile the expr expression.
	program, err := goexpr.Compile(fp.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}
	fp.program = program

	return fp, nil
}

// Filter evaluates the DNSTAP message using the expr expression.
// If the expression evaluates to true, the message is returned; otherwise, nil is returned.
func (f *Expr) Filter(ctx context.Context, dm *types.DnstapMessage) *types.DnstapMessage {
	// Evaluate the expression.
	val, err := dm.ConvertV1MapString()
	if err != nil {
		return nil
	}
	output, err := goexpr.Run(f.program, val)
	if err != nil {
		if f.ErrorLogEnabled {
			ctxzap.Error(ctx, "failed to evaluate expression", zap.Error(err))
		}
		return nil
	}

	// Check the evaluation result.
	if result, ok := output.(bool); ok {
		if result {
			return dm
		}
	} else if f.ErrorLogEnabled {
		// Log if the result is not a boolean and error logging is enabled.
		ctxzap.Error(ctx, "expression did not evaluate to a boolean", zap.Any("result", output))
	}
	return nil
}
