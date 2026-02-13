package kubetest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func SetupEnvTest(tb testing.TB) *rest.Config {
	tb.Helper()

	if testing.Short() || os.Getenv("KUBEBUILDER_ASSETS") == "" {
		tb.SkipNow()
	}

	env := &envtest.Environment{}
	cfg, err := env.Start()
	require.NoErrorf(tb, err, "start envtest: %v", err)

	tb.Cleanup(func() {
		if err := env.Stop(); err != nil {
			tb.Log("stop envtest: ", err)
		}
	})

	return cfg
}
