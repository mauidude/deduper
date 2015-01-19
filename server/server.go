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
	"time"

	"github.com/goraft/raft"
	"github.com/gorilla/mux"
	"github.com/mauidude/deduper/minhash"
	"github.com/mauidude/deduper/server/command"
)

var (
	Logger = log.New(os.Stdout, "[server] ", log.LstdFlags)
)

type Server struct {
	path       string
	host       string
	port       int
	name       string
	httpServer *http.Server
	raftServer raft.Server
	router     *mux.Router
	minhasher  *minhash.MinHasher
}

// This is a hack around Gorilla mux not providing the correct net/http
// HandleFunc() interface.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

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

	// Initialize and start HTTP server.
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	s.router.HandleFunc("/documents/similar", s.readHandler).Methods("POST")
	s.router.HandleFunc("/documents/{id}", s.writeHandler).Methods("POST")
	s.router.HandleFunc("/join", s.joinHandler).Methods("POST")

	Logger.Println("Listening at:", s.connectionString())

	return s.httpServer.ListenAndServe()
}

// Joins to the leader of an existing cluster.
func (s *Server) Join(leader string) error {
	command := &raft.DefaultJoinCommand{
		Name:             s.raftServer.Name(),
		ConnectionString: s.connectionString(),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(command)
	resp, err := http.Post(fmt.Sprintf("http://%s/join", leader), "application/json", &b)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
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

func (s *Server) readHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Add("Content-Type", "application/json")

	type similarityPost struct {
		Document  string  `json:document"`
		Threshold float64 `json:threshold"`
	}

	sp := &similarityPost{}
	json.NewDecoder(req.Body).Decode(sp)

	if sp.Threshold == 0 {
		sp.Threshold = .8
	} else if sp.Threshold > 1.0 || sp.Theshold < 0 {
		http.Error(w, `{"errors":["threshold must be between 0 and 1.0 inclusively"]}`, http.StatusBadRequest)
		return
	}

	matches := s.minhasher.FindSimilar(sp.Document, sp.Threshold)

	_ = json.NewEncoder(w).Encode(matches)
}

func (s *Server) writeHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	// Read the value from the POST body.
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	leader := s.raftServer.Leader()
	// if not leader
	if leader != s.raftServer.Name() {
		connString := s.raftServer.Peers()[leader].ConnectionString
		Logger.Println("not leader, forwarding request to", connString)

		_, err := http.Post(fmt.Sprintf("%s%s", connString, req.URL.Path), req.Header.Get("Content-Type"), bytes.NewBuffer(b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

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

func New(path string, host string, port int) *Server {
	s := &Server{
		path:      path,
		host:      host,
		port:      port,
		router:    mux.NewRouter(),
		minhasher: minhash.New(100, 2, 2),
	}

	// Read existing name or generate a new one.
	if b, err := ioutil.ReadFile(filepath.Join(path, "name")); err == nil {
		s.name = string(b)
	} else {
		s.name = fmt.Sprintf("%07x", rand.Int())[0:7]
		if err = ioutil.WriteFile(filepath.Join(path, "name"), []byte(s.name), 0644); err != nil {
			panic(err)
		}
	}

	return s
}
