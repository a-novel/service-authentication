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
// Algorithm: topological sort over the inheritance graph; nodes are processed
// in order of depth (mods with no parents first), and each resolved mod's
// inherited deps are appended in resolution order.
//
// Inlined from github.com/a-novel-kit/golib/deps after that package was
// deprecated for single-consumer-in-org. See manage-versions ("Step 0 —
// grep consumers first") for the bar; this consumer was below it.
func ResolveDependants[Mod comparable, Deps any](mods map[Mod][]Deps, deps map[Mod][]Mod) (map[Mod][]Deps, error) {
	// Reference: https://dnaeon.github.io/dependency-graph-resolution-algorithm-in-go/
	// Convert dependencies to a map. The algorithm performs better using maps behavior.
	depsGraph := map[Mod]map[Mod]bool{}
	for mod, localDeps := range deps {
		depsGraph[mod] = map[Mod]bool{}

		for _, dep := range localDeps {
			depsGraph[mod][dep] = true
		}
	}

	// To resolve every dependency, we first need to triage the mods regarding their depths: a mod without
	// dependencies has a depth of 0, a mod with only dependencies of depth 0 has a depth of 1, and so on.
	// Once we solved this triage, we can process mods in order, fully processing every mod of depth n before
	// moving to depth n+1.
	var resolvedMods []Mod

	// We're going to resolve dependencies in rounds, unwrapping a single dependency depth at a time. Once the map
	// of dependencies ie empty, this means we resolved everything, and can return the result.
	for len(depsGraph) > 0 {
		// A given depth n+1 must resolve to at least one node of depth n (because each level of depth depends on
		// the previous one). If we can't find any node of depth n, then we have a circular dependency.
		hasResolved := false

		for mod, dependencies := range depsGraph {
			if len(dependencies) > 0 {
				continue
			}

			hasResolved = true

			// A resolved node can be removed from the original graph, and added to the resolved graph.
			delete(depsGraph, mod)
			resolvedMods = append(resolvedMods, mod)

			// Remove the resolve dependencies from the dependants of other nodes in the graph.
			for _, dependantMod := range depsGraph {
				delete(dependantMod, mod)
			}
		}

		if !hasResolved {
			return nil, fmt.Errorf("%w: %v", ErrCircularDependency, printDepsGraph(depsGraph))
		}
	}

	// Now every mod is resolved, we can build the final map.
	resolved := map[Mod][]Deps{}

	for _, mod := range resolvedMods {
		resolved[mod] = mods[mod]

		// Because we resolve mods with the lowest depth first, we know that every dependency has been fully resolved
		// when we reach a certain mod.
		for _, dep := range deps[mod] {
			resolved[mod] = append(resolved[mod], resolved[dep]...)
		}
	}

	return resolved, nil
}
