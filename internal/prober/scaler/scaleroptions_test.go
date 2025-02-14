// Copyright 2022 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !kind_tests

package scaler

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

var (
	timeout  = 10 * time.Millisecond
	interval = 2 * time.Millisecond
)

func TestWithDependentResourceCheckTimeout(t *testing.T) {
	g := NewWithT(t)
	opts := scalerOptions{}
	fn := withResourceCheckTimeout(timeout)
	fn(&opts)
	g.Expect(*opts.resourceCheckTimeout).To(Equal(timeout))
}

func TestWithDependentResourceCheckInterval(t *testing.T) {
	g := NewWithT(t)
	opts := scalerOptions{}
	fn := withResourceCheckInterval(interval)
	fn(&opts)
	g.Expect(*opts.resourceCheckInterval).To(Equal(interval))
}

func TestWithScaleResourceBackOff(t *testing.T) {
	g := NewWithT(t)
	opts := scalerOptions{}
	fn := withScaleResourceBackOff(interval)
	fn(&opts)
	g.Expect(*opts.scaleResourceBackOff).To(Equal(interval))
}

func TestBuildScalerOptions(t *testing.T) {
	g := NewWithT(t)
	opts := buildScalerOptions(withResourceCheckTimeout(timeout), withResourceCheckInterval(interval))
	g.Expect(*opts.resourceCheckInterval).To(Equal(interval))
	g.Expect(*opts.resourceCheckTimeout).To(Equal(timeout))
}

func TestBuildScalerOptionsShouldFillDefaultValues(t *testing.T) {
	g := NewWithT(t)
	opts := buildScalerOptions()
	g.Expect(*opts.resourceCheckInterval).To(Equal(defaultResourceCheckInterval))
	g.Expect(*opts.resourceCheckTimeout).To(Equal(defaultResourceCheckTimeout))
}
