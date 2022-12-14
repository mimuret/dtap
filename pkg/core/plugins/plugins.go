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
package plugin

import (
	//filters
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/iphash"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/label"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/mask"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/matcher"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/nop"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/static"

	// input
	_ "github.com/mimuret/dtap/v2/pkg/plugin/input/file"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/input/tcp"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/input/unix"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/input/nats"

	// output
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/file"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/fluent"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/kafka"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/metrics"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/nats"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/nop"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/stdout"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/tcp"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/unix"
)
