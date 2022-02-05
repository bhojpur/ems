package quantile

// Copyright (c) 2018 Bhojpur Consulting Private Limited, India. All rights reserved.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

import (
	"encoding/json"
	"math"
	"sort"
)

type E2eProcessingLatencyAggregate struct {
	Count       int                  `json:"count"`
	Percentiles []map[string]float64 `json:"percentiles"`
	Topic       string               `json:"topic"`
	Channel     string               `json:"channel"`
	Addr        string               `json:"host"`
}

func (e *E2eProcessingLatencyAggregate) UnmarshalJSON(b []byte) error {
	var resp struct {
		Count       int                  `json:"count"`
		Percentiles []map[string]float64 `json:"percentiles"`
		Topic       string               `json:"topic"`
		Channel     string               `json:"channel"`
		Addr        string               `json:"host"`
	}
	err := json.Unmarshal(b, &resp)
	if err != nil {
		return err
	}

	for _, p := range resp.Percentiles {
		p["min"] = p["value"]
		p["max"] = p["value"]
		p["average"] = p["value"]
		p["count"] = float64(resp.Count)
	}

	e.Count = resp.Count
	e.Percentiles = resp.Percentiles
	e.Topic = resp.Topic
	e.Channel = resp.Channel
	e.Addr = resp.Addr

	return nil
}

func (e *E2eProcessingLatencyAggregate) Len() int { return len(e.Percentiles) }
func (e *E2eProcessingLatencyAggregate) Swap(i, j int) {
	e.Percentiles[i], e.Percentiles[j] = e.Percentiles[j], e.Percentiles[i]
}
func (e *E2eProcessingLatencyAggregate) Less(i, j int) bool {
	return e.Percentiles[i]["percentile"] > e.Percentiles[j]["percentile"]
}

// Add merges e2 into e by averaging the percentiles
func (e *E2eProcessingLatencyAggregate) Add(e2 *E2eProcessingLatencyAggregate) {
	e.Addr = "*"
	p := e.Percentiles
	e.Count += e2.Count
	for _, value := range e2.Percentiles {
		i := -1
		for j, v := range p {
			if value["quantile"] == v["quantile"] {
				i = j
				break
			}
		}
		if i == -1 {
			i = len(p)
			e.Percentiles = append(p, make(map[string]float64))
			p = e.Percentiles
			p[i]["quantile"] = value["quantile"]
		}
		p[i]["max"] = math.Max(value["max"], p[i]["max"])
		p[i]["min"] = math.Min(value["max"], p[i]["max"])
		p[i]["count"] += value["count"]
		if p[i]["count"] == 0 {
			p[i]["average"] = 0
			continue
		}
		delta := value["average"] - p[i]["average"]
		R := delta * value["count"] / p[i]["count"]
		p[i]["average"] = p[i]["average"] + R
	}
	sort.Sort(e)
}
