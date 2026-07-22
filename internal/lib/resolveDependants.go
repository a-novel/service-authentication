package lib

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// ErrCircularDependency is returned by ResolveDependants when the input map
// contains a cycle.
var ErrCircularDependency = errors.New("circular dependency detected")

func printDepsGraph[Mod comparable](deps map[Mod]map[Mod]bool) string {
	var output strings.Builder

	for mod, localDeps := range deps {
		fmt.Fprintf(&output, "\n\t%v -> %v", mod, lo.Keys(localDeps))
	}

	return output.String()
}

// ResolveDependants flattens a map of inter-module dependencies into a fully
// resolved map, where each module's entry contains both its direct dependencies
// and the transitive dependencies of every module it inherits from. Detects and
// errors on circular inheritance via ErrCircularDependency.
//
// Example: given
//
//	mods = {mod1: [dep1, dep2], mod2: [dep3]}
//	deps = {mod2: [mod1]}
//
// the result is:
//
//	{mod1: [dep1, dep2], mod2: [dep3, dep1, dep2]}
//
// Inherited dependencies come after the module's own, in resolution order.
func ResolveDependants[Mod comparable, Deps any](mods map[Mod][]Deps, deps map[Mod][]Mod) (map[Mod][]Deps, error) {
	// Seed every mod as a leaf so callers can omit empty entries from `deps`. A mod that
	// only ever appears as a parent would otherwise be missing from the graph, leaving
	// no depth-0 root and yielding a false circular dependency.
	depsGraph := map[Mod]map[Mod]bool{}
	for mod := range mods {
		depsGraph[mod] = map[Mod]bool{}
	}

	for mod, localDeps := range deps {
		if _, exists := depsGraph[mod]; !exists {
			depsGraph[mod] = map[Mod]bool{}
		}

		for _, dep := range localDeps {
			depsGraph[mod][dep] = true
		}
	}

	// Resolve in rounds of increasing depth: a mod with no remaining dependencies is at
	// the current depth, and each round strips one depth off the graph. resolvedMods
	// therefore ends up in depth order.
	var resolvedMods []Mod

	for len(depsGraph) > 0 {
		// A round that resolves nothing means every remaining mod sits on a cycle.
		hasResolved := false

		for mod, dependencies := range depsGraph {
			if len(dependencies) > 0 {
				continue
			}

			hasResolved = true

			delete(depsGraph, mod)
			resolvedMods = append(resolvedMods, mod)

			for _, dependantMod := range depsGraph {
				delete(dependantMod, mod)
			}
		}

		if !hasResolved {
			return nil, fmt.Errorf("%w: %v", ErrCircularDependency, printDepsGraph(depsGraph))
		}
	}

	resolved := map[Mod][]Deps{}

	for _, mod := range resolvedMods {
		resolved[mod] = mods[mod]

		// Depth order guarantees every dependency of mod is already fully resolved.
		for _, dep := range deps[mod] {
			resolved[mod] = append(resolved[mod], resolved[dep]...)
		}
	}

	return resolved, nil
}
