package raymond

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"sync"

	"github.com/komand/raymond/ast"
	"github.com/komand/raymond/parser"
)

// Template represents a handlebars template.
type Template struct {
	source    string
	program   *ast.Program
	helpers   map[string]reflect.Value
	partials  map[string]*partial
	mutex     sync.RWMutex // protects helpers and partials
	unescaped bool
}

// newTemplate instanciate a new template without parsing it
func newTemplate(source string, unescaped bool) *Template {
	return &Template{
		source:    source,
		helpers:   make(map[string]reflect.Value),
		partials:  make(map[string]*partial),
		unescaped: unescaped,
	}
}

// Parse instanciates a template by parsing given source.
func ParseUnescaped(source string) (*Template, error) {
	tpl := newTemplate(source, true)

	// parse template
	if err := tpl.parse(); err != nil {
		return nil, err
	}

	return tpl, nil
}

// ParseTemplate instanciates a template by parsing given source.
func ParseTemplate(source string, unescaped bool) (*Template, error) {
	tpl := newTemplate(source, unescaped)

	// parse template
	if err := tpl.parse(); err != nil {
		return nil, err
	}

	return tpl, nil
}

// Parse instanciates a template by parsing given source.
func Parse(source string) (*Template, error) {
	tpl := newTemplate(source, false)

	// parse template
	if err := tpl.parse(); err != nil {
		return nil, err
	}

	return tpl, nil
}

// MustParse instanciates a template by parsing given source. It panics on error.
func MustParseTemplate(source string, unescaped bool) *Template {
	result, err := ParseTemplate(source, unescaped)
	if err != nil {
		panic(err)
	}
	return result
}

// MustParse instanciates a template by parsing given source. It panics on error.
func MustParse(source string) *Template {
	return MustParseTemplate(source, false)
}

// ParseFile reads given file and returns parsed template.
func ParseFile(filePath string) (*Template, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ParseTemplate(string(b), false)
}

// parse parses the template
//
// It can be called several times, the parsing will be done only once.
func (tpl *Template) parse() error {
	if tpl.program == nil {
		var err error

		tpl.program, err = parser.Parse(tpl.source, tpl.unescaped)
		if err != nil {
			return err
		}
	}

	return nil
}

// Clone returns a copy of that template.
func (tpl *Template) Clone() *Template {
	result := newTemplate(tpl.source, tpl.unescaped)

	result.program = tpl.program

	tpl.mutex.RLock()
	defer tpl.mutex.RUnlock()

	for name, helper := range tpl.helpers {
		result.RegisterHelper(name, helper.Interface())
	}

	for name, partial := range tpl.partials {
		result.addPartial(name, partial.source, partial.tpl)
	}

	return result
}

func (tpl *Template) findHelper(name string) reflect.Value {
	tpl.mutex.RLock()
	defer tpl.mutex.RUnlock()

	return tpl.helpers[name]
}

// RegisterHelper registers a helper for that template.
func (tpl *Template) RegisterHelper(name string, helper interface{}) {
	tpl.mutex.Lock()
	defer tpl.mutex.Unlock()

	if tpl.helpers[name] != zero {
		panic(fmt.Sprintf("Helper %s already registered", name))
	}

	val := reflect.ValueOf(helper)
	ensureValidHelper(name, val)

	tpl.helpers[name] = val
}

// RegisterHelpers registers several helpers for that template.
func (tpl *Template) RegisterHelpers(helpers map[string]interface{}) {
	for name, helper := range helpers {
		tpl.RegisterHelper(name, helper)
	}
}

func (tpl *Template) addPartial(name string, source string, template *Template) {
	tpl.mutex.Lock()
	defer tpl.mutex.Unlock()

	if tpl.partials[name] != nil {
		panic(fmt.Sprintf("Partial %s already registered", name))
	}

	tpl.partials[name] = newPartial(name, source, template)
}

func (tpl *Template) findPartial(name string) *partial {
	tpl.mutex.RLock()
	defer tpl.mutex.RUnlock()

	return tpl.partials[name]
}

// RegisterPartial registers a partial for that template.
func (tpl *Template) RegisterPartial(name string, source string) {
	tpl.addPartial(name, source, nil)
}

// RegisterPartials registers several partials for that template.
func (tpl *Template) RegisterPartials(partials map[string]string) {
	for name, partial := range partials {
		tpl.RegisterPartial(name, partial)
	}
}

// RegisterPartialFile reads given file and registers its content as a partial with given name.
func (tpl *Template) RegisterPartialFile(filePath string, name string) error {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	tpl.RegisterPartial(name, string(b))

	return nil
}

// RegisterPartialFiles reads several files and registers them as partials, the filename base is used as the partial name.
func (tpl *Template) RegisterPartialFiles(filePaths ...string) error {
	if len(filePaths) == 0 {
		return nil
	}

	for _, filePath := range filePaths {
		name := fileBase(filePath)

		if err := tpl.RegisterPartialFile(filePath, name); err != nil {
			return err
		}
	}

	return nil
}

// RegisterPartialTemplate registers an already parsed partial for that template.
func (tpl *Template) RegisterPartialTemplate(name string, template *Template) {
	tpl.addPartial(name, "", template)
}

// Exec evaluates template with given context.
func (tpl *Template) Exec(ctx interface{}) (result string, err error) {
	return tpl.ExecWith(ctx, nil)
}

// MustExec evaluates template with given context. It panics on error.
func (tpl *Template) MustExec(ctx interface{}) string {
	result, err := tpl.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return result
}

// ExecWith evaluates template with given context and private data frame.
func (tpl *Template) ExecWith(ctx interface{}, privData *DataFrame) (result string, err error) {
	defer errRecover(&err)

	// parses template if necessary
	err = tpl.parse()
	if err != nil {
		return
	}

	// setup visitor
	v := newEvalVisitor(tpl, ctx, privData)

	// visit AST
	result, _ = tpl.program.Accept(v).(string)

	// named return values
	return
}

// errRecover recovers evaluation panic
func errRecover(errp *error) {
	e := recover()
	if e != nil {
		switch err := e.(type) {
		case runtime.Error:
			panic(e)
		case error:
			*errp = err
		default:
			panic(e)
		}
	}
}

// PrintAST returns string representation of parsed template.
func (tpl *Template) PrintAST() string {
	if err := tpl.parse(); err != nil {
		return fmt.Sprintf("PARSER ERROR: %s", err)
	}

	return ast.Print(tpl.program)
}

// Print returns string representation of parsed template.
func (tpl *Template) Print() string {
	// XXX
	return ast.PrintOriginal(tpl.program)
}

// Validate
func (tpl *Template) Validate(variables map[string]struct{}) (err error) {
	// parses template if necessary
	err = tpl.parse()
	if err != nil {
		return fmt.Errorf("Template could not be parsed: %s", err)
	}

	// setup visitor
	v := newValidateVisitor(tpl, variables)

	// visit AST
	err, _ = tpl.program.Accept(v).(error)

	// named return values
	return err
}

// Rename
func (tpl *Template) Rename(variables map[string]string) (err error) {
	// parses template if necessary
	err = tpl.parse()
	if err != nil {
		return fmt.Errorf("Template could not be parsed: %s", err)
	}

	defer errRecover(&err)

	// setup visitor
	v := newRenameVisitor(tpl, variables)

	// visit AST
	err, _ = tpl.program.Accept(v).(error)

	// named return values
	return err
}
