package config

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getTestDataDir() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return ""
	}

	cwd := filepath.Dir(filename)
	return path.Join(cwd, "testdata")
}

func TestLoadOk(t *testing.T) {
	fpath := filepath.Join(getTestDataDir(), "cfg1.yaml")
	os.Setenv("CONFIG_PATH", fpath)

	cfg, err := Load()

	assert.Nil(t, err)

	assert.Equal(t, cfg.Env, "dev")
	assert.Equal(t, cfg.StoragePath, "./storage.db")
	assert.Equal(t, cfg.Address, "localhost:5555")
	assert.Equal(t, cfg.Timeout, 8*time.Second)
	assert.Equal(t, cfg.IdleTimeout, 10*time.Second)
}

func TestLoadNotSetEnv(t *testing.T) {
	os.Unsetenv("CONFIG_PATH")
	_, err := Load()
	assert.EqualError(t, err, "CONFIG_PATH environment variable is not set")
}

func TestLoadNotEexistCfg(t *testing.T) {
	notExisten := filepath.Join(getTestDataDir(), "not_existen.yaml")
	os.Setenv("CONFIG_PATH", notExisten)

	_, err := Load()

	assert.ErrorContains(t, err, "no such file or directory")
}

func TestLoadErrorParsing(t *testing.T) {
	invalidFpath := filepath.Join(getTestDataDir(), "cfg2.yaml")
	os.Setenv("CONFIG_PATH", invalidFpath)

	_, err := Load()

	assert.ErrorContains(t, err, "cannot unmarshal")
}

func TestLoadInvalidCfg1(t *testing.T) {
	invalidFpath := filepath.Join(getTestDataDir(), "cfg3.yaml")
	os.Setenv("CONFIG_PATH", invalidFpath)

	_, err := Load()

	assert.ErrorContains(t, err, "must specify 'env' key in configuration")
}

func TestLoadInvalidCfg2(t *testing.T) {
	invalidFpath := filepath.Join(getTestDataDir(), "cfg4.yaml")
	os.Setenv("CONFIG_PATH", invalidFpath)

	_, err := Load()

	assert.ErrorContains(t, err, "unsupported 'env' value (use 'dev' or 'prod' only)")
}

func TestLoadInvalidCfg3(t *testing.T) {
	invalidFpath := filepath.Join(getTestDataDir(), "cfg5.yaml")
	os.Setenv("CONFIG_PATH", invalidFpath)

	_, err := Load()

	assert.ErrorContains(t, err, "must specify 'storage_path' key while using 'dev' env")
}
func TestLoadValidProd(t *testing.T) {
	invalidFpath := filepath.Join(getTestDataDir(), "cfg6.yaml")
	os.Setenv("CONFIG_PATH", invalidFpath)

	cfg, err := Load()
	assert.Equal(t, "db", cfg.PgHost)
	assert.Equal(t, 5432, cfg.PgPort)
	assert.Equal(t, "subscription_db", cfg.PgDbName)
	assert.Equal(t, 1, cfg.PgMaxPoolSize)
	assert.Equal(t, 3, cfg.PgConnectionAttempts)
	assert.Equal(t, 30*time.Second, cfg.PgConnectionTimeout)
	assert.NoError(t, err)
}

func TestLoadInvalidProd(t *testing.T) {
	invalidFpath := filepath.Join(getTestDataDir(), "cfg7.yaml")
	os.Setenv("CONFIG_PATH", invalidFpath)

	_, err := Load()
	assert.ErrorContains(t, err, "must specify 'pg_host' key while using 'prod' env")
}
