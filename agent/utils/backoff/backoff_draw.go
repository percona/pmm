// Copyright (C) 2023 Percona LLC
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

// Run it with:
//   go run -tags draw backoff_draw.go

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	. "github.com/percona/pmm/agent/utils/backoff"
)

func main() {
	flag.Parse()

	rand.Seed(0)

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	const simulations = 250
	const delays = 15

	delayBaseMin := 1 * time.Second
	delayBaseMax := 30 * time.Second
	b := New(delayBaseMin, delayBaseMax)
	v := make(plotter.XYs, simulations*delays)
	for s := 0; s < simulations; s++ {
		b.Reset()
		for d := 0; d < delays; d++ {
			i := s*delays + d
			v[i].X = float64(d + 1)
			v[i].Y = b.Delay().Seconds()
		}
	}

	h, err := plotter.NewScatter(v)
	if err != nil {
		panic(err)
	}
	h.Radius = 1
	p.Add(h)

	p.Add(plotter.NewGrid())
	p.X.Min = 0
	p.X.Padding = 0
	p.Y.Min = 0
	p.Y.Padding = 0
	p.Y.Label.Text = "Seconds"
	p.Y.Max += 1.0

	ticks := []plot.Tick{{Value: 1, Label: "1"}}
	maxV := int(delayBaseMax.Seconds()) * 2
	for v := 2; v <= maxV; v++ {
		tick := plot.Tick{Value: float64(v)}
		if v%2 == 0 {
			tick.Label = fmt.Sprint(v)
		}
		ticks = append(ticks, tick)
	}
	p.Y.Tick.Marker = plot.ConstantTicks(ticks)

	name := "backoff.png"
	if err := p.Save(30*vg.Centimeter, 30*vg.Centimeter, name); err != nil {
		panic(err)
	}
	log.Printf("%s saved.", name)
}
