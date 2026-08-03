package runtimeconfig_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/runtimeconfig"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDefaultsDescribeEveryEntryPoint(t *testing.T) {
	t.Parallel()
	got := runtimeconfig.Default()
	testutil.Equals(t, got.DatabasePath, "data/mixology.db")
	testutil.Equals(t, got.Actor, "owner")
	testutil.Equals(t, got.LogLevel, "info")
	testutil.Equals(t, got.LogFormat, "text")
	testutil.Equals(t, got.MetricsAddr, ":9090")
}
