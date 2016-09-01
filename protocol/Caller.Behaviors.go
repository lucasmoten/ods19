package protocol

import "decipher.com/object-drive-server/metadata/models"

// InGroup tests a group name against the Caller's embedded slice of groups to determine
// if the Caller is a member. Group names are flattened for comparison.
func (caller Caller) InGroup(group string) bool {

	for _, grp := range caller.Groups {
		if models.AACFlatten(group) == models.AACFlatten(grp) {
			return true
		}
	}
	return false
}
