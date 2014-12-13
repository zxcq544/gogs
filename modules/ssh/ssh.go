// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Prototype, git client looks like do not recognize req.Reply.
package ssh

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/Unknwon/com"
	"golang.org/x/crypto/ssh"

	"github.com/gogits/gogs/modules/log"
)

func handleServerConn(keyId string, chans <-chan ssh.NewChannel) {
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		ch, reqs, err := newChan.Accept()
		if err != nil {
			log.Error(3, "Error accepting channel: %v", err)
			continue
		}

		go func(in <-chan *ssh.Request) {
			defer ch.Close()
			for req := range in {
				ok, payload := false, strings.TrimLeft(string(req.Payload), "\x00&#")
				fmt.Println("Request:", req.Type, req.WantReply, payload)
				switch req.Type {
				case "env":
					args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
					if len(args) != 2 {
						break
					}
					args[0] = strings.TrimLeft(args[0], "\x04")
					_, _, err := com.ExecCmdBytes("env", args[0]+"="+args[1])
					if err != nil {
						log.Error(3, "env: %v", err)
						ch.Stderr().Write([]byte(err.Error()))
						break
					}
					ok = true
				case "exec":
					os.Setenv("SSH_ORIGINAL_COMMAND", strings.TrimLeft(payload, "'("))
					log.Info("Payload: %v", strings.TrimLeft(payload, "'("))
					cmd := exec.Command("/Users/jiahuachen/Applications/Go/src/github.com/gogits/gogs/gogs", "serv", "key-"+keyId)
					cmd.Stdout = ch
					cmd.Stdin = ch
					cmd.Stderr = ch.Stderr()

					statusCode := make([]byte, 4)
					if err := cmd.Run(); err != nil {
						log.Error(3, "exec: %v", err)
						binary.PutVarint(statusCode, 1)
					} else {
						ok = true
						binary.PutVarint(statusCode, 0)
					}
					ch.SendRequest("exit-status", false, []byte{0, 0, 0, 2})
				}
				fmt.Println("Done:", ok)
			}
			fmt.Println("Done!!!")
		}(reqs)
	}
}

func listen(config *ssh.ServerConfig, port string) {
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		panic(err)
	}
	for {
		// Once a ServerConfig has been configured, connections can be accepted.
		conn, err := listener.Accept()
		if err != nil {
			log.Error(3, "Error accepting incoming connection: %v", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sConn, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			log.Error(3, "Error on handshaking: %v", err)
			continue
		}
		// The incoming Request channel must be serviced.
		go ssh.DiscardRequests(reqs)
		go handleServerConn(sConn.Permissions.Extensions["key-id"], chans)
	}
}

// Listen starts a SSH server listens on given port.
func Listen(port string) {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// keyCache[string(ssh.MarshalAuthorizedKey(key))] = 2
			return &ssh.Permissions{Extensions: map[string]string{"key-id": "1"}}, nil
		},
	}

	privateBytes, err := ioutil.ReadFile("/Users/jiahuachen/.ssh/id_rsa")
	if err != nil {
		panic("error loading private key")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("error parsing private key")
	}
	config.AddHostKey(private)

	go listen(config, port)
}
