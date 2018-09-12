package replacerule

import (
	"testing"
)

func TestParseOneRule(t *testing.T) {
	manager := GetManager("mtime.com/(.*) gitlab.mtime-dev.com/go/$1.git")
	path := manager.Replace("mtime.com/core/auxom")
	if path != "gitlab.mtime-dev.com/go/core/auxom.git" {
		t.Errorf("error path: %s", path)
	} else {
		t.Logf("path: %s", path)
	}
}

func TestParseTwoRule(t *testing.T) {
	manager := GetManager("mtime.com/(.*) gitlab.mtime-dev.com/go/$1.git,mtime.com/core/(.*) gitlab.mtime-dev.com/go/core/$1.git;")
	path := manager.Replace("mtime.com/demo")
	if path != "gitlab.mtime-dev.com/go/demo.git" {
		t.Errorf("error path: %s", path)
	} else {
		t.Logf("path: %s", path)
	}
}
