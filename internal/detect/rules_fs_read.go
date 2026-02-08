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
