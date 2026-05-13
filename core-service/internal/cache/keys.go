package cache

func TeamMembersKey(teamID string) string {
	return "team:" + teamID + ":members"
}

func AssetKey(assetID string) string {
	return "asset:" + assetID
}

func AssetACLKey(assetID string) string {
	return "asset:" + assetID + ":acl"
}
