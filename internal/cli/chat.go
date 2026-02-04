package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/cshaiku/goshi/internal/app"
	"github.com/cshaiku/goshi/internal/llm"
)

// runChat runs the interactive chat loop with tool support.
func runChat(
	ctx context.Context,
	client llm.Client,
	systemPrompt string,
) error {

	session := app.NewChatSession(client, systemPrompt)

	router, err := app.NewToolRouter(".")
	if err != nil {
		return err
	}

	for {
		var input string
		fmt.Print("\nYou: ")
		_, err := fmt.Scanln(&input)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if input == "/quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		session.AppendUserMessage(input)

		stream, err := session.StreamResponse(ctx)
		if err != nil {
			return err
		}

		fmt.Print("Goshi: ")
		var assistant string

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				stream.Close()
				return err
			}

			fmt.Print(chunk)
			assistant += chunk
		}

		stream.Close()
		fmt.Println()

		session.AppendAssistantMessage(assistant)

		// Tool-call detection
		if toolMsg, ok := app.TryHandleToolCall(router, assistant); ok {
			session.AppendSystemMessage(toolMsg.Content)
		}
	}
}
