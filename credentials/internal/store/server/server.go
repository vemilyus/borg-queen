// Copyright (C) 2025 Alex Katlein
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"crypto/tls"
	"errors"
	"github.com/vemilyus/borg-queen/credentials/internal/store/cert"
	"github.com/vemilyus/borg-queen/credentials/internal/store/service"
	"google.golang.org/grpc"
	"net"
)

type Server struct {
	*grpc.Server
	net.Listener
}

func NewServer(state *service.State) (*Server, error) {
	server := &Server{
		Server: NewGrpcServer(state),
	}

	config := state.Config()

	var listener net.Listener
	var err error

	if state.IsProduction() {
		if config.Tls == nil {
			return nil, errors.New("TLS configuration is not set")
		}

		var certReloader *cert.X509KeyPairReloader
		certReloader, err = cert.NewX509KeyPairReloader(config.Tls.CertFile, config.Tls.KeyFile)
		if err != nil {
			return nil, errors.New("Failed to load TLS certificate: " + err.Error())
		}

		tlsConfig := &tls.Config{
			GetCertificate: certReloader.GetCertificate,
			NextProtos:     []string{"h2"},
		}

		listener, err = tls.Listen("tcp", config.ListenAddress, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", config.ListenAddress)
	}

	if err != nil {
		return nil, err
	}

	server.Listener = listener

	return server, nil
}

func (s *Server) Serve() error {
	return s.Server.Serve(s.Listener)
}

func (s *Server) Close() {
	defer func() { _ = s.Listener.Close() }()
	s.Server.GracefulStop()
}
