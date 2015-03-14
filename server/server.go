package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/goraft/raft"
	"github.com/gorilla/mux"
	"github.com/mauidude/deduper/minhash"
	"github.com/mauidude/deduper/server/command"
	"github.com/mauidude/deduper/server/middleware"
)

var (
	Logger = log.New(os.Stdout, "[server] ", log.LstdFlags)

	contentTypeJSON = "application/json"
)

// Server provides an HTTP interface to the deduper.
type Server struct {
	path       string
	host       string
	port       int
	name       string
	raftServer raft.Server
	router     *mux.Router
	minhasher  *minhash.MinHasher
}

// New creates a new Server.
func New(path string, host string, port int) *Server {
	s := &Server{
		path:      path,
		host:      host,
		port:      port,
		router:    mux.NewRouter(),
		minhasher: minhash.New(100, 2, 2),
	}

	// Read existing name or generate a new one.
	namePath := filepath.Join(path, "name")
	if b, err := ioutil.ReadFile(namePath); err == nil {
		s.name = string(b)
	} else {
		s.name = fmt.Sprintf("%07x", rand.Int())[0:7]
		if err = ioutil.WriteFile(namePath, []byte(s.name), 0644); err != nil {
			Logger.Fatalf("Unable to write to name file %s: %s", namePath, err.Error())
		}
	}

	return s
}

// This is a hack around Gorilla mux not providing the correct net/http
// HandleFunc() interface.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

// ListenAndServe starts the server listening on HTTP and
// connects to the given leader. If leader is an empty string
// this server will be a leader.
func (s *Server) ListenAndServe(leader string) error {
	var err error
	Logger.Printf("Initializing Raft Server: %s", s.path)

	// Initialize and start Raft server.
	transporter := raft.NewHTTPTransporter("/raft", 200*time.Millisecond)
	s.raftServer, err = raft.NewServer(s.name, s.path, transporter, nil, s.minhasher, "")
	if err != nil {
		Logger.Fatal(err)
	}

	transporter.Install(s.raftServer, s)
	s.raftServer.Start()

	if leader != "" {
		// Join to leader if specified.
		Logger.Println("Attempting to join leader:", leader)

		if !s.raftServer.IsLogEmpty() {
			Logger.Fatal("Cannot join with an existing log")
		}

		if err := s.Join(leader); err != nil {
			Logger.Fatal(err)
		}

	} else if s.raftServer.IsLogEmpty() {
		// Initialize the server by joining itself.

		Logger.Println("Initializing new cluster")

		_, err := s.raftServer.Do(&raft.DefaultJoinCommand{
			Name:             s.raftServer.Name(),
			ConnectionString: s.connectionString(),
		})
		if err != nil {
			Logger.Fatal(err)
		}

	} else {
		Logger.Println("Recovered from log")
	}

	Logger.Println("Initializing HTTP server")

	s.router.HandleFunc("/documents/similar", s.similarHandler).Methods("POST")
	s.router.HandleFunc("/join", s.joinHandler).Methods("POST")
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	route := s.router.HandleFunc("/documents/{id}", s.postHandler).Methods("POST")

	// Initialize and start HTTP server.
	httpServer := negroni.New()

	httpServer.Use(&middleware.ContentType{contentTypeJSON})
	httpServer.Use(middleware.NewLeadWrite(s.raftServer, route))

	httpServer.UseHandler(s.router)

	Logger.Println("Listening at:", s.connectionString())

	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), httpServer)
}

// Join joins to the leader of an existing cluster.
func (s *Server) Join(leader string) error {
	command := &raft.DefaultJoinCommand{
		Name:             s.raftServer.Name(),
		ConnectionString: s.connectionString(),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(command)
	resp, err := http.Post(fmt.Sprintf("http://%s/join", leader), contentTypeJSON, &b)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (s *Server) healthHandler(w http.ResponseWriter, req *http.Request) {
	type peer struct {
		ConnectionString string `json:"connection_string"`
	}

	type health struct {
		Name   string           `json:"name"`
		Peers  map[string]*peer `json:"peers"`
		Leader string           `json:"leader"`
		State  string           `json:"state"`
	}

	h := &health{
		Name:   s.raftServer.Name(),
		Peers:  make(map[string]*peer),
		Leader: s.raftServer.Leader(),
		State:  s.raftServer.State(),
	}

	for _, p := range s.raftServer.Peers() {
		peer := &peer{
			ConnectionString: p.ConnectionString,
		}

		h.Peers[p.Name] = peer
	}

	json.NewEncoder(w).Encode(h)
}

func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
	command := &raft.DefaultJoinCommand{}

	if err := json.NewDecoder(req.Body).Decode(&command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := s.raftServer.Do(command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) similarHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	threshold := .8

	t := req.URL.Query().Get("threshold")
	if t != "" {
		var err error
		if threshold, err = strconv.ParseFloat(t, 64); err != nil {
			http.Error(w, `{"errors":["threshold is not a valid float"]}`, http.StatusBadRequest)
			return
		}
	}

	if threshold > 1.0 || threshold < 0 {
		http.Error(w, `{"errors":["threshold must be between 0 and 1.0 exclusively"]}`, http.StatusBadRequest)
		return
	}

	matches := s.minhasher.FindSimilar(req.Body, threshold)

	_ = json.NewEncoder(w).Encode(matches)
}

func (s *Server) postHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	// Read the value from the POST body.
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	value := string(b)

	// Execute the command against the Raft server.
	_, err = s.raftServer.Do(command.NewWriteCommand(vars["id"], value))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// Returns the connection string.
func (s *Server) connectionString() string {
	return fmt.Sprintf("http://%s:%d", s.host, s.port)
}
