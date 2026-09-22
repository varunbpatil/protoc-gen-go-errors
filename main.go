package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	pgs "github.com/lyft/protoc-gen-star/v2"
	pgsgo "github.com/lyft/protoc-gen-star/v2/lang/go"

	"github.com/varunbpatil/protoc-gen-go-errors/errors"
)

const (
	errorSuffix = "Error"
)

// placeholderRE matches {field_name} references in an (errors.display) format.
var placeholderRE = regexp.MustCompile(`{([a-zA-Z0-9_]+)}`)

// Templates are parsed once at startup rather than per message.
var (
	headerTmpl = template.Must(template.New("header").Parse(fileHeaderTemplate))
	leafTmpl   = template.Must(template.New("leaf").Parse(leafErrorTemplate))
	sumTmpl    = template.Must(template.New("sum").Parse(sumErrorTemplate))
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
	return m.Artifacts()
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

	m.AddGeneratorFile(filename.String(), content)
}

func (m *errorsModule) generateFileContent(file pgs.File, errorMessages []pgs.Message) string {
	var content strings.Builder

	// Every error message generates code that uses fmt: leaf errors via
	// Error()'s fmt.Sprintf and sum errors via From()'s panic message. The
	// google.golang.org/protobuf/proto package is only imported when the file
	// contains sum errors, whose From() glue interface embeds proto.Message.
	usesFmt := len(errorMessages) > 0
	usesProto := false
	for _, msg := range errorMessages {
		if !m.isLeafError(msg) {
			usesProto = true
			break
		}
	}

	var imports []string
	if usesFmt {
		imports = append(imports, "fmt")
	}
	if usesProto {
		imports = append(imports, "google.golang.org/protobuf/proto")
	}

	// File header
	headerData := struct {
		PackageName string
		Imports     []string
	}{
		PackageName: m.ctx.PackageName(file).String(),
		Imports:     imports,
	}

	if err := headerTmpl.Execute(&content, headerData); err != nil {
		m.Failf("Failed to execute header template: %v", err)
	}

	// Collect the marker method names that each leaf error defined in this file
	// must implement. Every sum error (in any file of the same package) that
	// wraps a leaf requires its own marker, so a leaf shared by several sums
	// implements one marker method per sum.
	leafMarkers := m.collectLeafMarkers(file)

	// Generate methods for each error message
	for _, msg := range errorMessages {
		if m.isLeafError(msg) {
			m.generateLeafError(&content, msg, leafMarkers[m.ctx.Name(msg).String()])
			continue
		}

		oneofs := msg.OneOfs()
		if len(oneofs) != 1 {
			m.Failf("message %s must contain exactly one oneof, found %d", msg.Name(), len(oneofs))
		}
		m.generateSumError(&content, msg, oneofs[0])
	}

	return content.String()
}

func (m *errorsModule) generateLeafError(content *strings.Builder, msg pgs.Message, markers []string) {
	displayFormat, ok := m.getDisplayFormat(msg)
	if !ok {
		m.Failf("Missing (errors.display) option in message %s", msg.Name())
	}

	m.validateFieldReferencesOrFail(msg, displayFormat)

	leafData := m.buildLeafErrorData(msg, displayFormat)
	leafData.Markers = markers

	if err := leafTmpl.Execute(content, leafData); err != nil {
		m.Failf("Failed to execute leaf error template for %s: %v", msg.Name(), err)
	}
}

func (m *errorsModule) generateSumError(content *strings.Builder, msg pgs.Message, oneof pgs.OneOf) {
	sumData := m.buildSumErrorData(msg, oneof)

	if err := sumTmpl.Execute(content, sumData); err != nil {
		m.Failf("Failed to execute sum error template for %s: %v", msg.Name(), err)
	}
}

type LeafErrorData struct {
	GoName           string
	DisplayFormat    string
	FormatArgs       []string
	UnwrappableField *FieldData
	Markers          []string
}

type SumErrorData struct {
	GoName          string
	MarkerInterface string
	MarkerMethod    string
	Oneof           OneofData
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
	unwrappableField := m.findUnwrappableField(msg, displayFormat)

	var unwrappableFieldData *FieldData
	if unwrappableField != nil {
		unwrappableFieldData = &FieldData{
			GoName: m.ctx.Name(unwrappableField).String(),
		}
	}

	return LeafErrorData{
		GoName: m.ctx.Name(msg).String(),
		// DisplayFormat is rendered as a quoted Go string literal with the
		// field references converted to fmt verbs.
		DisplayFormat:    strconv.Quote(toPrintfFormat(displayFormat)),
		FormatArgs:       formatArgs,
		UnwrappableField: unwrappableFieldData,
	}
}

func (m *errorsModule) buildSumErrorData(msg pgs.Message, oneof pgs.OneOf) SumErrorData {
	var fields []FieldData
	for _, field := range oneof.Fields() {
		msgType := field.Type().Embed()
		if msgType == nil {
			m.Failf("oneof field %s in message %s must reference an error message", field.Name(), msg.Name())
		}
		if !strings.HasSuffix(m.ctx.Name(msgType).String(), errorSuffix) {
			m.Failf(
				"oneof field %s in message %s must reference a message whose name ends with %q",
				field.Name(), msg.Name(), errorSuffix,
			)
		}
		fields = append(fields, FieldData{
			GoName:  m.ctx.Name(field).String(),
			Message: &MessageData{GoName: m.ctx.Name(msgType).String()},
		})
	}

	goName := m.ctx.Name(msg).String()
	return SumErrorData{
		GoName:          goName,
		MarkerInterface: markerInterfaceName(goName),
		MarkerMethod:    markerMethodName(goName),
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

// collectLeafMarkers returns, for each leaf error message defined in file, the
// marker method names of every sum error in the same package that wraps it.
// The From() constructor of a sum error is typed against an unexported
// interface that only its own leaves can satisfy; each such interface requires
// a uniquely named marker method on the leaf so that a leaf shared by several
// sums can satisfy them all.
func (m *errorsModule) collectLeafMarkers(file pgs.File) map[string][]string {
	markers := map[string][]string{}
	for _, pkgFile := range file.Package().Files() {
		for _, pkgMsg := range pkgFile.AllMessages() {
			if pkgMsg.IsMapEntry() || !m.isErrorMessage(pkgMsg) || m.isLeafError(pkgMsg) {
				continue
			}
			oneofs := pkgMsg.OneOfs()
			if len(oneofs) != 1 {
				continue
			}
			sumMarker := markerMethodName(m.ctx.Name(pkgMsg).String())
			for _, field := range oneofs[0].Fields() {
				msgType := field.Type().Embed()
				if msgType == nil || !strings.HasSuffix(m.ctx.Name(msgType).String(), errorSuffix) {
					continue
				}
				if msgType.File().Name() != file.Name() {
					continue
				}
				goName := m.ctx.Name(msgType).String()
				markers[goName] = append(markers[goName], sumMarker)
			}
		}
	}
	return markers
}

// markerInterfaceName returns the unexported interface type that From() is
// typed against for a sum error. Only the sum's own leaves (which implement
// its marker method) satisfy it, so the compiler rejects leaves of other sum
// errors at the call site.
func markerInterfaceName(goName string) string {
	return "from" + goName
}

// markerMethodName returns the per-sum marker method that each of a sum
// error's leaves must implement. The sum's Go name is embedded in the method
// name so that a leaf shared by several sums can implement several markers
// without collision.
func markerMethodName(goName string) string {
	return strings.ToLower(goName[:1]) + goName[1:] + "Marker"
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
	defined := map[string]bool{}
	for _, field := range msg.Fields() {
		defined[field.Name().String()] = true
	}

	for _, name := range placeholderNames(display) {
		if !defined[name] {
			m.Failf("Field {%s} in (errors.display) not found in message %s", name, msg.Name())
		}
	}
}

func (m *errorsModule) buildFieldArgs(msg pgs.Message, displayFormat string) []string {
	var args []string
	for _, name := range placeholderNames(displayFormat) {
		found := false
		for _, field := range msg.Fields() {
			if field.Name().String() == name {
				args = append(args, fmt.Sprintf("e.Get%s()", m.ctx.Name(field).String()))
				found = true
				break
			}
		}
		if !found {
			m.Failf("field {%s} referenced in display format not found in message %s", name, msg.Name())
		}
	}

	return args
}

func (m *errorsModule) referencedFields(format string) map[string]bool {
	refFields := map[string]bool{}
	for _, name := range placeholderNames(format) {
		refFields[name] = true
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
		m.Failf("only one unwrappable field allowed in message %s, found: %v", msg.Name(), names)
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

// placeholderNames returns the field names referenced by {field_name}
// placeholders in a display format, in order of appearance.
func placeholderNames(format string) []string {
	matches := placeholderRE.FindAllStringSubmatch(format, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	return names
}

// toPrintfFormat converts an (errors.display) format into a fmt format string.
// Literal percent signs are escaped so that only the substituted field verbs
// are interpreted by fmt.
func toPrintfFormat(format string) string {
	format = strings.ReplaceAll(format, "%", "%%")
	return placeholderRE.ReplaceAllString(format, "%v")
}

// Templates
const fileHeaderTemplate = `
// Code generated by protoc-gen-go-errors. DO NOT EDIT.
package {{ .PackageName }}
{{- if .Imports }}

import (
{{- range .Imports }}
	"{{ . }}"
{{- end }}
)
{{- end }}
`

const leafErrorTemplate = `
func (e *{{ .GoName }}) Error() string {
	return fmt.Sprintf({{ .DisplayFormat }}, {{ range $i, $arg := .FormatArgs }}{{if $i}}, {{end}}{{ $arg }}{{end}})
}

func (e *{{ .GoName }}) Unwrap() error {
	{{- if .UnwrappableField }}
	if v := e.Get{{ .UnwrappableField.GoName }}(); v != nil {
		return v
	}
	{{- end }}
	return nil
}
{{- range .Markers }}

func (*{{ $.GoName }}) {{ . }}() {}
{{- end }}
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
		if v.{{ $field.GoName }} != nil {
			return v.{{ $field.GoName }}
		}
	{{- end }}
	}
	return nil
}

type {{ .MarkerInterface }} interface {
	error
	proto.Message
	{{ .MarkerMethod }}()
}

func (e *{{ .GoName }}) From(leaf {{ .MarkerInterface }}) *{{ .GoName }} {
	switch v := leaf.(type) {
	{{- range $field := .Oneof.Fields }}
	{{- if $field.Message }}
	case *{{ $field.Message.GoName }}:
		return &{{ $.GoName }}{Kind: &{{ $.GoName }}_{{ $field.GoName }}{
			{{ $field.GoName }}: v,
		}}
	{{- end }}
	{{- end }}
	default:
		panic(fmt.Sprintf("protoc-gen-go-errors: %T is not one of the error messages of {{ .GoName }}", leaf))
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
