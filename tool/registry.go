package tool

import (
	"context"

	"github.com/elek/rai/llm"
)

func AllTools() (res []llm.Tool) {
	res = append(res, llm.NewTool[GitInput]("git", "Execute any git command in the local repository", func(ctx context.Context, input GitInput) (string, error) {
		return Git(input), nil
	}))

	res = append(res, llm.NewTool[CatInput]("cat", "Read file content with optional offset and line limits", func(ctx context.Context, input CatInput) (string, error) {
		return Cat(input), nil
	}))

	res = append(res, llm.NewTool[FileListInput]("files", "List files in a directory, with options for recursive listing and pattern matching", func(ctx context.Context, input FileListInput) (string, error) {
		return ListFiles(input), nil
	}))

	res = append(res, llm.NewTool[CreateInput]("create", "Create a file with the specified content and path ", func(ctx context.Context, input CreateInput) (string, error) {
		return Create(input)
	}))

	res = append(res, llm.NewTool[InsertInput]("insert", "Insert additional content to a file from a specific line", func(ctx context.Context, input InsertInput) (string, error) {
		return Insert(input)
	}))

	res = append(res, llm.NewTool[BashInput]("bash", "Execute any bash command", func(ctx context.Context, input BashInput) (string, error) {
		return Bash(input), nil
	}))

	res = append(res, llm.NewTool[SkillInput]("skill", SkillToolDescription(), func(ctx context.Context, input SkillInput) (string, error) {
		return Skill(input), nil
	}))

	return res
}
