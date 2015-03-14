package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goraft/raft"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderWrite_Follower(t *testing.T) {
	// start "leader"
	leaderHandler := &mockHandler{}
	leaderListener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go http.Serve(leaderListener, leaderHandler)

	followerName := "jake_the_dog"
	leaderName := "finn_the_human"

	body := strings.NewReader("hey girl, hey")
	r, _ := http.NewRequest("POST", "/forward", body)
	r.Header.Add("Some-Header", "Some-Value")
	rw := httptest.NewRecorder()
	raftServer := &mockRaftServer{
		name:   followerName,
		leader: leaderName,
		peers: map[string]*raft.Peer{
			leaderName: &raft.Peer{
				ConnectionString: fmt.Sprintf("http://localhost:%d", leaderListener.Addr().(*net.TCPAddr).Port),
			},
		},
	}

	handler := &mockHandler{}
	route := mux.NewRouter().HandleFunc("/forward", handler.ServeHTTP).Methods("POST")
	lw := NewLeadWrite(raftServer, route)

	next := &mockHandler{}
	lw.ServeHTTP(rw, r, next.ServeHTTP)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "hey girl, hey", leaderHandler.body.String())

	assert.False(t, next.called)
	assert.Equal(t, "Some-Value", leaderHandler.r.Header.Get("Some-Header"))
	assert.Equal(t, followerName, leaderHandler.r.Header.Get("X-Follower-Redirect-For"))
}

func TestLeaderWrite_Leader(t *testing.T) {
	// start "leader"
	leaderHandler := &mockHandler{}
	leaderListener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go http.Serve(leaderListener, leaderHandler)

	followerName := "jake_the_dog"
	leaderName := "jake_the_dog"

	r, _ := http.NewRequest("POST", "/forward", nil)
	rw := httptest.NewRecorder()
	raftServer := &mockRaftServer{
		name:   followerName,
		leader: leaderName,
	}

	handler := &mockHandler{}
	route := mux.NewRouter().HandleFunc("/forward", handler.ServeHTTP).Methods("POST")
	lw := NewLeadWrite(raftServer, route)

	next := &mockHandler{}
	lw.ServeHTTP(rw, r, next.ServeHTTP)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.True(t, next.called)
}

func TestLeaderWrite_Follower_NoForward(t *testing.T) {
	// start "leader"
	leaderHandler := &mockHandler{}
	leaderListener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go http.Serve(leaderListener, leaderHandler)

	followerName := "jake_the_dog"
	leaderName := "finn_the_human"

	r, _ := http.NewRequest("POST", "/no_forward", nil)
	rw := httptest.NewRecorder()
	raftServer := &mockRaftServer{
		name:   followerName,
		leader: leaderName,
	}

	handler := &mockHandler{}
	route := mux.NewRouter().HandleFunc("/forward", handler.ServeHTTP).Methods("POST")
	lw := NewLeadWrite(raftServer, route)

	next := &mockHandler{}
	lw.ServeHTTP(rw, r, next.ServeHTTP)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.True(t, next.called)
	assert.False(t, leaderHandler.called)
}

type mockHandler struct {
	called bool
	rw     http.ResponseWriter
	r      *http.Request
	body   *bytes.Buffer
}

func (m *mockHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	m.called = true
	m.rw = rw
	m.r = r

	if r.Body != nil {
		defer r.Body.Close()

		m.body = &bytes.Buffer{}
		io.Copy(m.body, r.Body)
	}
}

type mockRaftServer struct {
	name   string
	leader string
	peers  map[string]*raft.Peer
}

func (m *mockRaftServer) Name() string {
	return m.name
}

func (m *mockRaftServer) Leader() string {
	return m.leader
}

func (m *mockRaftServer) Peers() map[string]*raft.Peer {
	return m.peers
}
