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

// Run it with:
//   go run -tags draw backoff_draw.go

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	. "github.com/percona/pmm-agent/supervisor"
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

	var b Backoff
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
	maxV := int(DelayBaseMax.Seconds()) * 2
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
