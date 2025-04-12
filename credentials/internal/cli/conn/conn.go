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

package conn

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"
)

const timeout = 100 * time.Millisecond

func CheckIfTls(storeHost string, storePort *uint16) (bool, error) {
	var actualPort uint16
	if storePort == nil {
		actualPort = 443
	} else {
		actualPort = *storePort
	}

	addr := fmt.Sprintf("%s:%d", storeHost, actualPort)
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), timeout, errors.New("timeout while testing "+addr))
	defer cancel()

	d := tls.Dialer{Config: conf}
	conn, err := d.DialContext(ctx, "tcp", addr)
	defer func() { _ = conn.Close() }()

	if err != nil {
		plainConn, plainErr := net.DialTimeout("tcp", addr, timeout)
		defer func() { _ = plainConn.Close() }()

		if plainErr != nil {
			return false, err
		}

		return false, nil
	}

	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	return true, nil
}
