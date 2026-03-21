package meshstate

import "sync"

type Node struct {
	Num       int64  `json:"num"`
	ID        string `json:"id,omitempty"`
	LongName  string `json:"long_name,omitempty"`
	ShortName string `json:"short_name,omitempty"`
	LastSeen  string `json:"last_seen,omitempty"`
	GatewayID string `json:"gateway_id,omitempty"`
}

type Snapshot struct {
	Nodes    []Node `json:"nodes"`
	Messages int    `json:"messages"`
}

type State struct {
	mu       sync.RWMutex
	nodes    map[int64]Node
	messages int
}

func New() *State                  { return &State{nodes: map[int64]Node{}} }
func (s *State) UpsertNode(n Node) { s.mu.Lock(); defer s.mu.Unlock(); s.nodes[n.Num] = n }
func (s *State) IncMessages()      { s.mu.Lock(); defer s.mu.Unlock(); s.messages++ }
func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := Snapshot{Nodes: make([]Node, 0, len(s.nodes)), Messages: s.messages}
	for _, n := range s.nodes {
		out.Nodes = append(out.Nodes, n)
	}
	return out
}
func (s *State) MeshActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.nodes) > 0 || s.messages > 0
}
