package experiments

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/protocol"
)

func RunFSHandshakeProbe(
	ctx context.Context,
	client *llm.Client,
	dir string,
) error {
	fmt.Fprintln(os.Stderr, "[fs-probe] ENTER RunFSHandshakeProbe")
	fmt.Fprintln(os.Stderr, "[fs-probe] time:", time.Now().Format(time.RFC3339))
	fmt.Fprintln(os.Stderr, "[fs-probe] dir:", dir)

	fmt.Fprintln(os.Stderr, "[fs-probe] listing filenames...")
	files, err := protocol.ListFilenames(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[fs-probe] ERROR listing files:", err)
		return err
	}

	fmt.Fprintf(os.Stderr, "[fs-probe] discovered %d files\n", len(files))
	for i, f := range files {
		fmt.Fprintf(os.Stderr, "[fs-probe]   file[%d]: %s\n", i, f)
	}

	fmt.Fprintln(os.Stderr, "[fs-probe] building prompt...")
	prompt := protocol.BuildFilenamePrompt(files)

	fmt.Fprintln(os.Stderr, "[fs-probe] prompt BEGIN =============================")
	fmt.Fprintln(os.Stderr, prompt)
	fmt.Fprintln(os.Stderr, "[fs-probe] prompt END ===============================")

	fmt.Fprintln(os.Stderr, "[fs-probe] opening LLM stream...")
	stream, err := client.Stream(ctx, []llm.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "[fs-probe] ERROR opening stream:", err)
		return err
	}
	fmt.Fprintln(os.Stderr, "[fs-probe] stream opened successfully")

	defer func() {
		fmt.Fprintln(os.Stderr, "[fs-probe] closing stream...")
		if err := stream.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "[fs-probe] ERROR closing stream:", err)
		} else {
			fmt.Fprintln(os.Stderr, "[fs-probe] stream closed")
		}
	}()

	var out strings.Builder
	chunkCount := 0

	fmt.Fprintln(os.Stderr, "[fs-probe] receiving stream chunks...")
	for {
		chunk, err := stream.Recv()
		if err != nil {
			fmt.Fprintln(os.Stderr, "[fs-probe] stream recv finished:", err)
			break
		}

		chunkCount++
		fmt.Fprintf(os.Stderr, "[fs-probe] chunk[%d]: %q\n", chunkCount, chunk)
		out.WriteString(chunk)
	}

	fmt.Fprintf(os.Stderr, "[fs-probe] total chunks received: %d\n", chunkCount)

	raw := out.String()
	fmt.Fprintln(os.Stderr, "[fs-probe] raw LLM output BEGIN =======================")
	fmt.Fprintln(os.Stderr, raw)
	fmt.Fprintln(os.Stderr, "[fs-probe] raw LLM output END =========================")

	fmt.Fprintln(os.Stderr, "[fs-probe] parsing LLM response...")
	req, err := protocol.ParseFileRequest(raw, files)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[fs-probe] ERROR parsing response:", err)
		return err
	}

	fmt.Fprintf(os.Stderr, "[fs-probe] parsed %d requested files\n", len(req.RequestedFiles))
	for i, r := range req.RequestedFiles {
		fmt.Fprintf(
			os.Stderr,
			"[fs-probe] request[%d]: path=%q reason=%q\n",
			i,
			r.Path,
			r.Reason,
		)
	}

	fmt.Fprintln(os.Stderr, "[fs-probe] EXIT RunFSHandshakeProbe")
	return nil
}
