Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
// path.go
package server

import (
	"sort"
)

type PathSortIface struct {
	paths Paths
	iface sort.Interface
}

type Paths []*Path

func ClonePaths(paths Paths) Paths {
	newPaths := make(Paths, 0, len(paths))
	for i := 0; i < len(paths); i++ {
		newPaths[i] = paths[i]
	}

	return newPaths
}

func (p Paths) Len() int {
	return len(p)
}

func (p Paths) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Paths) getPaths() Paths {
	return p
}

type ByPref struct {
	Paths
}

func (b ByPref) Less(i, j int) bool {
	return b.Paths[i].Pref < b.Paths[i].Pref
}

type BySmallestAS struct {
	Paths
}

func (b BySmallestAS) Less(i, j int) bool {
	return b.Paths[i].GetNumASes() < b.Paths[i].GetNumASes()
}

type ByLowestOrigin struct {
	Paths
}

func (b ByLowestOrigin) Less(i, j int) bool {
	return b.Paths[i].GetOrigin() < b.Paths[i].GetOrigin()
}

type ByIBGPOrEBGPRoutes struct {
	Paths
}

func (b ByIBGPOrEBGPRoutes) Less(i, j int) bool {
	return true
}

type ByLowestBGPId struct {
	Paths
}

func (b ByLowestBGPId) Less(i, j int) bool {
	return b.Paths[i].GetBGPId() < b.Paths[j].GetBGPId()
}

type ByShorterClusterLen struct {
	Paths
}

func (b ByShorterClusterLen) Less(i, j int) bool {
	return b.Paths[i].GetNumClusters() < b.Paths[j].GetNumClusters()
}

type ByLowestPeerAddress struct {
	Paths
}

func (b ByLowestPeerAddress) Less(i, j int) bool {
	if b.Paths[i].NeighborConf == nil {
		return true
	} else if b.Paths[j].NeighborConf == nil {
		return false
	}

	iNetIP := b.Paths[i].NeighborConf.Neighbor.NeighborAddress
	jNetIP := b.Paths[j].NeighborConf.Neighbor.NeighborAddress

	if len(iNetIP) < len(jNetIP) {
		return true
	} else if len(jNetIP) < len(iNetIP) {
		return false
	}

	for i, val := range iNetIP {
		if val < jNetIP[i] {
			return true
		} else if val > jNetIP[i] {
			return false
		}
	}

	return false
}
