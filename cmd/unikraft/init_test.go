// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

//go:build integration

package main

import (
	"os"
	"strings"
)

func init() {
	if v, ok := os.LookupEnv("UKC_TOKEN"); ok {
		testToken = v
		os.Unsetenv("UKC_TOKEN")
	}

	if v, ok := os.LookupEnv("UKC_METRO"); ok {
		testMetros = append(testMetros, v)
		os.Unsetenv("UKC_METRO")
	}
	if v, ok := os.LookupEnv("UKC_METROS"); ok {
		testMetros = append(testMetros, strings.Split(v, ",")...)
		os.Unsetenv("UKC_METROS")
	}

	testCases = append(testCases, helpTestCases...)
	testCases = append(testCases, authTestCases...)
	testCases = append(testCases, instancesTestCases...)
	testCases = append(testCases, volumesTestCases...)
	testCases = append(testCases, servicesTestCases...)
	testCases = append(testCases, certificatesTestCases...)
	testCases = append(testCases, imagesTestCases...)
	testCases = append(testCases, resourceTestCases...)
}
