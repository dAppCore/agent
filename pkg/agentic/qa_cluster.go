// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"hash/fnv"
	"math"
	"unicode"

	core "dappco.re/go"
	poindexter "forge.lthn.ai/Snider/Poindexter"
)

const (
	// qaClusterFeatureDimensions keeps the hashed feature vector compact while
	// leaving enough buckets for message/token separation in typical QA runs.
	qaClusterFeatureDimensions = 24

	// RFC §7 uses cosine distance 0.15 for "similar enough" QA findings.
	qaClusterMinimumCosineSimilarity = 0.85
	qaClusterCosineDistanceThreshold = 1 - qaClusterMinimumCosineSimilarity
)

var qaClusterEuclideanDistanceThreshold = math.Sqrt(2 - (2 * qaClusterMinimumCosineSimilarity))

type qaClusterPoint struct {
	Index int
}

type qaClusterUnion struct {
	parent []int
	size   []int
}

// clusterFindings groups the current cycle's findings by similarity so
// `.meta/report.json` surfaces recurring shapes instead of listing every
// repeated failure individually. When Poindexter cannot build the KD-tree, the
// function falls back to the previous exact-key bucketing so reporting keeps
// working.
//
// Usage example: `clusters := clusterFindings(report.Findings)`
func clusterFindings(findings []QAFinding) []DispatchCluster {
	if len(findings) == 0 {
		return nil
	}

	points := qaClusterPoints(findings)
	cosineTree, err := poindexter.NewKDTree(points,
		poindexter.WithMetric(poindexter.CosineDistance{}),
	)
	if err != nil {
		return clusterFindingsFallback(findings)
	}

	euclideanTree, err := poindexter.NewKDTree(points,
		poindexter.WithMetric(poindexter.EuclideanDistance{}),
	)
	if err != nil {
		return clusterFindingsFallback(findings)
	}

	union := newQAClusterUnion(len(findings))
	for _, point := range points {
		qaClusterUnionRadius(union, cosineTree, point, findings, qaClusterCosineDistanceThreshold)
		qaClusterUnionRadius(union, euclideanTree, point, findings, qaClusterEuclideanDistanceThreshold)
	}

	return qaClusterDispatchClusters(findings, union)
}

func clusterFindingsFallback(findings []QAFinding) []DispatchCluster {
	byKey := make(map[string]*DispatchCluster, len(findings))
	for _, finding := range findings {
		key := core.Sprintf("%s|%s|%s|%s", finding.Tool, finding.Severity, finding.Category, firstNonEmpty(finding.Code, finding.RuleID))
		cluster, ok := byKey[key]
		if !ok {
			cluster = &DispatchCluster{
				Tool:     finding.Tool,
				Severity: finding.Severity,
				Category: finding.Category,
				RuleID:   firstNonEmpty(finding.Code, finding.RuleID),
			}
			byKey[key] = cluster
		}
		cluster.Count++
		if len(cluster.Samples) < clusterSampleLimit {
			cluster.Samples = append(cluster.Samples, DispatchClusterSample{
				File:    finding.File,
				Line:    finding.Line,
				Message: finding.Message,
			})
		}
	}

	clusters := make([]DispatchCluster, 0, len(byKey))
	for _, cluster := range byKey {
		clusters = append(clusters, *cluster)
	}
	sortDispatchClusters(clusters)
	return clusters
}

func qaClusterPoints(findings []QAFinding) []poindexter.KDPoint[qaClusterPoint] {
	points := make([]poindexter.KDPoint[qaClusterPoint], len(findings))
	for index, finding := range findings {
		points[index] = poindexter.KDPoint[qaClusterPoint]{
			ID:     core.Sprintf("finding-%d", index),
			Coords: qaClusterFeatureVector(finding),
			Value:  qaClusterPoint{Index: index},
		}
	}
	return points
}

func qaClusterFeatureVector(finding QAFinding) []float64 {
	coords := make([]float64, qaClusterFeatureDimensions)

	qaClusterAddToken(coords, core.Concat("tool:", core.Lower(finding.Tool)), 4)
	qaClusterAddToken(coords, core.Concat("severity:", core.Lower(finding.Severity)), 3)
	qaClusterAddToken(coords, core.Concat("category:", core.Lower(finding.Category)), 2)
	qaClusterAddToken(coords, core.Concat("rule:", core.Lower(firstNonEmpty(finding.Code, finding.RuleID))), 2)
	qaClusterAddText(coords, finding.Title, 1.5)
	qaClusterAddText(coords, finding.Message, 1)

	if qaClusterVectorZero(coords) {
		qaClusterAddToken(coords, "finding", 1)
	}

	qaClusterNormalise(coords)
	return coords
}

func qaClusterAddText(coords []float64, text string, weight float64) {
	tokens := qaClusterTokens(text)
	for _, token := range tokens {
		qaClusterAddToken(coords, token, weight)
	}
	for index := 1; index < len(tokens); index++ {
		qaClusterAddToken(coords, core.Concat(tokens[index-1], "_", tokens[index]), weight+0.25)
	}
}

func qaClusterTokens(text string) []string {
	if core.Trim(text) == "" {
		return nil
	}

	buffer := make([]rune, 0, len(text))
	for _, value := range text {
		switch {
		case unicode.IsLetter(value), unicode.IsDigit(value):
			buffer = append(buffer, core.ToLower(value))
		default:
			buffer = append(buffer, ' ')
		}
	}

	parts := core.Split(string(buffer), " ")
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		part = core.Trim(part)
		if len(part) < 3 || qaClusterStopWord(part) {
			continue
		}
		tokens = append(tokens, part)
	}
	return tokens
}

func qaClusterStopWord(value string) bool {
	switch value {
	case "the", "and", "for", "with", "that", "this", "from", "into", "your", "you", "are", "was", "were", "has", "have", "had", "not", "but", "can", "could", "should", "would":
		return true
	default:
		return false
	}
}

func qaClusterAddToken(coords []float64, token string, weight float64) {
	if token == "" || weight == 0 {
		return
	}
	coords[qaClusterBucket(token)] += weight
}

func qaClusterBucket(token string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(token))
	return int(hash.Sum32() % qaClusterFeatureDimensions)
}

func qaClusterVectorZero(coords []float64) bool {
	for _, value := range coords {
		if value != 0 {
			return false
		}
	}
	return true
}

func qaClusterNormalise(coords []float64) {
	var sum float64
	for _, value := range coords {
		sum += value * value
	}
	if sum == 0 {
		return
	}

	length := math.Sqrt(sum)
	for index := range coords {
		coords[index] /= length
	}
}

func qaClusterUnionRadius(union *qaClusterUnion, tree *poindexter.KDTree[qaClusterPoint], point poindexter.KDPoint[qaClusterPoint], findings []QAFinding, threshold float64) {
	if union == nil || tree == nil {
		return
	}

	neighbours, _ := tree.Radius(point.Coords, threshold)
	for _, neighbour := range neighbours {
		leftIndex := point.Value.Index
		rightIndex := neighbour.Value.Index
		if leftIndex == rightIndex {
			continue
		}
		if !qaClusterCompatible(findings[leftIndex], findings[rightIndex]) {
			continue
		}
		union.Union(leftIndex, rightIndex)
	}
}

func qaClusterCompatible(left, right QAFinding) bool {
	if left.Tool != "" && right.Tool != "" && left.Tool != right.Tool {
		return false
	}
	if left.Severity != "" && right.Severity != "" && left.Severity != right.Severity {
		return false
	}
	return true
}

func newQAClusterUnion(size int) *qaClusterUnion {
	parent := make([]int, size)
	setSize := make([]int, size)
	for index := range parent {
		parent[index] = index
		setSize[index] = 1
	}
	return &qaClusterUnion{
		parent: parent,
		size:   setSize,
	}
}

func (union *qaClusterUnion) Find(index int) int {
	if union.parent[index] != index {
		union.parent[index] = union.Find(union.parent[index])
	}
	return union.parent[index]
}

func (union *qaClusterUnion) Union(left, right int) {
	leftRoot := union.Find(left)
	rightRoot := union.Find(right)
	if leftRoot == rightRoot {
		return
	}
	if union.size[leftRoot] < union.size[rightRoot] {
		leftRoot, rightRoot = rightRoot, leftRoot
	}
	union.parent[rightRoot] = leftRoot
	union.size[leftRoot] += union.size[rightRoot]
}

func qaClusterDispatchClusters(findings []QAFinding, union *qaClusterUnion) []DispatchCluster {
	byRoot := make(map[int][]int, len(findings))
	for index := range findings {
		root := union.Find(index)
		byRoot[root] = append(byRoot[root], index)
	}

	clusters := make([]DispatchCluster, 0, len(byRoot))
	for _, members := range byRoot {
		clusters = append(clusters, qaClusterSummary(findings, members))
	}
	sortDispatchClusters(clusters)
	return clusters
}

func qaClusterSummary(findings []QAFinding, members []int) DispatchCluster {
	cluster := DispatchCluster{
		Tool:     qaClusterDominantValue(findings, members, func(finding QAFinding) string { return finding.Tool }),
		Severity: qaClusterDominantValue(findings, members, func(finding QAFinding) string { return finding.Severity }),
		Category: qaClusterDominantValue(findings, members, func(finding QAFinding) string { return finding.Category }),
		RuleID: qaClusterDominantValue(findings, members, func(finding QAFinding) string {
			return firstNonEmpty(finding.Code, finding.RuleID)
		}),
		Count: len(members),
	}

	for _, index := range members {
		finding := findings[index]
		if len(cluster.Samples) >= clusterSampleLimit {
			break
		}
		cluster.Samples = append(cluster.Samples, DispatchClusterSample{
			File:    finding.File,
			Line:    finding.Line,
			Message: finding.Message,
		})
	}

	return cluster
}

func qaClusterDominantValue(findings []QAFinding, members []int, extract func(QAFinding) string) string {
	counts := make(map[string]int, len(members))
	bestValue := ""
	bestCount := 0
	for _, index := range members {
		value := extract(findings[index])
		if value == "" {
			continue
		}
		counts[value]++
		if counts[value] > bestCount || (counts[value] == bestCount && (bestValue == "" || value < bestValue)) {
			bestValue = value
			bestCount = counts[value]
		}
	}
	return bestValue
}

// sortDispatchClusters orders clusters by descending Count then ascending
// RuleID so the report is deterministic across runs and `core-agent status`
// always shows the same ordering for identical data.
func sortDispatchClusters(clusters []DispatchCluster) {
	for i := 1; i < len(clusters); i++ {
		candidate := clusters[i]
		j := i - 1
		for j >= 0 && clusterLess(candidate, clusters[j]) {
			clusters[j+1] = clusters[j]
			j--
		}
		clusters[j+1] = candidate
	}
}

func clusterLess(left, right DispatchCluster) bool {
	if left.Count != right.Count {
		return left.Count > right.Count
	}
	if left.Tool != right.Tool {
		return left.Tool < right.Tool
	}
	return left.RuleID < right.RuleID
}
