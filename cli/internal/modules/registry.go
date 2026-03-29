// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

// Registry maintains all registered modules.
type Registry struct {
	modules []Module
}

// NewRegistry creates an empty module registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a module to the registry.
func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// ModulesFor returns all modules that can handle the given node type.
func (r *Registry) ModulesFor(nodeType string) []Module {
	var result []Module
	for _, m := range r.modules {
		if m.CanHandle(nodeType) {
			result = append(result, m)
		}
	}
	return result
}

// All returns all registered modules.
func (r *Registry) All() []Module {
	return r.modules
}
