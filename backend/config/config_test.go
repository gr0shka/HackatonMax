package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"HackatonMax/config"
)

func TestLoadConfig_ReadsEnvFile(t *testing.T) {
	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	// Since .env exists in project root, it must load the key from .env instead of default "demo-key"
	assert.NotEqual(t, "demo-key", cfg.TwoGIS.APIKey)
	assert.NotEmpty(t, cfg.TwoGIS.APIKey)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	_ = os.Setenv("PORT", "9999")
	_ = os.Setenv("TWOGIS_API_KEY", "override-twogis-key")
	_ = os.Setenv("ML_SERVICE_URL", "http://ml-test:8000")
	_ = os.Setenv("OSRM_URL", "http://osrm-test:5000")
	defer func() {
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("TWOGIS_API_KEY")
		_ = os.Unsetenv("ML_SERVICE_URL")
		_ = os.Unsetenv("OSRM_URL")
	}()

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "9999", cfg.HTTP.Port)
	assert.Equal(t, "override-twogis-key", cfg.TwoGIS.APIKey)
	assert.Equal(t, "http://ml-test:8000", cfg.ML.BaseURL)
	assert.Equal(t, "http://osrm-test:5000", cfg.OSRM.BaseURL)
}
