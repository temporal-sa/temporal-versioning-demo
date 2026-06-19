package dashboard

import (
	"context"
	"fmt"
	"log/slog"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
)

const SessionSearchAttribute = "PizzaSessionID"

// EnsureSessionSearchAttribute registers the session visibility key this demo
// uses to list only the current browser session's orders. It is safe to call on
// every backend startup.
func EnsureSessionSearchAttribute(
	ctx context.Context,
	c client.Client,
	namespace string,
	logger *slog.Logger,
) error {
	if ok, err := sessionSearchAttributeReady(ctx, c, namespace); err != nil {
		return err
	} else if ok {
		return nil
	}

	_, err := c.OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
		Namespace: namespace,
		SearchAttributes: map[string]enumspb.IndexedValueType{
			SessionSearchAttribute: enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		},
	})
	if err == nil {
		logger.Info("registered Temporal search attribute", "name", SessionSearchAttribute)
		return nil
	}

	// Another backend replica may have registered it between List and Add.
	if ok, checkErr := sessionSearchAttributeReady(ctx, c, namespace); checkErr != nil {
		return fmt.Errorf("add search attribute %s: %w; recheck failed: %w",
			SessionSearchAttribute, err, checkErr)
	} else if ok {
		return nil
	}
	return fmt.Errorf("add search attribute %s: %w", SessionSearchAttribute, err)
}

func sessionSearchAttributeReady(ctx context.Context, c client.Client, namespace string) (bool, error) {
	resp, err := c.OperatorService().ListSearchAttributes(ctx, &operatorservice.ListSearchAttributesRequest{
		Namespace: namespace,
	})
	if err != nil {
		return false, fmt.Errorf("list search attributes: %w", err)
	}

	if typ, ok := resp.GetCustomAttributes()[SessionSearchAttribute]; ok {
		if typ != enumspb.INDEXED_VALUE_TYPE_KEYWORD {
			return false, fmt.Errorf("search attribute %s has type %s, want KEYWORD",
				SessionSearchAttribute, typ.String())
		}
		return true, nil
	}
	if typ, ok := resp.GetSystemAttributes()[SessionSearchAttribute]; ok {
		if typ != enumspb.INDEXED_VALUE_TYPE_KEYWORD {
			return false, fmt.Errorf("search attribute %s has type %s, want KEYWORD",
				SessionSearchAttribute, typ.String())
		}
		return true, nil
	}
	return false, nil
}
