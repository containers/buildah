package sshagent

import (
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh/agent"
)

func testNewKeySource() (*Source, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &Source{
		Keys: []interface{}{k},
	}, nil
}

func testClient(path string) ([]*agent.Key, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	ac := agent.NewClient(conn)
	keys, err := ac.List()
	if err != nil {
		return nil, err
	}
	return keys, nil

}

func TestAgentServer(t *testing.T) {
	src, err := testNewKeySource()
	require.NoError(t, err)
	ag, err := NewAgentServer(src)
	require.NoError(t, err)
	sock, err := ag.Serve("")
	require.NoError(t, err)
	// Get key from agent
	keys, err := testClient(sock)
	require.NoError(t, err)
	require.Equal(t, len(keys), 1)
	require.Equal(t, keys[0].Type(), "ssh-rsa")
	// Check for proper shutdown
	err = ag.Shutdown()
	require.NoError(t, err)

	_, err = testClient(sock)
	require.Error(t, err)
}
