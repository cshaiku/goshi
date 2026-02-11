package protocol

import "strings"

func BuildFilenamePrompt(files []string) string {
	var b strings.Builder

	b.WriteString("You will be given a list of filenames only.\n")
	b.WriteString("You do not have access to file contents.\n")
	b.WriteString("You must explicitly request files you want to read.\n")
	b.WriteString("Respond in JSON only.\n\n")
	b.WriteString("Files:\n")

	for _, f := range files {
		b.WriteString("- ")
		b.WriteString(f)
		b.WriteString("\n")
	}

	b.WriteString("\nWhich files do you want to read, and why?\n")

	return b.String()
}
