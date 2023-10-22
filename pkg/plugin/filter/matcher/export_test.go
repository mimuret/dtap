/*
 * Copyright (c) 2022 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package matcher

import (
	umatcher "github.com/mimuret/dnsutils/matcher"
	"github.com/prometheus/client_golang/prometheus"
)

func UpdateSet(f *Matcher, set *umatcher.MatcherSet) *Matcher {
	f.set = set
	return f
}

func GetSet(f *Matcher) *umatcher.MatcherSet {
	return f.set
}

func NewMatcher() *Matcher {
	return &Matcher{
		matchCounter:   prometheus.NewCounter(prometheus.CounterOpts{Name: "dummy_match_counter"}),
		filterdCounter: prometheus.NewCounter(prometheus.CounterOpts{Name: "dummy_filterd_counter"}),
	}
}
