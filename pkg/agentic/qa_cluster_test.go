// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
	"testing"
)

func TestQaCluster_ClusterUnion_Find_Good(t *testing.T) {
	union := newQAClusterUnion(3)
	union.parent[1] = 0
	root := union.Find(1)

	core.AssertEqual(t, 0, root)
	core.AssertEqual(t, 0, union.parent[1])
}

func TestQaCluster_ClusterUnion_Find_Bad(t *testing.T) {
	union := newQAClusterUnion(2)
	root := union.Find(0)

	core.AssertEqual(t, 0, root)
	core.AssertEqual(t, 0, union.parent[0])
}

func TestQaCluster_ClusterUnion_Find_Ugly(t *testing.T) {
	union := newQAClusterUnion(4)
	union.parent[3] = 2
	union.parent[2] = 1
	union.parent[1] = 0
	root := union.Find(3)

	core.AssertEqual(t, 0, root)
	core.AssertEqual(t, 0, union.parent[3])
}

func TestQaCluster_ClusterUnion_Union_Good(t *testing.T) {
	union := newQAClusterUnion(4)
	union.Union(0, 1)
	union.Union(1, 2)

	left := union.Find(0)
	right := union.Find(2)
	core.AssertEqual(t, left, right)
	core.AssertEqual(t, 3, union.size[left])
}

func TestQaCluster_ClusterUnion_Union_Bad(t *testing.T) {
	union := newQAClusterUnion(2)
	union.Union(0, 0)

	core.AssertEqual(t, 0, union.Find(0))
	core.AssertEqual(t, 1, union.size[0])
}

func TestQaCluster_ClusterUnion_Union_Ugly(t *testing.T) {
	union := newQAClusterUnion(3)
	union.Union(0, 1)
	union.Union(1, 2)
	union.Union(0, 2)

	root := union.Find(0)
	core.AssertEqual(t, root, union.Find(2))
	core.AssertEqual(t, 3, union.size[root])
}
