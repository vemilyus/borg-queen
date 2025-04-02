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

package cert

import (
	"crypto/tls"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type X509KeyPairReloader struct {
	lock        sync.RWMutex
	certFile    string
	keyFile     string
	certificate *tls.Certificate
}

func NewX509KeyPairReloader(certFile, keyFile string) (*X509KeyPairReloader, error) {
	reloader := X509KeyPairReloader{
		lock:        sync.RWMutex{},
		certFile:    certFile,
		keyFile:     keyFile,
		certificate: nil,
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	reloader.certificate = &cert

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		for range c {
			log.Info().Msg("Received SIGHUP, reloading certificate")
			if err = reloader.reload(); err != nil {
				log.Warn().Err(err).Msg("Reloading certificate failed")
			}
		}
	}()

	return &reloader, nil
}

func (reloader *X509KeyPairReloader) reload() error {
	cert, err := tls.LoadX509KeyPair(reloader.certFile, reloader.keyFile)
	if err != nil {
		return err
	}

	reloader.lock.Lock()
	defer reloader.lock.Unlock()

	reloader.certificate = &cert

	return nil
}

func (reloader *X509KeyPairReloader) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	reloader.lock.RLock()
	defer reloader.lock.RUnlock()

	return reloader.certificate, nil
}
