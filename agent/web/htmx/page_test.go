package htmx

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestHead_Robots_enable(t *testing.T) {
	t.Parallel()

	head := &Head{NoIndex: false}
	result := head.Robots()
	must.Eq(t, "all", result)
}

func TestHead_Robots_disable(t *testing.T) {
	t.Parallel()

	head := &Head{NoIndex: true}
	result := head.Robots()
	must.Eq(t, "noindex", result)
}
