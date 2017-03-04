package raymond

import (
	"fmt"
	"sync"
)

// partial represents a partial template
type partial struct {
	name      string
	source    string
	tpl       *Template
	unescaped bool
}

// partials stores all global partials
var partials map[string]*partial

// protects global partials
var partialsMutex sync.RWMutex

func init() {
	partials = make(map[string]*partial)
}

// newPartial instanciates a new partial
func newPartial(name string, source string, tpl *Template) *partial {
	return &partial{
		name:      name,
		source:    source,
		tpl:       tpl,
		unescaped: tpl.unescaped,
	}
}

// RegisterPartial registers a global partial. That partial will be available to all templates.
func RegisterPartial(name string, source string, unescaped bool) {
	partialsMutex.Lock()
	defer partialsMutex.Unlock()

	if partials[name] != nil {
		panic(fmt.Errorf("Partial already registered: %s", name))
	}

	partials[name] = newPartial(name, source, nil)
	partials[name].unescaped = unescaped
}

// RegisterPartials registers several global partials. Those partials will be available to all templates.
func RegisterPartials(partials map[string]string, unescaped bool) {
	for name, p := range partials {
		RegisterPartial(name, p, unescaped)
	}
}

// RegisterPartialTemplate registers a global partial with given parsed template. That partial will be available to all templates.
func RegisterPartialTemplate(name string, tpl *Template) {
	partialsMutex.Lock()
	defer partialsMutex.Unlock()

	if partials[name] != nil {
		panic(fmt.Errorf("Partial already registered: %s", name))
	}

	partials[name] = newPartial(name, "", tpl)
}

// findPartial finds a registered global partial
func findPartial(name string) *partial {
	partialsMutex.RLock()
	defer partialsMutex.RUnlock()

	return partials[name]
}

// template returns parsed partial template
func (p *partial) template() (*Template, error) {
	if p.tpl == nil {
		var err error

		p.tpl, err = ParseTemplate(p.source, p.unescaped)
		if err != nil {
			return nil, err
		}
	}

	return p.tpl, nil
}
