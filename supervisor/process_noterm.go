// pmm-agent
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// +build ignore

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("process_noterm: ")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, unix.SIGTERM)
	for range ch {
		log.Print("got SIGTERM, ignoring.")
	}
}
