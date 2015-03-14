package middleware

import (
	"fmt"
	"net/http"

	"github.com/goraft/raft"
	"github.com/gorilla/mux"
)

type RaftServer interface {
	Leader() string
	Name() string
	Peers() map[string]*raft.Peer
}

// LeaderWrite forwards incoming writes to followers
// to the leader node.
type LeaderWrite struct {
	Client *http.Client

	raftServer RaftServer
	routes     []*mux.Route
}

// NewLeadWrite creates a LeaderWrite. Provide any routes you want
// forwarded to the leader in the routes parameter. All redirected
// requests will append a `X-Follower-Redirect-For` header with the
// name of the Raft server that initiated the redirect.
func NewLeadWrite(r RaftServer, routes ...*mux.Route) *LeaderWrite {
	return &LeaderWrite{
		Client:     http.DefaultClient,
		raftServer: r,
		routes:     routes,
	}
}

// ServeHTTP proxies requests to the leader.
func (l *LeaderWrite) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !l.matches(r) {
		next(w, r)
		return
	}

	leader := l.raftServer.Leader()
	// if not leader
	if leader != l.raftServer.Name() {
		connString := l.raftServer.Peers()[leader].ConnectionString

		request, _ := http.NewRequest(r.Method, fmt.Sprintf("%s%s", connString, r.URL.Path), r.Body)
		defer r.Body.Close()

		// copy headers
		for k, vals := range r.Header {
			for _, v := range vals {
				request.Header.Add(k, v)
			}
		}

		request.Header.Add("X-Follower-Redirect-For", l.raftServer.Name())

		_, err := l.Client.Do(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	next(w, r)
}

func (l *LeaderWrite) matches(r *http.Request) bool {
	m := &mux.RouteMatch{}
	for _, route := range l.routes {
		if route.Match(r, m) {
			return true
		}
	}

	return false
}
