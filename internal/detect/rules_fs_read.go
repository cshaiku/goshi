package detect

var FSReadRules = []Rule{
	{
		Capability: CapabilityFSRead,
		Verbs: []string{
			"list",
			"show",
			"read",
			"open",
			"cat",
			"view",
			"examine",
			"print",
			"dump",
		},
		Nouns: []string{
			"file",
			"files",
			"folder",
			"folders",
			"directory",
			"directories",
			"path",
			"paths",
			"contents",
		},
		Window: 3,
	},
}

var FSWriteRules = []Rule{
	{
		Capability: CapabilityFSWrite,
		Verbs: []string{
			"write",
			"create",
			"save",
			"update",
			"edit",
			"modify",
			"patch",
			"add",
			"replace",
			"overwrite",
		},
		Nouns: []string{
			"file",
			"files",
			"path",
			"paths",
			"content",
		},
		Window: 3,
	},
}
