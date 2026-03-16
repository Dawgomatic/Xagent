// SWE100821: Tests for decomposeTool metadata and dagSpec JSON parsing.

package agent

import (
	"encoding/json"
	"testing"
)

// SWE100821: verify decomposeTool Name and Description
func TestDecomposeTool_Name(t *testing.T) {
	tool := NewDecomposeTool(&mockProvider{}, "mock")

	if tool.Name() != "decompose" {
		t.Errorf("expected name 'decompose', got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

// SWE100821: verify decomposeTool Parameters schema
func TestDecomposeTool_Parameters(t *testing.T) {
	tool := NewDecomposeTool(&mockProvider{}, "mock")
	params := tool.Parameters()

	if params == nil {
		t.Fatal("expected non-nil parameters")
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'properties' in parameters")
	}
	if _, ok := props["task"]; !ok {
		t.Error("expected 'task' property in parameters")
	}

	req, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected 'required' in parameters")
	}
	found := false
	for _, r := range req {
		if r == "task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'task' in required list")
	}
}

// SWE100821: verify dagSpec JSON round-trip
func TestDagSpec_JSONParsing(t *testing.T) {
	input := `{
		"nodes": [
			{"id": "t1", "description": "Research topic", "role": "researcher", "depends_on": []},
			{"id": "t2", "description": "Write code", "role": "coder", "depends_on": ["t1"]},
			{"id": "t3", "description": "Review code", "role": "reviewer", "depends_on": ["t2"]}
		]
	}`

	var spec dagSpec
	if err := json.Unmarshal([]byte(input), &spec); err != nil {
		t.Fatalf("failed to parse dagSpec JSON: %v", err)
	}

	if len(spec.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(spec.Nodes))
	}

	// SWE100821: verify first node
	if spec.Nodes[0].ID != "t1" {
		t.Errorf("expected node[0].ID='t1', got %q", spec.Nodes[0].ID)
	}
	if spec.Nodes[0].Role != "researcher" {
		t.Errorf("expected node[0].Role='researcher', got %q", spec.Nodes[0].Role)
	}
	if len(spec.Nodes[0].DependsOn) != 0 {
		t.Errorf("expected no dependencies for t1, got %v", spec.Nodes[0].DependsOn)
	}

	// SWE100821: verify dependency chain
	if len(spec.Nodes[1].DependsOn) != 1 || spec.Nodes[1].DependsOn[0] != "t1" {
		t.Errorf("expected t2 to depend on t1, got %v", spec.Nodes[1].DependsOn)
	}
	if len(spec.Nodes[2].DependsOn) != 1 || spec.Nodes[2].DependsOn[0] != "t2" {
		t.Errorf("expected t3 to depend on t2, got %v", spec.Nodes[2].DependsOn)
	}
}

// SWE100821: verify dagSpec with empty nodes
func TestDagSpec_EmptyNodes(t *testing.T) {
	input := `{"nodes": []}`
	var spec dagSpec
	if err := json.Unmarshal([]byte(input), &spec); err != nil {
		t.Fatalf("failed to parse empty dagSpec: %v", err)
	}
	if len(spec.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(spec.Nodes))
	}
}

// SWE100821: verify dagNodeSpec fields
func TestDagNodeSpec_Fields(t *testing.T) {
	node := dagNodeSpec{
		ID:          "test-1",
		Description: "Test task",
		Role:        "coder",
		DependsOn:   []string{"dep-1", "dep-2"},
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("failed to marshal dagNodeSpec: %v", err)
	}

	var decoded dagNodeSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal dagNodeSpec: %v", err)
	}

	if decoded.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got %q", decoded.ID)
	}
	if len(decoded.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(decoded.DependsOn))
	}
}
