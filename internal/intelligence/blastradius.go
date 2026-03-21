package intelligence

import (
    "fmt"
    "github.com/mel-project/mel/internal/models"
)

// EstimateBlastRadius for a set of priority items
func EstimateBlastRadius(priorities []PriorityItem, nodes []models.Node) (float64, string) {
    if len(priorities) == 0 {
        return 0.0, "Isolated: No active issues detected"
    }

    var totalNodes = float64(len(nodes))
    var affectedCount float64 = 0
    var systemic = false
    var message string

    for _, p := range priorities {
        if p.Category == "system" || p.Category == "database" || p.Category == "config" {
            systemic = true
            affectedCount = totalNodes
            break
        }
        if p.Category == "transport" {
            // Assume the transport affects some ratio of nodes if multiple transports exist
            transportName, _ := p.Metadata["resource_id"].(string)
            if transportName == "" {
                transportName, _ = p.Metadata["affected_transport"].(string)
            }
            // For now, simple heuristic: each transport failure affects at least 1 node
            // or 25% of the mesh if unspecified.
            affectedCount += totalNodes * 0.25
        }
    }

    if affectedCount > totalNodes { affectedCount = totalNodes }
    if totalNodes == 0 { affectedCount = 1 } // At least current node

    score := affectedCount / totalNodes
    if systemic { score = 1.0 }

    if score >= 0.75 {
        message = fmt.Sprintf("Systemic: Potential impact to %.0f%% of nodes and critical control paths", score*100)
    } else if score >= 0.25 {
        message = fmt.Sprintf("Segmented: Impact to %.0f%% of nodes via affected transport layers", score*100)
    } else {
        message = "Isolated: Impact confined to local components or single transport"
    }

    return score, message
}
