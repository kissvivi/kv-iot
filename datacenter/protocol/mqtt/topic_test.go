package mqtt

import (
	"errors"
	"testing"
)

func TestTopicTemplateToFilterAndMatch(t *testing.T) {
	filter, err := topicTemplateToFilter("iot/{productKey}/{deviceNo}/uplink")
	if err != nil {
		t.Fatalf("topicTemplateToFilter: %v", err)
	}
	if filter != "iot/+/+/uplink" {
		t.Fatalf("filter = %q", filter)
	}
	if !matchTopicFilter(filter, "iot/p1/d1/uplink") {
		t.Fatal("expected topic to match filter")
	}

	variables, matched := matchTopicTemplate("iot/{productKey}/{deviceNo}/uplink", "iot/p1/d1/uplink")
	if !matched {
		t.Fatal("expected template to match")
	}
	if variables["productKey"] != "p1" || variables["deviceNo"] != "d1" {
		t.Fatalf("variables = %#v", variables)
	}
}

func TestTopicTemplateEmbeddedVariableAndWildcard(t *testing.T) {
	variables, matched := matchTopicTemplate("site/${site}/device-${deviceNo}/#", "site/a/device-d1/state")
	if !matched {
		t.Fatal("expected embedded variables to match")
	}
	if variables["site"] != "a" || variables["deviceNo"] != "d1" {
		t.Fatalf("variables = %#v", variables)
	}

	filter, err := topicTemplateToFilter("device-{deviceNo}/state")
	if err != nil {
		t.Fatalf("topicTemplateToFilter: %v", err)
	}
	if filter != "+/state" {
		t.Fatalf("filter = %q, want +/state", filter)
	}
}

func TestRenderTemplate(t *testing.T) {
	rendered, err := renderString("iot/${productKey}/${deviceNo}/downlink", map[string]any{
		"productKey": "p1",
		"deviceNo":   "d1",
	})
	if err != nil {
		t.Fatalf("renderString: %v", err)
	}
	if rendered != "iot/p1/d1/downlink" {
		t.Fatalf("rendered = %q", rendered)
	}

	object, err := renderTemplateValue(map[string]any{
		"requestId": "${requestId}",
		"params":    "${params}",
	}, map[string]any{
		"requestId": "req-1",
		"params":    map[string]any{"enabled": true},
	})
	if err != nil {
		t.Fatalf("renderTemplateValue: %v", err)
	}
	renderedObject := object.(map[string]any)
	if renderedObject["requestId"] != "req-1" {
		t.Fatalf("requestId = %#v", renderedObject["requestId"])
	}
	if _, ok := renderedObject["params"].(map[string]any); !ok {
		t.Fatalf("params type = %T", renderedObject["params"])
	}

	if _, err := renderString("${missing}", map[string]any{}); !errors.Is(err, ErrTemplateVariable) {
		t.Fatalf("error = %v, want ErrTemplateVariable", err)
	}
}
