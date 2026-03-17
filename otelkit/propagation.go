package otelkit

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/propagation"
)

func newPropagator(names []string) (propagation.TextMapPropagator, error) {
	if len(names) == 0 {
		names = []string{"tracecontext", "baggage"}
	}
	propagators := make([]propagation.TextMapPropagator, 0, len(names))
	for _, name := range names {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "tracecontext":
			propagators = append(propagators, propagation.TraceContext{})
		case "baggage":
			propagators = append(propagators, propagation.Baggage{})
		case "b3":
			propagators = append(propagators, b3.New())
		case "b3multi":
			propagators = append(propagators, b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)))
		case "jaeger":
			propagators = append(propagators, jaeger.Jaeger{})
		case "xray":
			propagators = append(propagators, xray.Propagator{})
		case "none":
			return propagation.NewCompositeTextMapPropagator(), nil
		case "ottrace":
			return nil, fmt.Errorf("unsupported propagator %q", name)
		default:
			return nil, fmt.Errorf("unsupported propagator %q", name)
		}
	}
	return propagation.NewCompositeTextMapPropagator(propagators...), nil
}
