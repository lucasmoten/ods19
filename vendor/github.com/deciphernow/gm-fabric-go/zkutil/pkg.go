// Copyright 2017 Decipher Technology Studios LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zkutil

import (
	"path"
	"regexp"

	"github.com/samuel/go-zookeeper/zk"
)

// TODO: port the rest of:
// https://github.com/youtube/vitess/blob/master/go%2Fzk%2Fzkutil.go

const (
	// PermDirectory are default permissions for a node.
	PermDirectory = zk.PermAdmin | zk.PermCreate | zk.PermDelete | zk.PermRead | zk.PermWrite
	// PermFile allows a zk node to emulate file behavior by disallowing child nodes.
	PermFile = zk.PermAdmin | zk.PermRead | zk.PermWrite
)

var (
	// DefaultACLs ...
	DefaultACLs = zk.WorldACL(zk.PermRead | zk.PermCreate | zk.PermDelete | zk.PermWrite)

	// Two or more slashes
	slashes = regexp.MustCompile("//+")
)

// CreateRecursive ...
// Create a path and any pieces required, think mkdir -p.
// Intermediate znodes are always created empty.
func CreateRecursive(zconn *zk.Conn, zkPath string, value []byte, flags int32, aclv []zk.ACL) (pathCreated string, err error) {
	zkPath = slashes.ReplaceAllString(zkPath, "/")
	return createRec(zconn, zkPath, value, flags, aclv)
}

func createRec(zconn *zk.Conn, zkPath string, value []byte, flags int32, aclv []zk.ACL) (pathCreated string, err error) {
	pathCreated, err = zconn.Create(zkPath, value, flags, aclv)
	if err == zk.ErrNoNode {
		// Make sure that nodes are either "file" or "directory" to mirror file system
		// semantics.
		dirAclv := make([]zk.ACL, len(aclv))
		for i, acl := range aclv {
			dirAclv[i] = acl
			dirAclv[i].Perms = PermDirectory
		}
		// we only want the file to (potentially) be ephemeral/sequential,
		// not the parent directory.
		dirFlags := flags &^ (zk.FlagSequence | zk.FlagEphemeral)
		_, err = createRec(zconn, path.Dir(zkPath), []byte{}, dirFlags, dirAclv)
		if err != nil && err != zk.ErrNoNode {
			return "", err
		}
		pathCreated, err = zconn.Create(zkPath, value, flags, aclv)
	}
	return
}

// IsDirectory returns if this node should be treated as a directory.
func IsDirectory(aclv []zk.ACL) bool {
	for _, acl := range aclv {
		if acl.Perms != PermDirectory {
			return false
		}
	}
	return true
}
