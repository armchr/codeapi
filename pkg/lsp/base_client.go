package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/armchr/codeapi/internal/util"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

type BaseClient struct {
	client      base.LSPClient
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	nextID      int64
	fileHolders map[string]*base.FileHolder
	pendingReqs map[int]chan *base.JSONRPCMessage
	mu          *sync.Mutex
	initialized bool
	logger      *zap.Logger
}

func NewBaseClient(command string, logger *zap.Logger, args ...string) (*BaseClient, error) {
	logger.Info("Creating new LSP client", zap.String("command", command), zap.Strings("args", args))

	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Error("Failed to create stdin pipe", zap.Error(err))
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to create stdout pipe", zap.Error(err))
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("Failed to create stderr pipe", zap.Error(err))
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	logger.Info("Starting language server process", zap.String("command", command))
	if err := cmd.Start(); err != nil {
		logger.Error("Failed to start language server", zap.String("command", command), zap.Error(err))
		return nil, fmt.Errorf("failed to start language server: %w", err)
	}
	logger.Info("Language server process started successfully", zap.Int("pid", cmd.Process.Pid))

	// Check if process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		logger.Error("Language server process exited immediately", zap.Int("exit_code", cmd.ProcessState.ExitCode()))
		return nil, fmt.Errorf("language server process exited immediately with code %d", cmd.ProcessState.ExitCode())
	}

	client := &BaseClient{
		cmd:         cmd,
		stdin:       stdin,
		stdout:      stdout,
		stderr:      stderr,
		mu:          &sync.Mutex{},
		pendingReqs: make(map[int]chan *base.JSONRPCMessage),
		fileHolders: make(map[string]*base.FileHolder),
		logger:      logger,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Start stderr monitoring
	go client.monitorStderr()

	go client.readLoop(&wg)

	// Wait for the read loop to start
	wg.Wait()

	//var lspClient base.LSPClient = client

	return client, nil
}

func (t *BaseClient) GetRootPath() string {
	t.logger.Info("error: GetRootPath not implemented in BaseClient")
	panic("GetRootPath not implemented in BaseClient")
}

func (t *BaseClient) LanguageID(uri string) string {
	t.logger.Info("error: LanguageID not implemented in BaseClient")
	panic("LanguageID not implemented in BaseClient")
}

func (t *BaseClient) TestCommand(ctx context.Context) {
	t.logger.Info("Testing command execution")

	defer t.client.Shutdown(ctx)

	if _, err := t.client.Initialize(ctx); err != nil {
		t.logger.Fatal("Failed to initialize LSP client", zap.Error(err))
	}

	FILE := "src/server.js"
	FILE_URI, _ := util.ToUri(FILE, t.client.GetRootPath())

	// Test getting document symbols
	if err := t.client.DidOpenFile(ctx, FILE_URI); err != nil {
		t.logger.Error("Failed to open file", zap.String("file", "src/server.js"), zap.Error(err))
		return
	}

	if syms, err := t.GetDocumentSymbols(ctx, FILE_URI); err != nil {
		t.logger.Error("Failed to get document symbols", zap.String("file", FILE), zap.Error(err))
	} else {
		t.logger.Debug("Document symbols retrieved", zap.String("file", FILE), zap.Any("symbols", syms))
	}

	if callers, err := t.GetCallHierarchy(ctx, FILE_URI, "createLocalServer", base.Position{
		Line:      5,
		Character: 15,
	}, true); err != nil {
		t.logger.Error("Failed to get call hierarchy", zap.String("file", FILE), zap.Error(err))
	} else {
		t.logger.Debug("Call hierarchy retrieved", zap.String("file", FILE), zap.Any("callers", callers))
	}

	if callers, err := t.GetCallHierarchy(ctx, FILE_URI, "createLocalServer", base.Position{
		Line:      5,
		Character: 15,
	}, false); err != nil {
		t.logger.Error("Failed to get call hierarchy", zap.String("file", FILE), zap.Error(err))
	} else {
		t.logger.Debug("Call hierarchy retrieved", zap.String("file", FILE), zap.Any("callers", callers))
	}
	t.logger.Info("Successfully executed test command", zap.String("file", FILE))

}

func (t *BaseClient) Shutdown(ctx context.Context) error {
	if !t.initialized {
		return nil
	}

	_, err := t.sendRequest(ctx, "shutdown", nil)
	if err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	t.initialized = false
	return nil
}

func (c *BaseClient) sendRequest(ctx context.Context, method string, params interface{}) (*base.JSONRPCMessage, error) {
	id := int(atomic.AddInt64(&c.nextID, 1))
	c.logger.Info("Sending LSP request", zap.String("method", method), zap.Int("id", id))

	req := base.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}

	respChan := make(chan *base.JSONRPCMessage, 1)
	c.mu.Lock()
	c.pendingReqs[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		oldParams := params
		if oldParams != nil {
			c.logger.Info("Something")
		}

		c.logger.Info("Done with LSP request", zap.String("method", method), zap.Int("id", id))
		delete(c.pendingReqs, id)
		c.mu.Unlock()
	}()

	if err := c.writeMessage(&req); err != nil {
		c.logger.Error("Failed to write LSP request", zap.String("method", method), zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	c.logger.Debug("Waiting for LSP response", zap.String("method", method), zap.Int("id", id))
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			c.logger.Error("LSP request returned error", zap.String("method", method), zap.Int("id", id),
				zap.Int("error_code", resp.Error.Code), zap.String("error_message", resp.Error.Message))
			return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		c.logger.Info("LSP request completed successfully", zap.String("method", method), zap.Int("id", id))
		return resp, nil
	case <-ctx.Done():
		c.logger.Warn("LSP request cancelled due to context timeout", zap.String("method", method), zap.Int("id", id), zap.Error(ctx.Err()))
		return nil, ctx.Err()
	}
}

func (c *BaseClient) SendNotification(method string, params interface{}) error {
	c.logger.Debug("Sending LSP notification", zap.String("method", method))

	req := base.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	return c.writeMessage(&req)
}

func (c *BaseClient) writeMessage(msg *base.JSONRPCMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("Failed to marshal JSON-RPC message", zap.Error(err))
		return err
	}
	c.logger.Debug("Marshalled LSP message raw", zap.String("data", string(data)))

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	c.logger.Debug("Writing LSP message", zap.String("method", msg.Method), zap.Int("content_length", len(data)))

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		c.logger.Error("Failed to write LSP message header", zap.String("method", msg.Method), zap.Error(err))
		return err
	}

	if _, err := c.stdin.Write(data); err != nil {
		c.logger.Error("Failed to write LSP message body", zap.String("method", msg.Method), zap.Error(err))
		return err
	}

	c.logger.Debug("LSP message written successfully", zap.String("method", msg.Method))
	return nil
}

func (c *BaseClient) monitorStderr() {
	c.logger.Info("Starting stderr monitoring")
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		c.logger.Warn("LSP stderr output", zap.String("line", line))
	}
	if err := scanner.Err(); err != nil {
		c.logger.Error("Error reading stderr", zap.Error(err))
	}
	c.logger.Info("Stderr monitoring ended")
}

func (c *BaseClient) readLoop(wg *sync.WaitGroup) {
	c.logger.Info("Starting LSP message read loop")
	reader := bufio.NewReader(c.stdout)
	wg.Done()

	for {
		c.logger.Debug("Attempting to read line from stdout")
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				c.logger.Warn("LSP read loop got EOF - process may have terminated")
				// Check process state
				if c.cmd.ProcessState != nil {
					c.logger.Error("Process has exited", zap.Int("exit_code", c.cmd.ProcessState.ExitCode()))
				} else {
					c.logger.Error("Process state is nil but got EOF")
				}
			} else if strings.Contains(err.Error(), "file already closed") {
				c.logger.Debug("LSP read loop terminated normally - file closed")
			} else {
				c.logger.Error("LSP read loop terminated with unexpected error", zap.Error(err))
			}
			break
		}
		c.logger.Debug("Successfully read line from stdout", zap.String("line", string(line)))

		lineStr := string(line)
		if strings.HasPrefix(lineStr, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(lineStr, "Content-Length:"))
			length, err := strconv.Atoi(lengthStr)
			if err != nil {
				c.logger.Error("Failed to parse Content-Length header", zap.String("length_str", lengthStr), zap.Error(err))
				continue
			}

			c.logger.Debug("Reading LSP message", zap.Int("content_length", length))

			// Skip any additional headers until we reach the empty line
			for {
				headerLine, _, err := reader.ReadLine()
				if err != nil {
					c.logger.Error("Failed to read header line", zap.Error(err))
					break
				}
				headerStr := string(headerLine)

				// Empty line indicates end of headers
				if len(headerStr) == 0 {
					break
				}

				// Skip Content-Type and other headers
				c.logger.Debug("Skipping additional header", zap.String("header", headerStr))
			}

			content := make([]byte, length)
			if _, err := io.ReadFull(reader, content); err != nil {
				c.logger.Error("Failed to read LSP message content", zap.Int("expected_length", length), zap.Error(err))
				continue
			}
			c.logger.Debug("Received LSP message raw", zap.String("content", string(content)))

			var msg base.JSONRPCMessage
			if err := json.Unmarshal(content, &msg); err != nil {
				c.logger.Error("Failed to unmarshal LSP message", zap.String("content", string(content)), zap.Error(err))
				continue
			}

			c.logger.Debug("Received LSP message", zap.String("method", msg.Method),
				zap.Any("id", msg.ID), zap.Bool("is_response", msg.ID != nil && msg.Method == ""))

			if msg.Method == "window/logMessage" {
				c.logger.Debug("Received LSP debug message", zap.String("response", string(content)))
			}

			if msg.ID != nil {
				c.mu.Lock()
				if ch, exists := c.pendingReqs[*msg.ID]; exists {
					select {
					case ch <- &msg:
						c.logger.Debug("Delivered response to waiting request", zap.Int("id", *msg.ID))
					default:
						c.logger.Warn("Channel full, dropping response", zap.Int("id", *msg.ID))
					}
				} else {
					c.logger.Warn("No pending request found for response", zap.Int("id", *msg.ID))
				}
				c.mu.Unlock()
			}
		}
	}
}

func (c *BaseClient) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	return nil
}

/*
func convertSymbolsToFunctions(symbols interface{}, uri string) []model.Function {
	var functions []model.Function

	switch s := symbols.(type) {
	case []interface{}:
		for _, sym := range s {
			if symMap, ok := sym.(map[string]interface{}); ok {
				if fn := convertSingleSymbol(symMap, uri); fn != nil {
					functions = append(functions, *fn)
				}

				if children, exists := symMap["children"].([]interface{}); exists {
					functions = append(functions, convertSymbolsToFunctions(children, uri)...)
				}
			}
		}
	case nil:
		// Handle null response
		return functions
	default:
		// Log unexpected response type
		fmt.Printf("Unexpected symbols type: %T, value: %v\n", symbols, symbols)
	}

	return functions
}

func convertToLocationStruct(locationMap map[string]interface{}) base.Location {
	uri, _ := locationMap["uri"].(string)
	rangeMap, _ := locationMap["range"].(map[string]interface{})
	startMap, _ := rangeMap["start"].(map[string]interface{})
	endMap, _ := rangeMap["end"].(map[string]interface{})

	startLine, _ := startMap["line"].(float64)
	startChar, _ := startMap["character"].(float64)
	endLine, _ := endMap["line"].(float64)
	endChar, _ := endMap["character"].(float64)

	return base.Location{
		URI: uri,
		Range: base.Range{
			Start: base.Position{
				Line:      int(startLine),
				Character: int(startChar),
			},
			End: base.Position{
				Line:      int(endLine),
				Character: int(endChar),
			},
		},
	}
}

func convertSingleSymbol(symMap map[string]interface{}, uri string) *model.Function {
	kind, _ := symMap["kind"].(float64)
	name, _ := symMap["name"].(string)

	// Log all symbols for debugging
	fmt.Printf("Symbol: name=%s, kind=%d (function=%d, method=%d)\n", name, int(kind), base.SymbolKindFunction, base.SymbolKindMethod)

	if int(kind) != base.SymbolKindFunction && int(kind) != base.SymbolKindMethod {
		return nil
	}

	if name == "" {
		return nil
	}

	location := convertToLocationStruct(symMap["location"].(map[string]interface{}))

	return &model.FunctionDefinition{
		Name:     name,
		Location: location,
	}
}
*/
