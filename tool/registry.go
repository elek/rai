package tool

import (
	"context"

	"charm.land/fantasy"
)

type ToolDef struct {
	Name        string
	Description string
	Callback    any
}

func AllTools() (res []fantasy.AgentTool) {
	git := fantasy.NewAgentTool[GitInput]("git", "Execute any git command in the local repository", func(ctx context.Context, input GitInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		res := Git(input)
		return fantasy.ToolResponse{
			Content: res,
			Type:    "text",
		}, nil
	})
	res = append(res, git)

	cat := fantasy.NewAgentTool[CatInput]("cat", "Read file content with optional offset and line limits", func(ctx context.Context, input CatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		res := Cat(input)
		return fantasy.ToolResponse{
			Content: res,
			Type:    "text",
		}, nil
	})
	res = append(res, cat)

	files := fantasy.NewAgentTool[FileListInput]("files", "List files in a directory, with options for recursive listing and pattern matching", func(ctx context.Context, input FileListInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		res := ListFiles(input)
		return fantasy.ToolResponse{
			Content: res,
			Type:    "text",
		}, nil
	})
	res = append(res, files)

	create := fantasy.NewAgentTool[CreateInput]("create", "Create a file with the specified content and path ", func(ctx context.Context, input CreateInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		res, err := Create(input)
		return fantasy.ToolResponse{
			Content: res,
			Type:    "text",
		}, err
	})
	res = append(res, create)

	insert := fantasy.NewAgentTool[InsertInput]("insert", "Insert additional content to a file from a specific line", func(ctx context.Context, input InsertInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		res, err := Insert(input)
		return fantasy.ToolResponse{
			Content: res,
			Type:    "text",
		}, err
	})
	res = append(res, insert)

	skill := fantasy.NewAgentTool[SkillInput]("skill", "List available skills or read a skill template by name", func(ctx context.Context, input SkillInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		res := Skill(input)
		return fantasy.ToolResponse{
			Content: res,
			Type:    "text",
		}, nil
	})
	res = append(res, skill)

	return res
}
