// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ignore
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
