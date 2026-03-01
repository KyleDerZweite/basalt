// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

// Registry maintains all registered engines indexed by seed type.
type Registry struct {
	engines map[SeedType][]Engine
}

// NewRegistry creates an empty engine registry.
func NewRegistry() *Registry {
	return &Registry{engines: make(map[SeedType][]Engine)}
}

// Register adds an engine to the registry for each seed type it supports.
func (r *Registry) Register(e Engine) {
	for _, st := range e.SeedTypes() {
		r.engines[st] = append(r.engines[st], e)
	}
}

// EnginesFor returns all engines that can process the given seed type.
func (r *Registry) EnginesFor(st SeedType) []Engine {
	return r.engines[st]
}
