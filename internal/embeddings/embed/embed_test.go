package embed

import (
	"context"
	"strings"
	"testing"

	"github.com/sourcegraph/log"
	"github.com/sourcegraph/sourcegraph/internal/paths"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/internal/api"
	codeintelContext "github.com/sourcegraph/sourcegraph/internal/codeintel/context"
	"github.com/sourcegraph/sourcegraph/internal/codeintel/types"
	bgrepo "github.com/sourcegraph/sourcegraph/internal/embeddings/background/repo"
	"github.com/sourcegraph/sourcegraph/internal/embeddings/embed/client"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func mockFile(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n"))
}

func defaultSplitter(ctx context.Context, text, fileName string, splitOptions codeintelContext.SplitOptions) ([]codeintelContext.EmbeddableChunk, error) {
	return codeintelContext.SplitIntoEmbeddableChunks(text, fileName, splitOptions), nil
}

func TestEmbedRepo(t *testing.T) {
	ctx := context.Background()
	repoName := api.RepoName("repo/name")
	revision := api.CommitID("deadbeef")
	embeddingsClient := NewMockEmbeddingsClient()
	contextService := NewMockContextService()
	contextService.SplitIntoEmbeddableChunksFunc.SetDefaultHook(defaultSplitter)
	splitOptions := codeintelContext.SplitOptions{ChunkTokensThreshold: 8}
	mockFiles := map[string][]byte{
		// 2 embedding chunks (based on split options above)
		"a.go": mockFile(
			strings.Repeat("a", 32),
			"",
			strings.Repeat("b", 32),
		),
		// 2 embedding chunks
		"b.md": mockFile(
			"# "+strings.Repeat("a", 32),
			"",
			"## "+strings.Repeat("b", 32),
		),
		// 3 embedding chunks
		"c.java": mockFile(
			strings.Repeat("a", 32),
			"",
			strings.Repeat("b", 32),
			"",
			strings.Repeat("c", 32),
		),
		// Should be excluded
		"autogen.py": mockFile(
			"# "+strings.Repeat("a", 32),
			"// Do not edit",
		),
		// Should be excluded
		"lines_too_long.c": mockFile(
			strings.Repeat("a", 2049),
			strings.Repeat("b", 2049),
			strings.Repeat("c", 2049),
		),
		"not_included.jl": mockFile(
			strings.Repeat("a", 32),
			"",
			strings.Repeat("b", 32),
		),
		// Should be excluded
		"empty.rb": mockFile(""),
		// Should be excluded (binary file),
		"binary.bin": {0xFF, 0xF, 0xF, 0xF, 0xFF, 0xF, 0xF, 0xA},
	}

	mockRanks := map[string]float64{
		"a.go":             0.1,
		"b.md":             0.2,
		"c.java":           0.3,
		"autogen.py":       0.4,
		"lines_too_long.c": 0.5,
		"empty.rb":         0.6,
		"binary.bin":       0.7,
	}

	mockRepoPathRanks := types.RepoPathRanks{
		MeanRank: 0,
		Paths:    mockRanks,
	}

	reader := funcReader(func(_ context.Context, fileName string) ([]byte, error) {
		content, ok := mockFiles[fileName]
		if !ok {
			return nil, errors.Newf("file %s not found", fileName)
		}
		return content, nil
	})

	newReadLister := func(fileNames ...string) FileReadLister {
		fileEntries := make([]FileEntry, len(fileNames))
		for i, fileName := range fileNames {
			fileEntries[i] = FileEntry{Name: fileName, Size: 350}
		}
		return listReader{
			FileReader: reader,
			FileLister: staticLister(fileEntries),
		}
	}

	excludedGlobPatterns := GetDefaultExcludedFilePathPatterns()
	// include all but .jl files
	includePatterns := []string{"*.go", "*.md", "*.java", "*.py", "*.c", "*.rb", "*.bin"}
	includeGlobs := make([]*paths.GlobPattern, len(includePatterns))
	for idx, ip := range includePatterns {
		g, err := paths.Compile(ip)
		require.Nil(t, err)
		includeGlobs[idx] = g
	}

	opts := EmbedRepoOpts{
		RepoName: repoName,
		Revision: revision,
		FileFilters: FileFilters{
			ExcludePatterns:  excludedGlobPatterns,
			IncludePatterns:  includeGlobs,
			MaxFileSizeBytes: 100000,
		},
		SplitOptions:      splitOptions,
		MaxCodeEmbeddings: 100000,
		MaxTextEmbeddings: 100000,
	}

	logger := log.NoOp()
	noopReport := func(*bgrepo.EmbedRepoStats) {}

	t.Run("no files", func(t *testing.T) {
		index, _, stats, err := EmbedRepo(ctx, embeddingsClient, contextService, newReadLister(), mockRepoPathRanks, opts, logger, noopReport)
		require.NoError(t, err)
		require.Len(t, index.CodeIndex.Embeddings, 0)
		require.Len(t, index.TextIndex.Embeddings, 0)

		expectedStats := &bgrepo.EmbedRepoStats{
			CodeIndexStats: bgrepo.EmbedFilesStats{
				FilesSkipped: map[string]int{},
			},
			TextIndexStats: bgrepo.EmbedFilesStats{
				FilesSkipped: map[string]int{},
			},
		}
		require.Equal(t, expectedStats, stats)
	})

	t.Run("code files only", func(t *testing.T) {
		index, _, stats, err := EmbedRepo(ctx, embeddingsClient, contextService, newReadLister("a.go"), mockRepoPathRanks, opts, logger, noopReport)
		require.NoError(t, err)
		require.Len(t, index.TextIndex.Embeddings, 0)
		require.Len(t, index.CodeIndex.Embeddings, 6)
		require.Len(t, index.CodeIndex.RowMetadata, 2)
		require.Len(t, index.CodeIndex.Ranks, 2)

		expectedStats := &bgrepo.EmbedRepoStats{
			CodeIndexStats: bgrepo.EmbedFilesStats{
				FilesScheduled: 1,
				FilesEmbedded:  1,
				ChunksEmbedded: 2,
				BytesEmbedded:  65,
				FilesSkipped:   map[string]int{},
			},
			TextIndexStats: bgrepo.EmbedFilesStats{
				FilesSkipped: map[string]int{},
			},
		}
		// ignore durations
		require.Equal(t, expectedStats, stats)
	})

	t.Run("text files only", func(t *testing.T) {
		index, _, stats, err := EmbedRepo(ctx, embeddingsClient, contextService, newReadLister("b.md"), mockRepoPathRanks, opts, logger, noopReport)
		require.NoError(t, err)
		require.Len(t, index.CodeIndex.Embeddings, 0)
		require.Len(t, index.TextIndex.Embeddings, 6)
		require.Len(t, index.TextIndex.RowMetadata, 2)
		require.Len(t, index.TextIndex.Ranks, 2)

		expectedStats := &bgrepo.EmbedRepoStats{
			CodeIndexStats: bgrepo.EmbedFilesStats{
				FilesSkipped: map[string]int{},
			},
			TextIndexStats: bgrepo.EmbedFilesStats{
				FilesScheduled: 1,
				FilesEmbedded:  1,
				ChunksEmbedded: 2,
				BytesEmbedded:  70,
				FilesSkipped:   map[string]int{},
			},
		}
		// ignore durations
		require.Equal(t, expectedStats, stats)
	})

	t.Run("mixed code and text files", func(t *testing.T) {
		rl := newReadLister("a.go", "b.md", "c.java", "autogen.py", "empty.rb", "lines_too_long.c", "binary.bin")
		index, _, stats, err := EmbedRepo(ctx, embeddingsClient, contextService, rl, mockRepoPathRanks, opts, logger, noopReport)
		require.NoError(t, err)
		require.Len(t, index.CodeIndex.Embeddings, 15)
		require.Len(t, index.CodeIndex.RowMetadata, 5)
		require.Len(t, index.CodeIndex.Ranks, 5)
		require.Len(t, index.TextIndex.Embeddings, 6)
		require.Len(t, index.TextIndex.RowMetadata, 2)
		require.Len(t, index.TextIndex.Ranks, 2)

		expectedStats := &bgrepo.EmbedRepoStats{
			CodeIndexStats: bgrepo.EmbedFilesStats{
				FilesScheduled: 6,
				FilesEmbedded:  2,
				ChunksEmbedded: 5,
				BytesEmbedded:  163,
				FilesSkipped: map[string]int{
					"autogenerated": 1,
					"binary":        1,
					"longLine":      1,
					"small":         1,
				},
			},
			TextIndexStats: bgrepo.EmbedFilesStats{
				FilesScheduled: 1,
				FilesEmbedded:  1,
				ChunksEmbedded: 2,
				BytesEmbedded:  70,
				FilesSkipped:   map[string]int{},
			},
		}
		// ignore durations
		require.Equal(t, expectedStats, stats)
	})

	t.Run("not included files", func(t *testing.T) {
		rl := newReadLister("a.go", "b.md", "c.java", "autogen.py", "empty.rb", "lines_too_long.c", "binary.bin", "not_included.jl")
		index, _, stats, err := EmbedRepo(ctx, embeddingsClient, contextService, rl, mockRepoPathRanks, opts, logger, noopReport)
		require.NoError(t, err)
		require.Len(t, index.CodeIndex.Embeddings, 15)
		require.Len(t, index.CodeIndex.RowMetadata, 5)
		require.Len(t, index.CodeIndex.Ranks, 5)
		require.Len(t, index.TextIndex.Embeddings, 6)
		require.Len(t, index.TextIndex.RowMetadata, 2)
		require.Len(t, index.TextIndex.Ranks, 2)

		expectedStats := &bgrepo.EmbedRepoStats{
			CodeIndexStats: bgrepo.EmbedFilesStats{
				FilesScheduled: 7,
				FilesEmbedded:  2,
				ChunksEmbedded: 5,
				BytesEmbedded:  163,
				FilesSkipped: map[string]int{
					"autogenerated": 1,
					"binary":        1,
					"longLine":      1,
					"small":         1,
					"notIncluded":   1,
				},
			},
			TextIndexStats: bgrepo.EmbedFilesStats{
				FilesScheduled: 1,
				FilesEmbedded:  1,
				ChunksEmbedded: 2,
				BytesEmbedded:  70,
				FilesSkipped:   map[string]int{},
			},
		}
		// ignore durations
		require.Equal(t, expectedStats, stats)
	})

	t.Run("mixed code and text files", func(t *testing.T) {
		// 3 will be embedded, 4 will be skipped
		fileNames := []string{"a.go", "b.md", "c.java", "autogen.py", "empty.rb", "lines_too_long.c", "binary.bin"}
		rl := newReadLister(fileNames...)
		statReports := 0
		countingReporter := func(*bgrepo.EmbedRepoStats) {
			statReports++
		}
		_, _, _, err := EmbedRepo(ctx, embeddingsClient, contextService, rl, mockRepoPathRanks, opts, logger, countingReporter)
		require.NoError(t, err)
		require.Equal(t, 2, statReports, `
			Expected one update for flush. This is subject to change if the
			test changes, so a failure should be considered a notification of a
			change rather than a signal that something is wrong.
		`)
	})

	t.Run("embeddings limited", func(t *testing.T) {
		optsCopy := opts
		optsCopy.MaxCodeEmbeddings = 3
		optsCopy.MaxTextEmbeddings = 1

		rl := newReadLister("a.go", "b.md", "c.java", "autogen.py", "empty.rb", "lines_too_long.c", "binary.bin")
		index, _, _, err := EmbedRepo(ctx, embeddingsClient, contextService, rl, mockRepoPathRanks, optsCopy, logger, noopReport)
		require.NoError(t, err)

		// a.md has 2 chunks, c.java has 3 chunks
		require.Len(t, index.CodeIndex.Embeddings, index.CodeIndex.ColumnDimension*5)
		// b.md has 2 chunks
		require.Len(t, index.TextIndex.Embeddings, index.CodeIndex.ColumnDimension*2)
	})

	t.Run("misbehaving embeddings service", func(t *testing.T) {
		// We should not trust the embeddings service to return the correct number of dimensions.
		// We've had multiple issues in the past where the embeddings call succeeds, but returns
		// the wrong number of dimensions either because the model changed or because there was
		// some sort of uncaught error.
		optsCopy := opts
		optsCopy.MaxCodeEmbeddings = 3
		optsCopy.MaxTextEmbeddings = 1
		rl := newReadLister("a.go", "b.md", "c.java", "autogen.py", "empty.rb", "lines_too_long.c", "binary.bin")

		misbehavingClient := &misbehavingEmbeddingsClient{embeddingsClient, 32} // too many dimensions
		_, _, _, err := EmbedRepo(ctx, misbehavingClient, contextService, rl, mockRepoPathRanks, optsCopy, logger, noopReport)
		require.ErrorContains(t, err, "expected embeddings for batch to have length")

		misbehavingClient = &misbehavingEmbeddingsClient{embeddingsClient, 32} // too few dimensions
		_, _, _, err = EmbedRepo(ctx, misbehavingClient, contextService, rl, mockRepoPathRanks, optsCopy, logger, noopReport)
		require.ErrorContains(t, err, "expected embeddings for batch to have length")

		misbehavingClient = &misbehavingEmbeddingsClient{embeddingsClient, 0} // empty return
		_, _, _, err = EmbedRepo(ctx, misbehavingClient, contextService, rl, mockRepoPathRanks, optsCopy, logger, noopReport)
		require.ErrorContains(t, err, "expected embeddings for batch to have length")

		erroringClient := &erroringEmbeddingsClient{embeddingsClient, errors.New("whoops")} // normal error
		_, _, _, err = EmbedRepo(ctx, erroringClient, contextService, rl, mockRepoPathRanks, optsCopy, logger, noopReport)
		require.ErrorContains(t, err, "whoops")

		erroringClient = &erroringEmbeddingsClient{embeddingsClient, client.PartialError{errors.New("oopsie"), 3}} // partial error
		_, _, _, err = EmbedRepo(ctx, erroringClient, contextService, rl, mockRepoPathRanks, optsCopy, logger, noopReport)
		require.ErrorContains(t, err, "oopsie")
		require.ErrorContains(t, err, "c.java", "for a partial error, the error message should contain the file name")
	})
}

type erroringEmbeddingsClient struct {
	client.EmbeddingsClient
	err error
}

func (c *erroringEmbeddingsClient) GetEmbeddings(_ context.Context, texts []string) ([]float32, error) {
	return nil, c.err
}

type misbehavingEmbeddingsClient struct {
	client.EmbeddingsClient
	returnedDimsPerInput int
}

func (c *misbehavingEmbeddingsClient) GetEmbeddings(_ context.Context, texts []string) ([]float32, error) {
	return make([]float32, len(texts)*c.returnedDimsPerInput), nil
}

func NewMockEmbeddingsClient() client.EmbeddingsClient {
	return &mockEmbeddingsClient{}
}

type mockEmbeddingsClient struct{}

func (c *mockEmbeddingsClient) GetDimensions() (int, error) {
	return 3, nil
}

func (c *mockEmbeddingsClient) GetModelIdentifier() string {
	return "mock/some-model"
}

func (c *mockEmbeddingsClient) GetEmbeddings(_ context.Context, texts []string) ([]float32, error) {
	dimensions, err := c.GetDimensions()
	if err != nil {
		return nil, err
	}
	return make([]float32, len(texts)*dimensions), nil
}

type funcReader func(ctx context.Context, fileName string) ([]byte, error)

func (f funcReader) Read(ctx context.Context, fileName string) ([]byte, error) {
	return f(ctx, fileName)
}

type staticLister []FileEntry

func (l staticLister) List(_ context.Context) ([]FileEntry, error) {
	return l, nil
}

type listReader struct {
	FileReader
	FileLister
	FileDiffer
}
