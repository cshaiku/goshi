package selfmodel

type Model struct {
	Model       ModelMeta        `yaml:"model"`
	Application ApplicationMeta `yaml:"application"`
	Commands    CommandsMeta    `yaml:"commands"`
	Safety      SafetyMeta      `yaml:"safety_invariants"`
	Deps        Dependencies    `yaml:"dependencies"`
	Structure   Structure       `yaml:"project_structure"`
}

type ModelMeta struct {
	ModelVersion string `yaml:"model_version"`
	Enforcement  string `yaml:"enforcement"`
}

type ApplicationMeta struct {
	Name string `yaml:"name"`
}

type CommandsMeta struct {
	Required []struct {
		Name string `yaml:"name"`
	} `yaml:"required"`
}

type SafetyMeta struct {
	DryRunDefault bool `yaml:"dry_run_default"`
}

type Dependencies struct {
	RequiredBinaries []struct {
		Name string `yaml:"name"`
	} `yaml:"required_binaries"`
}

type Structure struct {
	RequiredDirectories []string `yaml:"required_directories"`
}
