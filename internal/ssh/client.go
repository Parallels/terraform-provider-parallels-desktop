package ssh

import (
	"fmt"
	"io"
	"os"

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
	client *ssh.Client
	config *ssh.ClientConfig
	Host   string
	Port   string
	Auth   SshAuthorization
}

var globalSshClient *SshClient

func NewSshClient(host, port string, auth SshAuthorization) (*SshClient, error) {
	if globalSshClient != nil {
		return globalSshClient, nil
	}

	globalSshClient = &SshClient{
		Host: host,
		Port: port,
		Auth: auth,
	}

	var config *ssh.ClientConfig
	if globalSshClient.Auth.KeyFile != "" {

		key, err := helper.ReadFromFile(globalSshClient.Auth.KeyFile)
		if err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(key)
		config = &ssh.ClientConfig{
			User: globalSshClient.Auth.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}
	} else if globalSshClient.Auth.PrivateKey != "" {
		key, err := ssh.ParsePrivateKey([]byte(globalSshClient.Auth.PrivateKey))
		if err != nil {
			return nil, err
		}
		config = &ssh.ClientConfig{
			User: globalSshClient.Auth.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(key),
			},
		}
	} else {
		// Connect to the remote host
		config = &ssh.ClientConfig{
			User: globalSshClient.Auth.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(globalSshClient.Auth.Password),
			},
		}
	}

	config.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	globalSshClient.config = config

	return globalSshClient, nil
}

func (c *SshClient) Connect() error {
	if c.config == nil {
		return fmt.Errorf("SSH Client not configured")
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

func (c *SshClient) RunCommand(command string) (string, error) {
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
	output, err := session.CombinedOutput(command)
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

	// Open the local file to transfer
	f, err := os.Open(localFile)
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

	fmt.Printf("File %s transferred to %s:%s\n", localFile, c.BaseAddress(), remoteFile)

	return nil
}
