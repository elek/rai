package tool

import "github.com/elek/rai/schema"

func AllTools() (res []schema.Tool) {
	res = append(res, schema.Tool{
		Name:        "cat",
		Description: "Read file content with optional offset and line limits",
		Callback:    Cat,
	})
	res = append(res, schema.Tool{
		Name:        "files",
		Description: "List files in a directory, with options for recursive listing and pattern matching",
		Callback:    ListFiles,
	})
	res = append(res, schema.Tool{
		Name:        "git",
		Description: "Execute any git command in the local repository",
		Callback:    Git,
	})

	return res
}
