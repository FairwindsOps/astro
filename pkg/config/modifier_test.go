package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ddapi "github.com/zorkian/go-datadog-api"
)

func TestModifierIgnore(t *testing.T) {
	modifiers := newModifiers()
	monitor := &ddapi.Monitor{
		Name: ddapi.String("Test"),
	}
	annotations := map[string]string{
		"ignore.astro.fairwinds.com/ignored-monitor.chk": "true",
	}
	modifiers.Run(monitor, "ignored-monitor", annotations)
	assert.Equal(t, monitor, &ddapi.Monitor{})
}

func TestModifierIgnoreGlobal(t *testing.T) {
	modifiers := newModifiers()
	monitor := &ddapi.Monitor{
		Name: ddapi.String("Test"),
	}
	annotations := map[string]string{
		"ignore.astro.fairwinds.com/global.chk": "true",
	}
	modifiers.Run(monitor, "ignored-monitor", annotations)
	assert.Equal(t, monitor, &ddapi.Monitor{})
}

func TestModifierOverride(t *testing.T) {
	modifiers := newModifiers()
	monitor := &ddapi.Monitor{
		Name: ddapi.String("Test"),
	}
	annotations := map[string]string{
		"override.astro.fairwinds.com/override-monitor.name": "foo",
	}
	modifiers.Run(monitor, "override-monitor", annotations)
	assert.Equal(t, *monitor.Name, "foo")
}

func TestModifierGlobalOverride(t *testing.T) {
	modifiers := newModifiers()
	monitor := &ddapi.Monitor{
		Name: ddapi.String("Test"),
	}
	annotations := map[string]string{
		"override.astro.fairwinds.com/global.name": "GlobalOverride",
	}
	modifiers.Run(monitor, "override-monitor", annotations)
	assert.Equal(t, *monitor.Name, "GlobalOverride")
}
