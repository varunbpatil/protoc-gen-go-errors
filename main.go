package main

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"

	pgs "github.com/lyft/protoc-gen-star/v2"
	pgsgo "github.com/lyft/protoc-gen-star/v2/lang/go"

	"github.com/varunbpatil/protoc-gen-go-errors/errors"
)

const (
	errorSuffix = "Error"
)

func main() {
	pgs.Init(pgs.DebugEnv("DEBUG")).
		RegisterModule(ErrorsModule()).
		RegisterPostProcessor(pgsgo.GoFmt()).
		Render()
}

func ErrorsModule() *errorsModule {
	return &errorsModule{ModuleBase: &pgs.ModuleBase{}}
}

type errorsModule struct {
	*pgs.ModuleBase
	ctx pgsgo.Context
}

func (m *errorsModule) InitContext(ctx pgs.BuildContext) {
	m.ModuleBase.InitContext(ctx)
	m.ctx = pgsgo.InitContext(ctx.Parameters())
}

func (m *errorsModule) Name() string {
	return "go-errors"
}

func (m *errorsModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	for _, file := range targets {
		m.processFile(file)
	}
	return m.ModuleBase.Artifacts()
}

func (m *errorsModule) processFile(file pgs.File) {
	if len(file.AllMessages()) == 0 {
		return
	}

	var errorMessages []pgs.Message
	for _, msg := range file.AllMessages() {
		if msg.IsMapEntry() || !m.isErrorMessage(msg) {
			continue
		}
		errorMessages = append(errorMessages, msg)
	}

	if len(errorMessages) == 0 {
		return
	}

	// Use the naming pattern: {base}.errors.pb.go
	baseName := strings.TrimSuffix(file.InputPath().BaseName(), ".proto")
	filename := m.ctx.OutputPath(file).SetBase(baseName).SetExt(".errors.pb.go")

	// Generate file content
	content := m.generateFileContent(file, errorMessages)

	m.ModuleBase.AddGeneratorFile(filename.String(), content)
}

func (m *errorsModule) generateFileContent(file pgs.File, errorMessages []pgs.Message) string {
	var content strings.Builder

	// File header
	headerData := struct {
		PackageName string
	}{
		PackageName: m.ctx.PackageName(file).String(),
	}

	headerTmpl := template.Must(template.New("header").Parse(fileHeaderTemplate))
	if err := headerTmpl.Execute(&content, headerData); err != nil {
		m.ModuleBase.Failf("Failed to execute header template: %v", err)
	}

	// Generate methods for each error message
	for _, msg := range errorMessages {
		if m.isLeafError(msg) {
			m.generateLeafError(&content, msg)
		} else {
			oneofs := msg.OneOfs()
			if len(oneofs) > 1 {
				m.ModuleBase.Failf("Multiple oneofs not allowed in message %s", msg.Name())
			}
			m.generateSumError(&content, msg, oneofs[0])
		}
	}

	return content.String()
}

func (m *errorsModule) generateLeafError(content *strings.Builder, msg pgs.Message) {
	displayFormat, ok := m.getDisplayFormat(msg)
	if !ok {
		m.ModuleBase.Failf("Missing (errors.display) option in message %s", msg.Name())
	}

	m.validateFieldReferencesOrFail(msg, displayFormat)

	leafData := m.buildLeafErrorData(msg, displayFormat)

	leafTmpl := template.Must(template.New("leaf").Parse(leafErrorTemplate))
	if err := leafTmpl.Execute(content, leafData); err != nil {
		m.ModuleBase.Failf("Failed to execute leaf error template for %s: %v", msg.Name(), err)
	}
}

func (m *errorsModule) generateSumError(content *strings.Builder, msg pgs.Message, oneof pgs.OneOf) {
	sumData := m.buildSumErrorData(msg, oneof)

	sumTmpl := template.Must(template.New("sum").Parse(sumErrorTemplate))
	if err := sumTmpl.Execute(content, sumData); err != nil {
		m.ModuleBase.Failf("Failed to execute sum error template for %s: %v", msg.Name(), err)
	}
}

type LeafErrorData struct {
	GoName           string
	DisplayFormat    string
	FormatArgs       []string
	UnwrappableField *FieldData
}

type SumErrorData struct {
	GoName string
	Oneof  OneofData
}

type OneofData struct {
	GoName string
	Fields []FieldData
}

type FieldData struct {
	GoName  string
	Message *MessageData
}

type MessageData struct {
	GoName string
}

func (m *errorsModule) buildLeafErrorData(msg pgs.Message, displayFormat string) LeafErrorData {
	formatArgs := m.buildFieldArgs(msg, displayFormat)
	convertedFormat := m.convertToFmtPrintf(displayFormat, msg)
	unwrappableField := m.findUnwrappableField(msg, displayFormat)

	var unwrappableFieldData *FieldData
	if unwrappableField != nil {
		unwrappableFieldData = &FieldData{
			GoName: m.ctx.Name(unwrappableField).String(),
		}
	}

	return LeafErrorData{
		GoName:           m.ctx.Name(msg).String(),
		DisplayFormat:    convertedFormat,
		FormatArgs:       formatArgs,
		UnwrappableField: unwrappableFieldData,
	}
}

func (m *errorsModule) buildSumErrorData(msg pgs.Message, oneof pgs.OneOf) SumErrorData {
	var fields []FieldData
	for _, field := range oneof.Fields() {
		fieldData := FieldData{
			GoName: m.ctx.Name(field).String(),
		}
		// For oneof fields, check if they reference a message
		if field.Type().ProtoType() == pgs.MessageT || field.Type().IsEmbed() {
			if msgType := field.Type().Embed(); msgType != nil {
				fieldData.Message = &MessageData{
					GoName: m.ctx.Name(msgType).String(),
				}
			}
		}
		fields = append(fields, fieldData)
	}

	return SumErrorData{
		GoName: m.ctx.Name(msg).String(),
		Oneof: OneofData{
			GoName: m.ctx.Name(oneof).String(),
			Fields: fields,
		},
	}
}

// Helper methods

func (m *errorsModule) isErrorMessage(msg pgs.Message) bool {
	return strings.HasSuffix(m.ctx.Name(msg).String(), errorSuffix)
}

func (m *errorsModule) isLeafError(msg pgs.Message) bool {
	for _, field := range msg.Fields() {
		if field.InOneOf() {
			return false
		}
	}
	return true
}

func (m *errorsModule) getDisplayFormat(msg pgs.Message) (string, bool) {
	var displayValue string
	ok, err := msg.Extension(errors.E_Display, &displayValue)
	if err != nil {
		// Extension not found or type mismatch
		return "", false
	}

	if ok && displayValue != "" {
		return displayValue, true
	}

	return "", false
}

func (m *errorsModule) validateFieldReferencesOrFail(msg pgs.Message, display string) {
	refs := m.referencedFields(display)
	defined := map[string]bool{}

	for _, field := range msg.Fields() {
		defined[field.Name().String()] = true
	}

	for name := range refs {
		if !defined[name] {
			m.ModuleBase.Failf("Field {%s} in (errors.display) not found in message %s", name, msg.Name())
		}
	}
}

func (m *errorsModule) convertToFmtPrintf(format string, msg pgs.Message) string {
	refs := m.referencedFields(format)
	for _, field := range msg.Fields() {
		name := field.Name().String()
		if refs[name] {
			format = strings.ReplaceAll(format, "{"+name+"}", "%v")
		}
	}
	return format
}

func (m *errorsModule) buildFieldArgs(msg pgs.Message, displayFormat string) []string {
	re := regexp.MustCompile(`{([a-zA-Z0-9_]+)}`)
	matches := re.FindAllStringSubmatch(displayFormat, -1)

	var args []string
	for _, match := range matches {
		name := match[1]
		found := false
		for _, field := range msg.Fields() {
			if field.Name().String() == name {
				args = append(args, fmt.Sprintf("e.Get%s()", m.ctx.Name(field).String()))
				found = true
				break
			}
		}
		if !found {
			m.ModuleBase.Failf("field {%s} referenced in display format not found in message %s", name, msg.Name())
		}
	}

	return args
}

func (m *errorsModule) referencedFields(format string) map[string]bool {
	re := regexp.MustCompile(`{([a-zA-Z0-9_]+)}`)
	matches := re.FindAllStringSubmatch(format, -1)

	refFields := map[string]bool{}
	for _, match := range matches {
		if len(match) > 1 {
			refFields[match[1]] = true
		}
	}
	return refFields
}

func (m *errorsModule) findUnwrappableField(msg pgs.Message, displayFormat string) pgs.Field {
	refs := m.referencedFields(displayFormat)
	var unwrappables []pgs.Field

	for _, field := range msg.Fields() {
		name := field.Name().String()
		if refs[name] && (field.Type().ProtoType() == pgs.MessageT || field.Type().IsEmbed()) {
			if msgType := field.Type().Embed(); msgType != nil && m.hasDisplayOption(msgType) {
				unwrappables = append(unwrappables, field)
			}
		}
	}

	if len(unwrappables) > 1 {
		var names []string
		for _, f := range unwrappables {
			names = append(names, f.Name().String())
		}
		m.ModuleBase.Failf("only one unwrappable field allowed in message %s, found: %v", msg.Name(), names)
	}

	if len(unwrappables) == 1 {
		return unwrappables[0]
	}

	return nil
}

func (m *errorsModule) hasDisplayOption(entity pgs.Entity) bool {
	if msg, ok := entity.(pgs.Message); ok {
		_, ok := m.getDisplayFormat(msg)
		return ok
	}
	return false
}

// Templates
const fileHeaderTemplate = `
// Code generated by protoc-gen-go-errors. DO NOT EDIT.
package {{ .PackageName }}

import "fmt"
`

const leafErrorTemplate = `
func (e *{{ .GoName }}) Error() string {
	return fmt.Sprintf("{{ .DisplayFormat }}", {{ range $i, $arg := .FormatArgs }}{{if $i}}, {{end}}{{ $arg }}{{end}})
}

func (e *{{ .GoName }}) Unwrap() error {
	{{- if .UnwrappableField }}
	return e.Get{{ .UnwrappableField.GoName }}()
	{{- else }}
	return nil
	{{- end }}
}
`

const sumErrorTemplate = `
func (e *{{ .GoName }}) Error() string {
	switch v := e.{{ .Oneof.GoName }}.(type) {
	{{- range $field := .Oneof.Fields }}
	case *{{ $.GoName }}_{{ $field.GoName }}:
		return v.{{ $field.GoName }}.Error()
	{{- end }}
	default:
		return "unknown error"
	}
}

func (e *{{ .GoName }}) Unwrap() error {
	switch v := e.{{ .Oneof.GoName }}.(type) {
	{{- range $field := .Oneof.Fields }}
	case *{{ $.GoName }}_{{ $field.GoName }}:
		return v.{{ $field.GoName }}
	{{- end }}
	default:
		return nil
	}
}

{{- range $field := .Oneof.Fields }}
{{- if $field.Message }}

func (e *{{ $.GoName }}) From{{ $field.Message.GoName }}(leaf *{{ $field.Message.GoName }}) *{{ $.GoName }} {
	return &{{ $.GoName }}{Kind: &{{ $.GoName }}_{{ $field.GoName }}{
		{{ $field.GoName }}: leaf,
	}}
}
{{- end }}
{{- end }}
`
