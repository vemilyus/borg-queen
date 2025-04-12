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

package logging

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	golog "log"
	"os"
	"strings"
	"time"
)

type zerologLogger struct {
	logger zerolog.Logger
}

func (z *zerologLogger) Write(p []byte) (n int, err error) {
	output := strings.TrimSpace(string(p))

	z.logger.Debug().Msg(output)
	return len(p), nil
}

func InitLogging(prod bool) {
	logWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: prod}
	logWriter.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %5s |", i))
	}

	if prod {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	initLogging(logWriter)
}

func InitSimpleLogging() {
	infoLevel := zerolog.InfoLevel.String()
	warnLevel := zerolog.WarnLevel.String()

	logWriter := zerolog.ConsoleWriter{
		Out: os.Stderr,
		FormatLevel: func(i interface{}) string {
			if i == infoLevel {
				return " "
			} else if i == warnLevel {
				return "!"
			}

			return strings.ToUpper(fmt.Sprintf("%s:", i))
		},
		FormatTimestamp: func(i interface{}) string { return "" },
	}

	initLogging(logWriter)
}

func initLogging(logWriter zerolog.ConsoleWriter) {
	log.Logger = log.Output(logWriter)

	golog.SetFlags(0)
	golog.SetOutput(&zerologLogger{logger: log.Logger})
}
