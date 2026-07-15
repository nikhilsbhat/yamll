package yamll

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

func (result ExplainResult) String() string {
	return result.RenderText()
}

func (result ExplainResult) RenderText() string {
	var builder strings.Builder

	builder.WriteString("path: ")
	builder.WriteString(result.Path)
	builder.WriteString("\n\n")

	builder.WriteString("final:\n  origin: ")
	builder.WriteString(result.FinalOrigin.Origin)
	builder.WriteString("\n  value:\n")
	writeExplainIndentedValue(&builder, result.FinalValue, "    ")

	builder.WriteString("\nresolution:\n")

	for index, step := range result.Steps {
		builder.WriteString("  ")
		builder.WriteString(strconv.Itoa(index + 1))
		builder.WriteString(". origin: ")
		builder.WriteString(step.Origin)
		builder.WriteByte('\n')
		builder.WriteString("     source: ")
		builder.WriteString(step.Source)
		builder.WriteString("\n     action: ")
		builder.WriteString(step.Action)
		builder.WriteString("\n     value:\n")
		writeExplainIndentedValue(&builder, step.Value, "       ")
	}

	builder.WriteString("\nmerge rule: ")
	builder.WriteString(result.MergeRule)
	builder.WriteString("\n")

	return builder.String()
}

func (result ExplainResult) RenderJSON() ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func (result ExplainResult) RenderYAML() ([]byte, error) {
	return yaml.MarshalWithOptions(result, yaml.Indent(yamlIndent), yaml.IndentSequence(true))
}

func writeExplainIndentedValue(builder *strings.Builder, value, indent string) {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		builder.WriteString(indent)
		builder.WriteString("(empty)\n")

		return
	}

	for line := range strings.SplitSeq(value, "\n") {
		builder.WriteString(indent)
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
}
