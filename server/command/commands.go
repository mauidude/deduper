package command

import (
	"strings"

	"github.com/goraft/raft"
	"github.com/mauidude/deduper/minhash"
)

type WriteCommand struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Creates a new write command.
func NewWriteCommand(id string, value string) *WriteCommand {
	return &WriteCommand{
		ID:    id,
		Value: value,
	}
}

// The name of the command in the log.
func (c *WriteCommand) CommandName() string {
	return "write"
}

// Writes a value to a key.
func (c *WriteCommand) Apply(server raft.Server) (interface{}, error) {
	mh := server.Context().(*minhash.MinHasher)
	mh.Add(c.ID, strings.NewReader(c.Value))
	return nil, nil
}
