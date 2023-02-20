package yrs

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	dsn := fmt.Sprintf("file:%s/yrs.db", os.TempDir())
	_, err := New("sqlite3", dsn)
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(strings.TrimPrefix(dsn, "file:"))
	if err != nil {
		t.Error(err)
	}
}
