// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleClusterUnion_Find() {
	union := newQAClusterUnion(3)
	union.parent[1] = 0
	core.Println(union.Find(1))
	// Output: 0
}

func ExampleClusterUnion_Union() {
	union := newQAClusterUnion(3)
	union.Union(0, 1)
	core.Println(union.Find(0) == union.Find(1))
	// Output: true
}
