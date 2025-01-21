package ssh

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cjlapao/common-go/helper"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SshAuthorization struct {
	User       string
	Password   string
	PrivateKey string
	KeyFile    string
}

type SshClient struct {
	config *ssh.ClientConfig
	Host   string
	Port   string
	Auth   SshAuthorization
}

func NewSshClient(host, port string, auth SshAuthorization) (*SshClient, error) {
	sslClient := &SshClient{
		Host: host,
		Port: port,
		Auth: auth,
	}

	var config *ssh.ClientConfig
	switch {
	case sslClient.Auth.KeyFile != "":
		key, err := helper.ReadFromFile(sslClient.Auth.KeyFile)
		if err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		config = &ssh.ClientConfig{
			User: sslClient.Auth.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}
	case sslClient.Auth.PrivateKey != "":
		key, err := ssh.ParsePrivateKey([]byte(sslClient.Auth.PrivateKey))
		if err != nil {
			return nil, err
		}
		config = &ssh.ClientConfig{
			User: sslClient.Auth.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(key),
			},
		}
	default:
		// Connect to the remote host
		config = &ssh.ClientConfig{
			User: sslClient.Auth.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(sslClient.Auth.Password),
			},
		}
	}

	config.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	sslClient.config = config

	return sslClient, nil
}

func (c *SshClient) Connect() error {
	if c.config == nil {
		return errors.New("SSH Client not configured")
	}

	conn, err := ssh.Dial("tcp", c.BaseAddress(), c.config)
	if err != nil {
		return err
	}

	if err := conn.Close(); err != nil {
		return err
	}

	return nil
}

func (c *SshClient) BaseAddress() string {
	baseAddress := c.Host
	if c.Port != "" {
		baseAddress = fmt.Sprintf("%s:%s", baseAddress, c.Port)
	}

	return baseAddress
}

func (c *SshClient) RunCommand(command string, arguments []string) (string, error) {
	cmd := command + " " + strings.Join(arguments, " ")
	conn, err := ssh.Dial("tcp", c.BaseAddress(), c.config)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// Run the command
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), err
	}

	conn.Close()
	return string(output), nil
}

func (c *SshClient) TransferFile(localFile, remoteFile string) error {
	conn, err := ssh.Dial("tcp", c.BaseAddress(), c.config)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Open an SFTP session
	sftp, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	defer sftp.Close()

	// Clean the file path to prevent path traversal
	cleanLocalFile := filepath.Clean(localFile)
	// Open the local file to transfer
	f, err := os.Open(cleanLocalFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read the local file contents
	contents, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Create the remote file
	remoteF, err := sftp.Create(remoteFile)
	if err != nil {
		return err
	}
	defer remoteF.Close()

	// Write the local file contents to the remote file
	_, err = remoteF.Write(contents)
	if err != nil {
		return err
	}

	return nil
}

func (c *SshClient) Close() error {
	return nil
}

func (c *SshClient) Username() string {
	return c.Auth.User
}

func (c *SshClient) Password() string {
	return c.Auth.Password
}
