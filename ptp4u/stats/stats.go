/*
Copyright (c) Facebook, Inc. and its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package stats implements statistics collection and reporting.
It is used by server to report internal statistics, such as number of
requests and responses.
*/
package stats

import (
	"fmt"
	"strings"
	"sync"

	ptp "github.com/facebookincubator/ptp/protocol"
)

// Stats is a metric collection interface
type Stats interface {
	// Start starts a stat reporter
	// Use this for passive reporters
	Start(monitoringport int)

	// Snapshot the values so they can be reported atomically
	Snapshot()

	// Reset atomically sets all the counters to 0
	Reset()

	// IncSubscription atomically add 1 to the counter
	IncSubscription(t ptp.MessageType)

	// IncRX atomically add 1 to the counter
	IncRX(t ptp.MessageType)

	// IncTX atomically add 1 to the counter
	IncTX(t ptp.MessageType)

	// IncRXSignaling atomically add 1 to the counter
	IncRXSignaling(t ptp.MessageType)

	// IncTXSignaling atomically add 1 to the counter
	IncTXSignaling(t ptp.MessageType)

	// IncWorkerSubs atomically add 1 to the counter
	IncWorkerSubs(workerid int)

	// DecSubscription atomically removes 1 from the counter
	DecSubscription(t ptp.MessageType)

	// DecRX atomically removes 1 from the counter
	DecRX(t ptp.MessageType)

	// DecTX atomically removes 1 from the counter
	DecTX(t ptp.MessageType)

	// DecRXSignaling atomically removes 1 from the counter
	DecRXSignaling(t ptp.MessageType)

	// DecTXSignaling atomically removes 1 from the counter
	DecTXSignaling(t ptp.MessageType)

	// DecWorkerSubs atomically removes 1 from the counter
	DecWorkerSubs(workerid int)

	// SetMaxWorkerQueue atomically sets worker queue len
	SetMaxWorkerQueue(workerid int, queue int64)

	// SetMaxTXTSAttempts atomically sets number of retries for get latest TX timestamp
	SetMaxTXTSAttempts(workerid int, retries int64)

	// SetUTCOffset atomically sets the utcoffset
	SetUTCOffset(utcoffset int64)
}

// syncMapInt64 sync map of PTP messages
type syncMapInt64 struct {
	sync.Mutex
	m map[int]int64
}

// init initializes the underlying map
func (s *syncMapInt64) init() {
	s.m = make(map[int]int64)
}

// keys returns slice of keys of the underlying map
func (s *syncMapInt64) keys() []int {
	keys := make([]int, 0, len(s.m))
	s.Lock()
	for k := range s.m {
		keys = append(keys, k)
	}
	s.Unlock()
	return keys
}

// load gets the value by the key
func (s *syncMapInt64) load(key int) int64 {
	s.Lock()
	defer s.Unlock()
	return s.m[key]
}

// inc increments the counter for the given key
func (s *syncMapInt64) inc(key int) {
	s.Lock()
	s.m[key]++
	s.Unlock()
}

// dec decrements the counter for the given key
func (s *syncMapInt64) dec(key int) {
	s.Lock()
	s.m[key]--
	s.Unlock()
}

// store saves the value with the key
func (s *syncMapInt64) store(key int, value int64) {
	s.Lock()
	s.m[key] = value
	s.Unlock()
}

// copy all key-values between maps
func (s *syncMapInt64) copy(dst *syncMapInt64) {
	for _, t := range s.keys() {
		dst.store(t, s.load(t))
	}
}

// reset stats to 0
func (s *syncMapInt64) reset() {
	s.Lock()
	for t := range s.m {
		s.m[t] = 0
	}

	s.Unlock()
}

type counters struct {
	rx            syncMapInt64
	rxSignaling   syncMapInt64
	subscriptions syncMapInt64
	tx            syncMapInt64
	txSignaling   syncMapInt64
	txtsattempts  syncMapInt64
	workerQueue   syncMapInt64
	workerSubs    syncMapInt64
	utcoffset     int64
}

func (c *counters) init() {
	c.subscriptions.init()
	c.rx.init()
	c.tx.init()
	c.rxSignaling.init()
	c.txSignaling.init()
	c.workerQueue.init()
	c.workerSubs.init()
	c.txtsattempts.init()
}

func (c *counters) reset() {
	c.subscriptions.reset()
	c.rx.reset()
	c.tx.reset()
	c.rxSignaling.reset()
	c.txSignaling.reset()
	c.workerQueue.reset()
	c.workerSubs.reset()
	c.txtsattempts.reset()
	c.utcoffset = 0
}

// toMap converts counters to a map
func (c *counters) toMap() (export map[string]int64) {
	res := make(map[string]int64)

	for _, t := range c.subscriptions.keys() {
		c := c.subscriptions.load(t)
		mt := strings.ToLower(ptp.MessageType(t).String())
		res[fmt.Sprintf("subscriptions.%s", mt)] = c
	}

	for _, t := range c.rx.keys() {
		c := c.rx.load(t)
		mt := strings.ToLower(ptp.MessageType(t).String())
		res[fmt.Sprintf("rx.%s", mt)] = c
	}

	for _, t := range c.tx.keys() {
		c := c.tx.load(t)
		mt := strings.ToLower(ptp.MessageType(t).String())
		res[fmt.Sprintf("tx.%s", mt)] = c
	}

	for _, t := range c.rxSignaling.keys() {
		c := c.rxSignaling.load(t)
		mt := strings.ToLower(ptp.MessageType(t).String())
		res[fmt.Sprintf("rx.signaling.%s", mt)] = c
	}

	for _, t := range c.txSignaling.keys() {
		c := c.txSignaling.load(t)
		mt := strings.ToLower(ptp.MessageType(t).String())
		res[fmt.Sprintf("tx.signaling.%s", mt)] = c
	}

	for _, t := range c.workerQueue.keys() {
		c := c.workerQueue.load(t)
		res[fmt.Sprintf("worker.%d.queue", t)] = c
	}

	for _, t := range c.workerSubs.keys() {
		c := c.workerSubs.load(t)
		res[fmt.Sprintf("worker.%d.subscriptions", t)] = c
	}

	for _, t := range c.txtsattempts.keys() {
		c := c.txtsattempts.load(t)
		res[fmt.Sprintf("worker.%d.txtsattempts", t)] = c
	}

	res["utcoffset"] = c.utcoffset

	return res
}
