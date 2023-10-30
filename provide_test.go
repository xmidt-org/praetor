// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package praetor

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProvideSuite struct {
	suite.Suite
}

func TestProvide(t *testing.T) {
	suite.Run(t, new(ProvideSuite))
}
