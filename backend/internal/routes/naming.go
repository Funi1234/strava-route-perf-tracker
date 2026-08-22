package routes

import "fmt"

type TownNamer interface {
	TownName(lat, lng float64) (string, error)
}

// NameRoutes assigns each route a name of the form "{Town} Route {n}",
// numbering routes within a town sequentially in the order they're given
// (Cluster returns them sorted by descending activity count, so the most-run
// route in a town becomes "Route 1"). If a route's town can't be resolved,
// it falls back to "Unknown".
func NameRoutes(clusters []Route, namer TownNamer) []Route {
	counts := make(map[string]int)
	named := make([]Route, len(clusters))
	for i, cl := range clusters {
		town := "Unknown"
		if namer != nil {
			lat, lng := cl.StartCentroid()
			if name, err := namer.TownName(lat, lng); err == nil && name != "" {
				town = name
			}
		}
		counts[town]++
		cl.Name = fmt.Sprintf("%s Route %d", town, counts[town])
		named[i] = cl
	}
	return named
}
