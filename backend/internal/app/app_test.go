package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"

	"HackatonMax/internal/app"
)

func TestAppModule_Validate(t *testing.T) {
	err := fx.ValidateApp(app.Module)
	assert.NoError(t, err, "Fx dependency graph must be valid without missing providers or cycles")
}
