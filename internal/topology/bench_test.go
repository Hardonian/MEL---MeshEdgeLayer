package topology

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkScoreNode(b *testing.B) {
	now := time.Now().UTC()
	node := Node{
		NodeNum:          1234,
		LastSeenAt:       now.Add(-5 * time.Minute).Format(time.RFC3339),
		LastDirectSeenAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		TrustClass:       TrustTrusted,
		LastSNR:          8.0,
	}
	links := make([]Link, 5)
	for i := range links {
		links[i] = Link{Observed: true, QualityScore: 0.8, Reliability: 0.8}
	}
	thresholds := DefaultStaleThresholds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScoreNode(node, links, thresholds, now)
	}
}

func BenchmarkScoreLink(b *testing.B) {
	now := time.Now().UTC()
	link := Link{
		LastObservedAt:   now.Add(-3 * time.Minute).Format(time.RFC3339),
		Observed:         true,
		SourceTrustLevel: 0.8,
		Reliability:      0.75,
		ObservationCount: 25,
	}
	thresholds := DefaultStaleThresholds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScoreLink(link, thresholds, now)
	}
}

func BenchmarkAnalyze(b *testing.B) {
	for _, size := range []int{10, 50, 200, 1000} {
		b.Run(fmt.Sprintf("nodes=%d", size), func(b *testing.B) {
			now := time.Now().UTC()
			nodes := make([]Node, size)
			for i := range nodes {
				nodes[i] = Node{
					NodeNum:          int64(i + 1),
					ShortName:        fmt.Sprintf("N%d", i),
					LastSeenAt:       now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
					LastDirectSeenAt: now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
					TrustClass:       TrustTrusted,
				}
			}
			// Create a mesh-like topology: each node connects to 2-3 neighbors
			var links []Link
			for i := 0; i < size-1; i++ {
				links = append(links, Link{
					EdgeID:           fmt.Sprintf("e%d-%d", i, i+1),
					SrcNodeNum:       int64(i + 1),
					DstNodeNum:       int64(i + 2),
					Observed:         true,
					LastObservedAt:   now.Format(time.RFC3339),
					Reliability:      0.8,
					SourceTrustLevel: 0.8,
					ObservationCount: 10,
				})
			}
			// Add some cross-links for mesh connectivity
			for i := 0; i < size-2; i += 3 {
				links = append(links, Link{
					EdgeID:           fmt.Sprintf("x%d-%d", i, i+2),
					SrcNodeNum:       int64(i + 1),
					DstNodeNum:       int64(i + 3),
					Observed:         true,
					LastObservedAt:   now.Format(time.RFC3339),
					Reliability:      0.6,
					SourceTrustLevel: 0.7,
					ObservationCount: 5,
				})
			}
			thresholds := DefaultStaleThresholds()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Analyze(nodes, links, thresholds, now)
			}
		})
	}
}

func BenchmarkBFS(b *testing.B) {
	adj := make(map[int64][]int64)
	for i := int64(0); i < 500; i++ {
		adj[i] = append(adj[i], i+1)
		adj[i+1] = append(adj[i+1], i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visited := make(map[int64]bool)
		bfs(0, adj, visited)
	}
}
