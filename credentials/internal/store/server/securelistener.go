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
	"github.com/rs/zerolog/log"
	"net"
)

type secureListener struct {
	net.Listener
}

func NewSecureListener(l net.Listener) net.Listener {
	return &secureListener{Listener: l}
}

func (s *secureListener) Accept() (net.Conn, error) {
	for {
		c, err := s.Listener.Accept()
		if err != nil {
			return nil, err
		}

		remoteAddr := c.RemoteAddr().(*net.TCPAddr)
		localAddr := c.LocalAddr().(*net.TCPAddr)

		isLocal, err := isLocalHost(remoteAddr.IP, localAddr.IP)
		if err != nil {
			log.Error().Err(err).Msgf("failed to determine if remote address is local: %s", remoteAddr.IP)
			_ = c.Close()
			continue
		}

		if isLocal {
			log.Warn().Msgf("rejecting connection from remote address: %s (same host)", remoteAddr.IP)
			_ = c.Close()
			continue
		}

		return c, nil
	}
}

func isLocalHost(remoteIP, localIP net.IP) (bool, error) {
	if remoteIP.IsLoopback() {
		return true, nil
	}

	if remoteIP.Equal(localIP) {
		return true, nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.Equal(remoteIP) {
				return true, nil
			}
		}
	}

	return false, nil
}
